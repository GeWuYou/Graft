package container

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"graft/server/internal/moduleapi"
)

const (
	containerLifecycleTaskOwnerPrefix    = "container_lifecycle_"
	containerLifecycleBatchOwnerPrefix   = "container_lifecycle_batch_"
	containerLifecycleBatchOwnerIDPrefix = "sha256:"
)

type containerLifecycleTaskInput struct {
	ContainerRef string `json:"container_ref"`
	Force        bool   `json:"force"`
}

type containerLifecycleTaskOwnerAuthorizer struct {
	service *service
	action  string
	batch   bool
}

func (a containerLifecycleTaskOwnerAuthorizer) OwnerType() string {
	if a.batch {
		return containerLifecycleBatchOwnerType(a.action)
	}
	return containerLifecycleTaskOwnerType(a.action)
}

func (a containerLifecycleTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return moduleapi.ErrUnauthenticated
	}
	if owner.Type != a.OwnerType() {
		return errors.New("container lifecycle task owner type is invalid")
	}
	if err := validateContainerLifecycleTaskOwner(owner.ID, a.batch); err != nil {
		return err
	}
	if !isContainerLifecycleTaskOwnerAction(action) {
		return errors.New("container lifecycle task owner action is unsupported")
	}
	return a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permissionForAction(a.action))
}

func validateContainerLifecycleTaskOwner(ownerID string, batch bool) error {
	if !batch {
		_, err := parseRef(ownerID)
		return err
	}
	if isContainerLifecycleBatchOwnerID(ownerID) {
		return nil
	}
	// COMPAT(owner=container module Task owner, cleanup=all pre-hash batch Tasks reach terminal state)
	// 旧 Task 的 owner 仍是引用列表 JSON；仅用于授权已持久化任务，新的批量提交一律写入固定长度摘要。
	var refs []string
	if err := json.Unmarshal([]byte(ownerID), &refs); err != nil {
		return fmt.Errorf("decode container lifecycle batch owner: %w", err)
	}
	if len(refs) == 0 {
		return errors.New("container lifecycle batch owner is invalid")
	}
	for _, rawRef := range refs {
		if _, err := parseRef(rawRef); err != nil {
			return err
		}
	}
	return nil
}

func isContainerLifecycleTaskOwnerAction(action moduleapi.TaskOwnerAction) bool {
	return action == moduleapi.TaskOwnerActionView || action == moduleapi.TaskOwnerActionCancel || action == moduleapi.TaskOwnerActionRetry
}

func registerContainerLifecycleTasks(registrar moduleapi.TaskRuntimeRegistrar, service *service) error {
	if registrar == nil || service == nil {
		return errors.New("container lifecycle task dependencies are unavailable")
	}
	for _, action := range containerLifecycleTaskActions() {
		if err := registrar.RegisterTaskOwnerAuthorizer(containerLifecycleTaskOwnerAuthorizer{service: service, action: action}); err != nil {
			return err
		}
		if err := registrar.RegisterTaskOwnerAuthorizer(containerLifecycleTaskOwnerAuthorizer{service: service, action: action, batch: true}); err != nil {
			return err
		}
	}
	return nil
}

// SubmitContainerLifecycleAction 提交单阶段容器生命周期 Task，副作用只能由受约束的 Runtime Agent 执行。
func (s *service) SubmitContainerLifecycleAction(ctx context.Context, ref Ref, action string, options ActionOptions, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("task service is unavailable")
	}
	if !isContainerLifecycleTaskAction(action) {
		return moduleapi.TaskReceipt{}, errInvalidBatchAction
	}
	if _, err := parseRef(ref.Value); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	input, err := json.Marshal(containerLifecycleTaskInput{ContainerRef: ref.Value, Force: options.Force})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container lifecycle task input: %w", err)
	}
	execution, err := s.containerExternalExecution(ctx, containerLifecycleOperation(action), input)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	receipt, submitErr := s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:           containerLifecycleTaskType(action),
		Owner:          moduleapi.TaskOwner{Type: containerLifecycleTaskOwnerType(action), ID: ref.Value},
		RequestedBy:    requestedBy,
		IdempotencyKey: idempotencyKey,
		Input:          input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{
			Key:               action,
			ExecutorType:      containerLifecycleTaskExecutorType(action),
			Input:             input,
			RetryPolicy:       moduleapi.StageRetryPolicy{MaxAttempts: 1},
			RecoveryPolicy:    moduleapi.StageRecoveryManualReconcile,
			ExternalExecution: execution,
		}}},
	})
	s.publishLifecycleTaskSubmissionAudit(ctx, ref, action, options, receipt, submitErr)
	return receipt, submitErr
}

