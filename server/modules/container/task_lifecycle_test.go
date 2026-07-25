package container

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
	containercontract "graft/server/modules/container/contract"
)

//nolint:gocognit,gocyclo,cyclop // 该测试在单一场景中集中断言每个生命周期动作的所有不可变 TaskPlan 字段。
func TestSubmitContainerLifecycleActionFreezesSingleManualReconcileStage(t *testing.T) {
	t.Parallel()

	for _, action := range containerLifecycleTaskActions() {
		t.Run(action, func(t *testing.T) {
			tasks := &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}}
			service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, tasks: tasks})
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			receipt, err := service.SubmitContainerLifecycleAction(context.Background(), Ref{Value: "container-1"}, action, ActionOptions{Force: action == containerActionRemove}, 7, "lifecycle-"+action)
			if err != nil {
				t.Fatalf("submit lifecycle task: %v", err)
			}
			if receipt.TaskID != 42 || len(tasks.submissions) != 1 {
				t.Fatalf("unexpected receipt or submissions: %#v %#v", receipt, tasks.submissions)
			}
			submission := tasks.submissions[0]
			if submission.Type != containerLifecycleTaskType(action) || submission.Owner.Type != containerLifecycleTaskOwnerType(action) || submission.Owner.ID != "container-1" || submission.RequestedBy != 7 || submission.IdempotencyKey != "lifecycle-"+action {
				t.Fatalf("unexpected task submission: %#v", submission)
			}
			if len(submission.Plan.Stages) != 1 {
				t.Fatalf("expected one stage, got %#v", submission.Plan)
			}
			stage := submission.Plan.Stages[0]
			if stage.Key != action || stage.ExecutorType != containerLifecycleTaskExecutorType(action) || stage.RetryPolicy.MaxAttempts != 1 || stage.RecoveryPolicy != moduleapi.StageRecoveryManualReconcile {
				t.Fatalf("unexpected stage plan: %#v", stage)
			}
			var input containerLifecycleTaskInput
			if err := json.Unmarshal(stage.Input, &input); err != nil || input.Ref != "container-1" || input.Force != (action == containerActionRemove) {
				t.Fatalf("unexpected frozen stage input %s: %v", stage.Input, err)
			}
		})
	}
}

func TestContainerLifecycleTaskExecutorUsesRunAction(t *testing.T) {
	t.Parallel()

	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	input, err := json.Marshal(containerLifecycleTaskInput{Ref: "container-1"})
	if err != nil {
		t.Fatalf("marshal task input: %v", err)
	}
	executor := &containerLifecycleTaskExecutor{service: service, action: containerActionRestart, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), &dockerImagePullStageRun{input: input, stageID: 13}); err != nil {
		t.Fatalf("execute lifecycle task: %v", err)
	}
}

func TestContainerLifecycleTaskExecutorRetainsDangerousActionPolicy(t *testing.T) {
	t.Parallel()

	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: false})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	input, err := json.Marshal(containerLifecycleTaskInput{Ref: "container-1"})
	if err != nil {
		t.Fatalf("marshal task input: %v", err)
	}
	executor := &containerLifecycleTaskExecutor{service: service, action: containerActionStart, cancels: make(map[uint64]context.CancelFunc)}
	err = executor.Execute(context.Background(), &dockerImagePullStageRun{input: input, stageID: 13})
	if !errors.Is(err, errDangerousActionsDisabled) {
		t.Fatalf("expected existing dangerous action policy error, got %v", err)
	}
}

func TestContainerLifecycleTaskOwnerAuthorizerUsesActionPermission(t *testing.T) {
	t.Parallel()

	for _, action := range containerLifecycleTaskActions() {
		t.Run(action, func(t *testing.T) {
			authorizer := &recordingAuthorizer{}
			service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, authorizer: authorizer})
			if err != nil {
				t.Fatalf("new service: %v", err)
			}
			err = (containerLifecycleTaskOwnerAuthorizer{service: service, action: action}).AuthorizeTaskOwner(
				context.Background(),
				&moduleapi.CurrentUser{ID: 7},
				moduleapi.TaskOwnerActionRetry,
				moduleapi.TaskOwner{Type: containerLifecycleTaskOwnerType(action), ID: "container-1"},
			)
			if err != nil {
				t.Fatalf("authorize task owner: %v", err)
			}
			if len(authorizer.permissions) != 1 || authorizer.permissions[0] != permissionForAction(action) {
				t.Fatalf("expected %s permission, got %#v", permissionForAction(action), authorizer.permissions)
			}
		})
	}
}

func TestContainerLifecycleTaskPermissionsStayActionSpecific(t *testing.T) {
	t.Parallel()

	if permissionForAction(containerActionStart) != containercontract.ContainerStartPermission.String() || permissionForAction(containerActionStop) != containercontract.ContainerStopPermission.String() || permissionForAction(containerActionRestart) != containercontract.ContainerRestartPermission.String() || permissionForAction(containerActionRemove) != containercontract.ContainerRemovePermission.String() {
		t.Fatal("expected action-specific lifecycle permissions")
	}
}
