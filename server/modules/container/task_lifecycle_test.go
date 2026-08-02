package container

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"graft/server/internal/httpx"
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

func TestSubmitContainerLifecycleActionPublishesAcceptedTaskAudit(t *testing.T) {
	t.Parallel()

	bus, eventsPtr := newAuditCaptureBus(t, 1)
	service, err := newRouteTestService(containerServiceOptions{
		runtime:     fakeRuntime{},
		auditBus:    bus,
		moduleName:  moduleID,
		enabled:     true,
		tasks:       &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}},
		defaultTail: defaultContainerLogsDefaultTail,
		maxTail:     defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	ctx := httpx.WithRequestAuditContext(context.Background(), httpx.RequestAuditContext{RequestID: "req-lifecycle-submit"})
	if _, err := service.SubmitContainerLifecycleAction(ctx, Ref{Value: "container-1"}, containerActionStart, ActionOptions{}, 7, "submit-key"); err != nil {
		t.Fatalf("submit lifecycle task: %v", err)
	}
	events := *eventsPtr
	if len(events) != 1 {
		t.Fatalf("expected one task submission audit, got %#v", events)
	}
	event := events[0]
	if event.Action != containercontract.ContainerAuditActionStart.String() || !event.Success || event.Metadata["submission"] != "success" || event.Metadata["task_id"] != uint64(42) || event.Metadata["execution_state"] != "not_started" || event.Metadata["requestId"] != "req-lifecycle-submit" {
		t.Fatalf("unexpected task submission audit %#v", event)
	}
}

func TestBatchLifecycleActionSubmitsOneTaskWithOrderedContainerStages(t *testing.T) {
	t.Parallel()

	tasks := &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}}
	service, err := newRouteTestService(containerServiceOptions{
		runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: true, tasks: tasks,
		defaultTail: defaultContainerLogsDefaultTail, maxTail: defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.BatchLifecycleAction(context.Background(), BatchActionCommand{Action: containerActionRemove, IDs: []string{"container-1", "container-2"}, Force: true}, 7, "batch-key")
	if err != nil {
		t.Fatalf("submit batch lifecycle action: %v", err)
	}
	assertBatchLifecycleTaskSubmission(t, tasks, result)
}

func assertBatchLifecycleTaskSubmission(t *testing.T, tasks *containerTaskRuntimeStub, result BatchLifecycleActionResult) {
	t.Helper()
	if len(tasks.submissions) != 1 || result.AcceptedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("expected one accepted task for two containers, got submissions=%#v result=%#v", tasks.submissions, result)
	}
	submission := tasks.submissions[0]
	if submission.Type != containerLifecycleBatchTaskType(containerActionRemove) || submission.Owner.Type != containerLifecycleBatchOwnerType(containerActionRemove) || submission.IdempotencyKey != "batch-key" {
		t.Fatalf("unexpected batch task submission %#v", submission)
	}
	ownerRefs := batchLifecycleTaskOwnerRefs(t, submission.Owner.ID)
	assertBatchLifecycleTaskStages(t, submission.Plan.Stages)
	assertBatchLifecycleTaskItems(t, result.Items, ownerRefs)
}

func batchLifecycleTaskOwnerRefs(t *testing.T, ownerID string) []string {
	t.Helper()
	var ownerRefs []string
	if err := json.Unmarshal([]byte(ownerID), &ownerRefs); err != nil || len(ownerRefs) != 2 || ownerRefs[0] != "container-1" || ownerRefs[1] != "container-2" {
		t.Fatalf("unexpected batch owner %q: %v", ownerID, err)
	}
	return ownerRefs
}

func assertBatchLifecycleTaskStages(t *testing.T, stages []moduleapi.StagePlan) {
	t.Helper()
	if len(stages) != 2 || stages[0].Key != "remove-1" || stages[1].Key != "remove-2" {
		t.Fatalf("unexpected batch stages %#v", stages)
	}
}

func assertBatchLifecycleTaskItems(t *testing.T, items []BatchLifecycleActionItem, ownerRefs []string) {
	t.Helper()
	for index, item := range items {
		if !item.Accepted || item.TaskID != 42 || item.ID != ownerRefs[index] {
			t.Fatalf("expected each item to reference the shared task, got %#v", items)
		}
	}
}

func TestBatchLifecycleActionPublishesParseAndSubmissionFailureAudits(t *testing.T) {
	t.Parallel()

	bus, eventsPtr := newAuditCaptureBus(t, 2)
	service, err := newRouteTestService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		auditBus:                bus,
		moduleName:              moduleID,
		enabled:                 true,
		dangerousActionsEnabled: true,
		tasks:                   &containerTaskRuntimeStub{err: moduleapi.ErrTaskSubmissionConflict},
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.BatchLifecycleAction(context.Background(), BatchActionCommand{Action: containerActionStart, IDs: []string{"bad/id", "container-1"}}, 7, "batch-key")
	if err != nil {
		t.Fatalf("batch lifecycle action: %v", err)
	}
	if result.FailedCount != 2 || len(result.Items) != 2 {
		t.Fatalf("unexpected batch result %#v", result)
	}
	events := *eventsPtr
	if len(events) != 2 {
		t.Fatalf("expected parse and submission failure audits, got %#v", events)
	}
	for _, event := range events {
		if event.Action != containercontract.ContainerAuditActionStart.String() || event.Success || event.Metadata["submission"] != "failed" || event.Metadata["execution_state"] != "not_started" {
			t.Fatalf("unexpected failed task submission audit %#v", event)
		}
	}
}

