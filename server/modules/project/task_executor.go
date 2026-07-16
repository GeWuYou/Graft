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

	"graft/server/internal/moduleapi"
)

const (
	applicationTaskOwnerType = "application"
	composeStagePrefix       = "application.compose."
	composeOutputStreams     = 2
)

type composeStageInput struct {
	WorkspacePath string   `json:"workspace_path"`
	Args          []string `json:"args"`
}

type composeStageExecutor struct {
	typeName moduleapi.StageExecutorType
	mu       sync.Mutex
	cancels  map[uint64]context.CancelFunc
}

func (e *composeStageExecutor) Type() moduleapi.StageExecutorType { return e.typeName }

func (e *composeStageExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	var input composeStageInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode compose stage input: %w", err)
	}
	if err := ensureLifecycleCommandArgs(input.Args); err != nil {
		return err
	}
	commandCtx, cancel := withComposeCommandTimeout(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	// #nosec G204 -- docker 命令及参数计划均由 Application 模块生成，不直接使用请求输入。
	command := exec.CommandContext(commandCtx, "docker", input.Args...)
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
		if err := registrar.RegisterStageExecutor(&composeStageExecutor{typeName: moduleapi.StageExecutorType(composeStagePrefix + name), cancels: make(map[uint64]context.CancelFunc)}); err != nil {
			return err
		}
	}
	return registrar.RegisterTaskOwnerAuthorizer(projectTaskOwnerAuthorizer{service: service})
}
