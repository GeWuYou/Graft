package project

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"sync"

	generated "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

const (
	applicationTaskOwnerType = "application"
	composeStagePrefix       = "application.compose."
	composeOutputStreams     = 2
)

type composeStageInput struct {
	ApplicationRecordID uint64   `json:"application_record_id"`
	WorkspacePath       string   `json:"workspace_path"`
	Args                []string `json:"args"`
}

type composeStageExecutor struct {
	typeName moduleapi.StageExecutorType
	service  *Service
	mu       sync.Mutex
	cancels  map[uint64]context.CancelFunc
}

func (e *composeStageExecutor) Type() moduleapi.StageExecutorType { return e.typeName }

func (e *composeStageExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	var input composeStageInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode compose stage input: %w", err)
	}
	args, err := e.commandArgs(ctx, input)
	if err != nil {
		return err
	}
	commandCtx, cancel := withComposeCommandTimeout(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	// #nosec G204 -- docker 命令及参数计划均由 Application 模块生成，不直接使用请求输入。
	command := exec.CommandContext(commandCtx, "docker", args...)
	command.Dir = input.WorkspacePath
	command.Env = os.Environ()
	stdout, err := command.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		return err
	}
	if err := command.Start(); err != nil {
		return err
	}
	var wg sync.WaitGroup
	read := func(stream string, reader interface{ Read([]byte) (int, error) }) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		for scanner.Scan() {
			// Compose 会将正常进度写入 stderr；任务成败仍以命令退出状态为准。
			_ = run.AppendLog(ctx, moduleapi.TaskLogEntry{Stream: stream, Level: "info", Line: scanner.Text()})
		}
	}
	wg.Add(composeOutputStreams)
	go read("stdout", stdout)
	go read("stderr", stderr)
	wg.Wait()
	return command.Wait()
}

// commandArgs 在 restart 阶段开始前读取当前运行态，避免提交排队期间的状态变化冻结 Compose 子命令。
func (e *composeStageExecutor) commandArgs(ctx context.Context, input composeStageInput) ([]string, error) {
	if e.typeName != moduleapi.StageExecutorType(composeStagePrefix+"restart") {
		return input.Args, ensureLifecycleCommandArgs(input.Args)
	}
	// COMPAT(owner=Project 生命周期任务计划, cleanup=所有缺少 application_record_id 的历史 restart 阶段任务完成)
	// 旧持久化输入不包含项目记录 ID，保留其已验证的 restart 参数，避免升级后使排队任务无法执行。
	if input.ApplicationRecordID == 0 {
		return input.Args, ensureLifecycleCommandArgs(input.Args)
	}
	if e.service == nil {
		return nil, errors.New("project service is unavailable")
	}
	aggregate, err := e.service.getAggregate(ctx, input.ApplicationRecordID)
	if err != nil {
		return nil, err
	}
	args, err := lifecycleCommandArgsForRuntime(
		aggregate,
		generated.ApplicationActionResponseActionApplicationActionRestart,
		e.service.lifecycleRuntimeStatus(ctx, aggregate, generated.ApplicationActionResponseActionApplicationActionRestart),
	)
	if err != nil {
		return nil, err
	}
	return args, ensureLifecycleCommandArgs(args)
}

func (e *composeStageExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
	e.mu.Lock()
	cancel := e.cancels[run.StageID()]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

// registerProjectTaskExecutors 注册项目 Compose 阶段执行器和任务 owner 鉴权器。
// 任务运行时注册器不可用或任一注册失败时返回错误。
func registerProjectTaskExecutors(registrar moduleapi.TaskRuntimeRegistrar, service *Service) error {
	if registrar == nil {
		return errors.New("task runtime registrar is unavailable")
	}
	for _, name := range []string{"down", "pull", "build", "up", "stop", "restart", "image-prune"} {
		if err := registrar.RegisterStageExecutor(&composeStageExecutor{typeName: moduleapi.StageExecutorType(composeStagePrefix + name), service: service, cancels: make(map[uint64]context.CancelFunc)}); err != nil {
			return err
		}
	}
	return registrar.RegisterTaskOwnerAuthorizer(projectTaskOwnerAuthorizer{service: service})
}
