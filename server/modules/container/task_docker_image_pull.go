package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	openapigen "graft/server/internal/contract/openapi/generated"
	"graft/server/internal/moduleapi"
)

const (
	dockerImagePullTaskType = moduleapi.TaskType(containerImagePullOperation)
	dockerImagePullExecutor = moduleapi.StageExecutorType(containerImagePullOperation)
)

type dockerImagePullTaskInput struct {
	ImageRef string `json:"image_ref"`
}

// SubmitDockerImagePull 提交包含单个不可重试 Docker 拉取阶段的冻结 Task 计划。
func (s *service) SubmitDockerImagePull(ctx context.Context, reference string, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("task service is unavailable")
	}
	if err := validateDockerImageReference(reference); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	input, err := json.Marshal(dockerImagePullTaskInput{ImageRef: reference})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal docker image pull task input: %w", err)
	}
	execution, err := s.containerExternalExecution(ctx, containerImagePullOperation, input)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:           dockerImagePullTaskType,
		Owner:          moduleapi.TaskOwner{Type: containerExternalTaskOwnerType(containerImagePullOperation, false), ID: reference},
		RequestedBy:    requestedBy,
		IdempotencyKey: idempotencyKey,
		Input:          input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{
			Key:               "pull",
			ExecutorType:      dockerImagePullExecutor,
			Input:             input,
			RetryPolicy:       moduleapi.StageRetryPolicy{MaxAttempts: 1},
			RecoveryPolicy:    moduleapi.StageRecoveryManualReconcile,
			ExternalExecution: execution,
		}}},
	})
}

func taskReceiptResponse(receipt moduleapi.TaskReceipt) openapigen.TaskReceipt {
	// Task IDs are persisted as bigint and generated API uses int64.
	return openapigen.TaskReceipt{TaskId: int64(receipt.TaskID), Status: openapigen.TaskStatus(receipt.Status)} // #nosec G115 -- PostgreSQL task IDs fit signed bigint.
}
