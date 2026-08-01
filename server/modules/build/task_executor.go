package build

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"graft/server/internal/moduleapi"
)

type dockerfileBuildExecutor struct {
	contexts moduleapi.ApplicationBuildContextResolver
	docker   moduleapi.DockerImageBuildCapability
	mu       sync.Mutex
	cancels  map[uint64]context.CancelFunc
}

func (e *dockerfileBuildExecutor) Type() moduleapi.StageExecutorType { return buildStageExecutor }

func (e *dockerfileBuildExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.contexts == nil || e.docker == nil {
		return errors.New("build executor is unavailable")
	}
	var input moduleapi.BuildTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode build task input: %w", err)
	}
	var args []moduleapi.DockerImageBuildArg
	if err := json.Unmarshal(input.BuildArgs, &args); err != nil {
		return fmt.Errorf("decode build arguments: %w", err)
	}
	contextInfo, err := e.contexts.ResolveApplicationBuildContext(ctx, input.ApplicationID)
	if err != nil {
		return err
	}
	commandCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() { e.mu.Lock(); delete(e.cancels, run.StageID()); e.mu.Unlock(); cancel() }()
	_, err = e.docker.BuildImage(commandCtx, moduleapi.DockerImageBuildInput{WorkspaceRoot: contextInfo.WorkspaceRoot, ContextPath: input.ContextPath, DockerfilePath: input.DockerfilePath, ImageRepository: input.ImageRepository, ImageTag: input.ImageTag, BuildArgs: args}, func(logCtx context.Context, entry moduleapi.TaskLogEntry) error { return run.AppendLog(logCtx, entry) })
	return err
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

func registerBuildTaskExecutor(registrar moduleapi.TaskRuntimeRegistrar, contexts moduleapi.ApplicationBuildContextResolver, docker moduleapi.DockerImageBuildCapability) error {
	if registrar == nil {
		return errors.New("build task registrar is unavailable")
	}
	return registrar.RegisterStageExecutor(&dockerfileBuildExecutor{contexts: contexts, docker: docker, cancels: make(map[uint64]context.CancelFunc)})
}