// SubmitContainerLifecycleBatchAction 提交一个包含多个顺序阶段的容器生命周期 Task，使批量操作保留单一进度、日志与取消入口。
func (s *service) SubmitContainerLifecycleBatchAction(ctx context.Context, refs []Ref, action string, options ActionOptions, requestedBy uint64, idempotencyKey string) (moduleapi.TaskReceipt, error) {
	if s == nil || s.tasks == nil {
		return moduleapi.TaskReceipt{}, errors.New("task service is unavailable")
	}
	if !isContainerLifecycleTaskAction(action) || len(refs) == 0 {
		return moduleapi.TaskReceipt{}, errInvalidBatchAction
	}
	if err := s.requireRuntimeAccess(ctx); err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	if !s.dangerousActionsAllowed(ctx) {
		return moduleapi.TaskReceipt{}, errDangerousActionsDisabled
	}
	ownerRefs, stages, err := s.containerLifecycleBatchStages(ctx, refs, action, options)
	if err != nil {
		return moduleapi.TaskReceipt{}, err
	}
	ownerID, err := containerLifecycleBatchOwnerID(ownerRefs)
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container lifecycle batch owner: %w", err)
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:           containerLifecycleBatchTaskType(action),
		Owner:          moduleapi.TaskOwner{Type: containerLifecycleBatchOwnerType(action), ID: ownerID},
		RequestedBy:    requestedBy,
		IdempotencyKey: idempotencyKey,
		Plan:           moduleapi.TaskPlan{Stages: stages},
	})
}

func (s *service) containerLifecycleBatchStages(ctx context.Context, refs []Ref, action string, options ActionOptions) ([]string, []moduleapi.StagePlan, error) {
	ownerRefs := make([]string, 0, len(refs))
	stages := make([]moduleapi.StagePlan, 0, len(refs))
	for index, ref := range refs {
		parsed, err := parseRef(ref.Value)
		if err != nil {
			return nil, nil, err
		}
		input, err := json.Marshal(containerLifecycleTaskInput{ContainerRef: parsed.Value, Force: options.Force})
		if err != nil {
			return nil, nil, fmt.Errorf("marshal container lifecycle batch input: %w", err)
		}
		execution, err := s.containerExternalExecution(ctx, containerLifecycleOperation(action), input)
		if err != nil {
			return nil, nil, err
		}
		ownerRefs = append(ownerRefs, parsed.Value)
		stages = append(stages, moduleapi.StagePlan{
			Key:               fmt.Sprintf("%s-%d", action, index+1),
			ExecutorType:      containerLifecycleTaskExecutorType(action),
			Input:             input,
			RetryPolicy:       moduleapi.StageRetryPolicy{MaxAttempts: 1},
			RecoveryPolicy:    moduleapi.StageRecoveryManualReconcile,
			ExternalExecution: execution,
		})
	}
	return ownerRefs, stages, nil
}

func containerLifecycleBatchOwnerID(refs []string) (string, error) {
	encodedRefs, err := json.Marshal(refs)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(encodedRefs)
	return containerLifecycleBatchOwnerIDPrefix + hex.EncodeToString(digest[:]), nil
}

func isContainerLifecycleBatchOwnerID(ownerID string) bool {
	if !strings.HasPrefix(ownerID, containerLifecycleBatchOwnerIDPrefix) {
		return false
	}
	digest, err := hex.DecodeString(strings.TrimPrefix(ownerID, containerLifecycleBatchOwnerIDPrefix))
	return err == nil && len(digest) == sha256.Size && hex.EncodeToString(digest) == strings.TrimPrefix(ownerID, containerLifecycleBatchOwnerIDPrefix)
}

func containerLifecycleTaskActions() []string {
	return []string{containerActionStart, containerActionStop, containerActionRestart, containerActionRemove}
}

func isContainerLifecycleTaskAction(action string) bool {
	for _, candidate := range containerLifecycleTaskActions() {
		if action == candidate {
			return true
		}
	}
	return false
}

func containerLifecycleTaskOwnerType(action string) string {
	return containerLifecycleTaskOwnerPrefix + action
}

func containerLifecycleBatchOwnerType(action string) string {
	return containerLifecycleBatchOwnerPrefix + action
}

func containerLifecycleTaskType(action string) moduleapi.TaskType {
	return moduleapi.TaskType("container.lifecycle." + action + ".v1")
}

func containerLifecycleBatchTaskType(action string) moduleapi.TaskType {
	return moduleapi.TaskType("container.lifecycle." + action + ".batch.v1")
}

func containerLifecycleTaskExecutorType(action string) moduleapi.StageExecutorType {
	return moduleapi.StageExecutorType("container.lifecycle." + action + ".v1")
}
