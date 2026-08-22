package container

import (
	"context"
	"encoding/json"
	"testing"

	"graft/server/internal/moduleapi"
)

//nolint:gocyclo // The assertions intentionally cover every immutable TaskPlan field in one scenario.
func TestSubmitDockerImagePullFreezesSingleManualReconcileStage(t *testing.T) {
	t.Parallel()

	tasks := &containerTaskRuntimeStub{receipt: moduleapi.TaskReceipt{TaskID: 42, Status: moduleapi.TaskStatusPending}}
	service, err := newRouteTestService(containerServiceOptions{runtime: fakeRuntime{}, enabled: true, tasks: tasks})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	receipt, err := service.SubmitDockerImagePull(context.Background(), "alpine:3.20", 7, "pull-alpine")
	if err != nil {
		t.Fatalf("submit docker image pull: %v", err)
	}
	if receipt.TaskID != 42 || len(tasks.submissions) != 1 {
		t.Fatalf("unexpected receipt or submissions: %#v %#v", receipt, tasks.submissions)
	}
	submission := tasks.submissions[0]
	if submission.Type != dockerImagePullTaskType || submission.Owner.Type != containerExternalTaskOwnerType(containerImagePullOperation, false) || submission.Owner.ID != "alpine:3.20" || submission.RequestedBy != 7 || submission.IdempotencyKey != "pull-alpine" {
		t.Fatalf("unexpected task submission: %#v", submission)
	}
	if len(submission.Plan.Stages) != 1 {
		t.Fatalf("expected one stage, got %#v", submission.Plan)
	}
	stage := submission.Plan.Stages[0]
	if stage.Key != "pull" || stage.ExecutorType != dockerImagePullExecutor || stage.RetryPolicy.MaxAttempts != 1 || stage.RecoveryPolicy != moduleapi.StageRecoveryManualReconcile {
		t.Fatalf("unexpected stage plan: %#v", stage)
	}
	assertContainerExternalExecution(t, stage, containerImagePullOperation)
	if stage.ExternalExecution.AbsoluteDeadline != containerImagePullDeadline {
		t.Fatalf("expected pull-specific deadline, got %#v", stage.ExternalExecution)
	}
	var input dockerImagePullTaskInput
	if err := json.Unmarshal(stage.Input, &input); err != nil || input.ImageRef != "alpine:3.20" {
		t.Fatalf("unexpected frozen stage input %s: %v", stage.Input, err)
	}
}
