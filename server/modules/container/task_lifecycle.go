package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"graft/server/internal/moduleapi"
)

const containerLifecycleTaskOwnerPrefix = "container_lifecycle_"

type containerLifecycleTaskInput struct {
	Ref   string `json:"ref"`
	Force bool   `json:"force"`
}

type containerLifecycleTaskExecutor struct {
	service *service
	action  string
	mu      sync.Mutex
	cancels map[uint64]context.CancelFunc
}

func (e *containerLifecycleTaskExecutor) Type() moduleapi.StageExecutorType {
	return containerLifecycleTaskExecutorType(e.action)
}

func (e *containerLifecycleTaskExecutor) Execute(ctx context.Context, run moduleapi.StageRun) error {
	if e == nil || e.service == nil {
		return errors.New("container lifecycle task executor is unavailable")
	}
	var input containerLifecycleTaskInput
	if err := json.Unmarshal(run.Input(), &input); err != nil {
		return fmt.Errorf("decode container lifecycle task input: %w", err)
	}
	ref, err := parseRef(input.Ref)
	if err != nil {
		return err
	}
	actionCtx, cancel := context.WithCancel(ctx)
	e.mu.Lock()
	e.cancels[run.StageID()] = cancel
	e.mu.Unlock()
	defer func() {
		e.mu.Lock()
		delete(e.cancels, run.StageID())
		e.mu.Unlock()
		cancel()
	}()
	_, err = e.service.runAction(actionCtx, ref, e.action, ActionOptions{Force: input.Force})
	return err
}

func (e *containerLifecycleTaskExecutor) Cancel(_ context.Context, run moduleapi.StageRun) error {
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

type containerLifecycleTaskOwnerAuthorizer struct {
	service *service
	action  string
}

func (a containerLifecycleTaskOwnerAuthorizer) OwnerType() string {
	return containerLifecycleTaskOwnerType(a.action)
}

func (a containerLifecycleTaskOwnerAuthorizer) AuthorizeTaskOwner(ctx context.Context, actor *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, owner moduleapi.TaskOwner) error {
	if actor == nil || a.service == nil || a.service.authorizer == nil {
		return moduleapi.ErrUnauthenticated
	}
	if owner.Type != a.OwnerType() {
		return errors.New("container lifecycle task owner type is invalid")
	}
	if _, err := parseRef(owner.ID); err != nil {
		return err
	}
	if action != moduleapi.TaskOwnerActionView && action != moduleapi.TaskOwnerActionCancel && action != moduleapi.TaskOwnerActionRetry {
		return errors.New("container lifecycle task owner action is unsupported")
	}
	return a.service.authorizer.Authorize(ctx, moduleapi.RequestAuthContext{User: actor}, permissionForAction(a.action))
}

func registerContainerLifecycleTasks(registrar moduleapi.TaskRuntimeRegistrar, service *service) error {
	if registrar == nil || service == nil {
		return errors.New("container lifecycle task dependencies are unavailable")
	}
	for _, action := range containerLifecycleTaskActions() {
		if err := registrar.RegisterStageExecutor(&containerLifecycleTaskExecutor{
			service: service,
			action:  action,
			cancels: make(map[uint64]context.CancelFunc),
		}); err != nil {
			return err
		}
		if err := registrar.RegisterTaskOwnerAuthorizer(containerLifecycleTaskOwnerAuthorizer{service: service, action: action}); err != nil {
			return err
		}
	}
	return nil
}

// SubmitContainerLifecycleAction 提交单阶段、人工对账恢复的容器生命周期 Task，实际副作用仍由 runAction 持有。
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
	input, err := json.Marshal(containerLifecycleTaskInput{Ref: ref.Value, Force: options.Force})
	if err != nil {
		return moduleapi.TaskReceipt{}, fmt.Errorf("marshal container lifecycle task input: %w", err)
	}
	return s.tasks.Submit(ctx, moduleapi.SubmitTaskInput{
		Type:           containerLifecycleTaskType(action),
		Owner:          moduleapi.TaskOwner{Type: containerLifecycleTaskOwnerType(action), ID: ref.Value},
		RequestedBy:    requestedBy,
		IdempotencyKey: idempotencyKey,
		Input:          input,
		Plan: moduleapi.TaskPlan{Stages: []moduleapi.StagePlan{{
			Key:            action,
			ExecutorType:   containerLifecycleTaskExecutorType(action),
			Input:          input,
			RetryPolicy:    moduleapi.StageRetryPolicy{MaxAttempts: 1},
			RecoveryPolicy: moduleapi.StageRecoveryManualReconcile,
		}}},
	})
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

func containerLifecycleTaskType(action string) moduleapi.TaskType {
	return moduleapi.TaskType("container.lifecycle." + action + ".v1")
}

func containerLifecycleTaskExecutorType(action string) moduleapi.StageExecutorType {
	return moduleapi.StageExecutorType("container.lifecycle." + action + ".v1")
}