func TestBatchLifecycleActionPublishesPolicyFailureAudit(t *testing.T) {
	t.Parallel()

	bus, eventsPtr := newAuditCaptureBus(t, 1)
	service, err := newRouteTestService(containerServiceOptions{
		runtime: &managedActionRuntime{detail: Detail{Summary: Summary{
			ID: "web", Name: "graft-web", Image: "graft/web:latest", Runtime: runtimeNameDocker,
			Orchestrator: OrchestratorInfo{Type: containerOrchestratorCompose, Managed: true, GroupScopeKind: composeProjectScopeKind, GroupValue: "graft", MemberScopeKind: composeServiceScopeKind, MemberValue: "web"},
		}}},
		systemConfig: serviceTestPolicyConfig{
			serviceTestSystemConfig: serviceTestSystemConfig{values: map[string]bool{
				containercontract.ContainerRuntimeEnabledConfig.String():          true,
				containercontract.ContainerDangerousActionsEnabledConfig.String(): true,
			}},
			values: map[string]string{containercontract.ContainerComposeActionLevelConfig.String(): string(mustRawJSON("warn"))},
		},
		auditBus:                bus,
		moduleName:              moduleID,
		enabled:                 true,
		dangerousActionsEnabled: true,
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	result, err := service.BatchLifecycleAction(context.Background(), BatchActionCommand{Action: containerActionRemove, IDs: []string{"web"}, Force: true}, 7, "batch-key")
	if err != nil {
		t.Fatalf("batch lifecycle action: %v", err)
	}
	if result.FailedCount != 1 || len(result.Items) != 1 || result.Items[0].ErrorCode != containercontract.ContainerDangerousActionsDisabled.String() {
		t.Fatalf("unexpected policy result %#v", result)
	}
	events := *eventsPtr
	if len(events) != 1 || events[0].Action != containercontract.ContainerAuditActionRemove.String() || events[0].Success || events[0].Metadata["result"] != "failed" || events[0].Metadata["force"] != true {
		t.Fatalf("unexpected policy failure audit %#v", events)
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
	run := &dockerImagePullStageRun{input: input, stageID: 13}
	executor := &containerLifecycleTaskExecutor{service: service, action: containerActionRestart, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), run); err != nil {
		t.Fatalf("execute lifecycle task: %v", err)
	}
	if len(run.logs) != 1 || run.logs[0].Stream != "system" || run.logs[0].Level != "info" || run.logs[0].Line != "container lifecycle action restart completed" {
		t.Fatalf("expected sanitized success result log, got %#v", run.logs)
	}
}

func TestContainerLifecycleTaskExecutorWritesFailureResultLog(t *testing.T) {
	t.Parallel()

	actionErr := errors.New("runtime action failed")
	service, err := newRouteTestService(containerServiceOptions{runtime: failingRuntime{err: actionErr}, enabled: true, dangerousActionsEnabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	input, err := json.Marshal(containerLifecycleTaskInput{Ref: "container-1"})
	if err != nil {
		t.Fatalf("marshal task input: %v", err)
	}
	run := &dockerImagePullStageRun{input: input, stageID: 13}
	executor := &containerLifecycleTaskExecutor{service: service, action: containerActionRemove, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), run); !errors.Is(err, actionErr) {
		t.Fatalf("expected action failure, got %v", err)
	}
	if len(run.logs) != 1 || run.logs[0].Stream != "system" || run.logs[0].Level != "error" || run.logs[0].Line != "container lifecycle action remove failed" {
		t.Fatalf("expected sanitized failure result log, got %#v", run.logs)
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
			for _, batch := range []bool{false, true} {
				assertContainerLifecycleTaskOwnerAuthorization(t, action, batch)
			}
		})
	}
}

func assertContainerLifecycleTaskOwnerAuthorization(t *testing.T, action string, batch bool) {
	t.Helper()
	authorizer := &recordingAuthorizer{}
	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, authorizer: authorizer})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	owner := moduleapi.TaskOwner{Type: containerLifecycleTaskOwnerType(action), ID: "container-1"}
	if batch {
		owner = moduleapi.TaskOwner{Type: containerLifecycleBatchOwnerType(action), ID: `["container-1","container-2"]`}
	}
	err = (containerLifecycleTaskOwnerAuthorizer{service: service, action: action, batch: batch}).AuthorizeTaskOwner(
		context.Background(), &moduleapi.CurrentUser{ID: 7}, moduleapi.TaskOwnerActionRetry, owner,
	)
	if err != nil {
		t.Fatalf("authorize task owner: %v", err)
	}
	if len(authorizer.permissions) != 1 || authorizer.permissions[0] != permissionForAction(action) {
		t.Fatalf("expected %s permission, got %#v", permissionForAction(action), authorizer.permissions)
	}
}

func TestContainerLifecycleTaskPermissionsStayActionSpecific(t *testing.T) {
	t.Parallel()

	if permissionForAction(containerActionStart) != containercontract.ContainerStartPermission.String() || permissionForAction(containerActionStop) != containercontract.ContainerStopPermission.String() || permissionForAction(containerActionRestart) != containercontract.ContainerRestartPermission.String() || permissionForAction(containerActionRemove) != containercontract.ContainerRemovePermission.String() {
		t.Fatal("expected action-specific lifecycle permissions")
	}
}
