package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
			service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: true, tasks: tasks})
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
			assertContainerExternalExecution(t, stage, containerLifecycleOperation(action))
			var input containerLifecycleTaskInput
			if err := json.Unmarshal(stage.Input, &input); err != nil || input.ContainerRef != "container-1" || input.Force != (action == containerActionRemove) {
				t.Fatalf("unexpected frozen stage input %s: %v", stage.Input, err)
			}
		})
	}
}

func TestSubmitContainerLifecycleActionPublishesAcceptedTaskAudit(t *testing.T) {
	t.Parallel()

	bus, eventsPtr := newAuditCaptureBus(t, 1)
	service, err := newRouteTestService(containerServiceOptions{
		runtime:                 fakeRuntime{},
		auditBus:                bus,
		moduleName:              moduleID,
		enabled:                 true,
		dangerousActionsEnabled: true,
		tasks:                   &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}},
		defaultTail:             defaultContainerLogsDefaultTail,
		maxTail:                 defaultContainerLogsMaxTail,
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
	assertBatchLifecycleTaskSubmission(t, tasks, result, containerActionRemove, []string{"container-1", "container-2"})
}

func assertBatchLifecycleTaskSubmission(t *testing.T, tasks *containerTaskRuntimeStub, result BatchLifecycleActionResult, action string, expectedRefs []string) {
	t.Helper()
	if len(tasks.submissions) != 1 || result.AcceptedCount != len(expectedRefs) || len(result.Items) != len(expectedRefs) {
		t.Fatalf("expected one accepted task for %d containers, got submissions=%#v result=%#v", len(expectedRefs), tasks.submissions, result)
	}
	submission := tasks.submissions[0]
	if submission.Type != containerLifecycleBatchTaskType(action) || submission.Owner.Type != containerLifecycleBatchOwnerType(action) || submission.IdempotencyKey != "batch-key" {
		t.Fatalf("unexpected batch task submission %#v", submission)
	}
	if !isContainerLifecycleBatchOwnerID(submission.Owner.ID) {
		t.Fatalf("expected fixed-length batch owner digest, got %q", submission.Owner.ID)
	}
	assertBatchLifecycleTaskStages(t, submission.Plan.Stages, action, expectedRefs)
	assertBatchLifecycleTaskItems(t, result.Items, expectedRefs)
}

func assertBatchLifecycleTaskStages(t *testing.T, stages []moduleapi.StagePlan, action string, expectedRefs []string) {
	t.Helper()
	if len(stages) != len(expectedRefs) {
		t.Fatalf("unexpected batch stages %#v", stages)
	}
	for index, stage := range stages {
		if stage.Key != fmt.Sprintf("%s-%d", action, index+1) {
			t.Fatalf("unexpected batch stage key %#v", stage)
		}
		assertContainerExternalExecution(t, stage, containerLifecycleOperation(action))
		var input containerLifecycleTaskInput
		if err := json.Unmarshal(stage.Input, &input); err != nil {
			t.Fatalf("decode batch stage input: %v", err)
		}
		if input.ContainerRef != expectedRefs[index] {
			t.Fatalf("expected stage %d to target %q, got %q", index, expectedRefs[index], input.ContainerRef)
		}
	}
}

func assertContainerExternalExecution(t *testing.T, stage moduleapi.StagePlan, operation string) {
	t.Helper()
	execution := stage.ExternalExecution
	if execution == nil || execution.RuntimeTargetID != 1 || execution.ProviderID != containerExecutionProvider ||
		execution.Capability != containerExecutionCapability || execution.CapabilityVersion != containerExecutionCapabilityVersion ||
		execution.Protocol != containerExecutionProtocol || execution.OperationID != operation || execution.PayloadSHA256 == "" ||
		execution.LeaseTTL != containerExecutionLeaseTTL || execution.AbsoluteDeadline <= execution.LeaseTTL {
		t.Fatalf("unexpected container external execution binding %#v", execution)
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(stage.Input, &envelope); err != nil {
		t.Fatalf("decode provider-neutral input: %v", err)
	}
	if _, exists := envelope["operation"]; exists {
		t.Fatalf("operation must be owned by ExternalExecution, got %s", stage.Input)
	}
	if _, exists := envelope["version"]; exists {
		t.Fatalf("version must be owned by ExternalExecution, got %s", stage.Input)
	}
}

func TestSubmitContainerLifecycleBatchActionUsesFixedLengthOwnerForMaximumBatch(t *testing.T) {
	t.Parallel()

	refs := make([]Ref, maxContainerBatchActionIDs)
	for index := range refs {
		refs[index] = Ref{Value: fmt.Sprintf("%064x", index)}
	}
	tasks := &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}}
	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, dangerousActionsEnabled: true, tasks: tasks})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if _, err := service.SubmitContainerLifecycleBatchAction(context.Background(), refs, containerActionRemove, ActionOptions{Force: true}, 7, "batch-key"); err != nil {
		t.Fatalf("submit maximum batch lifecycle task: %v", err)
	}
	submission := tasks.submissions[0]
	if len(submission.Owner.ID) > moduleapi.TaskOwnerIDMaxRunes || !isContainerLifecycleBatchOwnerID(submission.Owner.ID) {
		t.Fatalf("expected owner id to fit tasks.owner_id, got %q", submission.Owner.ID)
	}
	expectedRefs := make([]string, len(refs))
	for index, ref := range refs {
		expectedRefs[index] = ref.Value
	}
	assertBatchLifecycleTaskStages(t, submission.Plan.Stages, containerActionRemove, expectedRefs)
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
		runtime: &countingRuntime{detail: Detail{Summary: Summary{
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

func TestValidateContainerLifecycleTaskOwnerWrapsMalformedLegacyBatchOwner(t *testing.T) {
	t.Parallel()

	err := validateContainerLifecycleTaskOwner("not-json", true)
	var syntaxErr *json.SyntaxError
	if !errors.As(err, &syntaxErr) {
		t.Fatalf("expected wrapped JSON syntax error, got %v", err)
	}
	if err := validateContainerLifecycleTaskOwner(`[]`, true); err == nil {
		t.Fatal("expected empty legacy batch owner to stay invalid")
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
