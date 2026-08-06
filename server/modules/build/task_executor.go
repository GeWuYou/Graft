package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"graft/server/internal/moduleapi"
	buildstore "graft/server/modules/build/store"
)

const artifactSettlementTimeout = 5 * time.Second

type dockerfileBuildExecutor struct {
	repository buildstore.Repository
	docker     moduleapi.DockerImageBuildCapability
	mu         sync.Mutex
	cancels    map[uint64]context.CancelFunc
}

func (e *dockerfileBuildExecutor) Type() moduleapi.StageExecutorType { return buildStageExecutor }

func (e *dockerfileBuildExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.repository == nil || e.docker == nil {
		return errors.New("build executor is unavailable")
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode build task input: %w", err)
	}
	if input.BuildID == "" {
		return errors.New("build task snapshot identity is missing")
	}
	contextInfo, err := e.repository.GetJobByTaskID(ctx, run.TaskID())
	if err != nil {
		return err
	}
	if contextInfo.BuildID != input.BuildID {
		return errors.New("build task snapshot identity does not match task input")
	}
	commandCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	result, err := e.docker.BuildImage(commandCtx, moduleapi.DockerImageBuildInput{WorkspaceRoot: contextInfo.WorkspaceRoot, ContextPath: contextInfo.ContextPath, DockerfilePath: contextInfo.DockerfilePath, ImageRepository: contextInfo.ImageRepository, ImageTag: contextInfo.ImageTag, BuildArgs: contextInfo.BuildArgs}, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	if err != nil {
		return err
	}
	// Docker 已成功后仍需保留短暂的结算预算，避免调用方取消丢失 Build 产物事实。
	settlementCtx, settlementCancel := context.WithTimeout(context.WithoutCancel(ctx), artifactSettlementTimeout)
	defer settlementCancel()
	return e.repository.SettleDockerArtifact(settlementCtx, run.TaskID(), result)
}

func (e *dockerfileBuildExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
	e.mu.Lock()
	cancel := e.cancels[run.StageID()]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func registerBuildTaskExecutor(registrar moduleapi.TaskRuntimeRegistrar, repository buildstore.Repository, docker moduleapi.DockerImageBuildCapability) error {
	if registrar == nil {
		return errors.New("build task registrar is unavailable")
	}
	return registrar.RegisterStageExecutor(&dockerfileBuildExecutor{repository: repository, docker: docker, cancels: make(map[uint64]context.CancelFunc)})
}
