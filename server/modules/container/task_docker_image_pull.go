package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

const (
	dockerImageTaskOwnerType = "docker_image"
	dockerImagePullTaskType  = moduleapi.TaskType("container.docker-image-pull.v1")
	dockerImagePullExecutor  = moduleapi.StageExecutorType("container.docker-image-pull.v1")
	dockerImagePullLogParts  = 3
)

type dockerImagePullTaskInput struct {
	Reference string `json:"reference"`
}

type dockerImagePullTaskExecutor struct {
	service *service
	mu      sync.Mutex
	cancels map[uint64]context.CancelFunc
}

// Type 返回 Docker 镜像拉取阶段的稳定执行器类型。
func (e *dockerImagePullTaskExecutor) Type() moduleapi.StageExecutorType {
	return dockerImagePullExecutor
}

// Execute 使用冻结的镜像引用拉取镜像，并只将 Docker 已脱敏的进度写入 Task 日志。
func (e *dockerImagePullTaskExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.service == nil {
		return errors.New("docker image pull task executor is unavailable")
	}
	var input dockerImagePullTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode docker image pull task input: %w", err)
	}
	if err := validateDockerImageReference(input.Reference); err != nil {
		return err
	}
	pullCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.cancels, run.StageID())
		e.mu.Unlock()
		cancel()
	}()
	return e.service.PullDockerImage(pullCtx, input.Reference, func(event DockerImagePullEvent) error {
		if err := run.AppendLog(ctx, moduleapi.TaskLogEntry{Stream: "stdout", Level: dockerImagePullLogLevel(event), Line: dockerImagePullLogLine(event)}); err != nil {
			return fmt.Errorf("append docker image pull task log: %w", err)
		}
		return nil
	})
}

// Cancel 取消当前阶段持有的 Docker 拉取上下文，运行时仍负责持久化最终取消状态。
func (e *dockerImagePullTaskExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
	if e == nil {
		return nil
	}
	e.mu.Lock()
	cancel := e.cancels[run.StageID()]
	e.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func dockerImagePullLogLevel(event DockerImagePullEvent) string {
	if event.Error {
		return "error"
	}
	return "info"
}

func dockerImagePullLogLine(event DockerImagePullEvent) string {
	parts := make([]string, 0, dockerImagePullLogParts)
	if status := safeDockerImageProgress(event.Status); status != "" {
		parts = append(parts, status)
	}
	if id := safeDockerImageProgress(event.ID); id != "" {
		parts = append(parts, id)
	}
	if progress := safeDockerImageProgress(event.Progress); progress != "" {
		parts = append(parts, progress)
	}
	if len(parts) == 0 {
		return "docker image pull progress"
	}
	return strings.Join(parts, " ")
}

type dockerImageTaskOwnerAuthorizer struct{ service *service }

// OwnerType 返回 Docker 镜像 Task 的资源所有者类型。
func (a dockerImageTaskOwnerAuthorizer) OwnerType() string { return dockerImageTaskOwnerType }

// AuthorizeTaskOwner 按原始镜像拉取权限约束该资源下的 Task 查看、取消和人工重试操作。
func (a dockerImageTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return moduleapi.ErrUnauthenticated
	}
	if owner.Type != dockerImageTaskOwnerType {
		return errors.New("docker image task owner type is invalid")
	}
	if err := validateDockerImageReference(owner.ID); err != nil {
		return err
	}
	permission := containercontract.DockerImagePullPermission.String()
	if action != moduleapi.TaskOwnerActionView && action != moduleapi.TaskOwnerActionCancel && action != moduleapi.TaskOwnerActionRetry {
		return errors.New("docker image task owner action is unsupported")
	}
	return a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permission)
}

func registerDockerImagePullTask(registrar moduleapi.TaskRuntimeRegistrar, service *service) error {
	if registrar == nil || service == nil {
		return errors.New("docker image pull task dependencies are unavailable")
	}
	if err := registrar.RegisterStageExecutor(&dockerImagePullTaskExecutor{service: service, cancels: make(map[uint64]context.CancelFunc)}); err != nil {
		return err
	}
	return registrar.RegisterTaskOwnerAuthorizer(dockerImageTaskOwnerAuthorizer{service: service})
}

// SubmitDockerImagePull 提交包含单个不可重试 Docker 拉取阶段的冻结 Task 计划。
func (s *service) SubmitDockerImagePull(ctx context.Context, reference string, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("task service is unavailable")
	}
	if err := validateDockerImageReference(reference); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	input, err := json.Marshal(dockerImagePullTaskInput{Reference: reference})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal docker image pull task input: %w", err)
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:           dockerImagePullTaskType,
		Owner:          moduleapi.TaskOwner{Type: dockerImageTaskOwnerType, ID: reference},
		RequestedBy:    requestedBy,
		IdempotencyKey: idempotencyKey,
		Input:          input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{
			Key:            "pull",
			ExecutorType:   dockerImagePullExecutor,
			Input:          input,
			RetryPolicy:    moduleapi.StageRetryPolicy{MaxAttempts: 1},
			RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
		}}},
	})
}

func taskReceiptResponse(receipt moduleapi.TaskReceipt) openapigen.TaskReceipt {
	// Task IDs are persisted as bigint and generated API uses int64.
	return openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)} // #nosec G115 -- PostgreSQL task IDs fit signed bigint.
}
