package container

import (
	"context"
	"encoding/json"
	"errors"
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
	if submission.Type != dockerImagePullTaskType || submission.Owner.Type != dockerImageTaskOwnerType || submission.Owner.ID != "alpine:3.20" || submission.RequestedBy != 7 || submission.IdempotencyKey != "pull-alpine" {
		t.Fatalf("unexpected task submission: %#v", submission)
	}
	if len(submission.Plan.Stages) != 1 {
		t.Fatalf("expected one stage, got %#v", submission.Plan)
	}
	stage := submission.Plan.Stages[0]
	if stage.Key != "pull" || stage.ExecutorType != dockerImagePullExecutor || stage.RetryPolicy.MaxAttempts != 1 || stage.RecoveryPolicy != moduleapi.StageRecoveryManualReconcile {
		t.Fatalf("unexpected stage plan: %#v", stage)
	}
	var input dockerImagePullTaskInput
	if err := json.Unmarshal(stage.Input, &input); err != nil || input.Reference != "alpine:3.20" {
		t.Fatalf("unexpected frozen stage input %s: %v", stage.Input, err)
	}
}

func TestDockerImagePullTaskExecutorWritesSanitizedProgress(t *testing.T) {
	t.Parallel()

	service, err := newRouteTestService(containerServiceOptions{runtime: pullErrorRuntime{}, enabled: true})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	input, err := json.Marshal(dockerImagePullTaskInput{Reference: "alpine:3.20"})
	if err != nil {
		t.Fatalf("marshal task input: %v", err)
	}
	run := &dockerImagePullStageRun{input: input, stageID: 13}
	executor := &dockerImagePullTaskExecutor{service: service, cancels: make(map[uint64]context.CancelFunc)}
	if err := executor.Execute(context.Background(), run); !errors.Is(err, errDockerImagePullFailed) {
		t.Fatalf("expected mapped pull failure, got %v", err)
	}
	if len(run.logs) != 1 || run.logs[0].Level != "error" || run.logs[0].Line != "error" {
		t.Fatalf("expected one sanitized error log, got %#v", run.logs)
	}
}

func TestDockerImagePullTaskExecutorCancelsRunningPull(t *testing.T) {
	t.Parallel()

	cancelled := make(chan struct{})
	executor := &dockerImagePullTaskExecutor{cancels: map[uint64]context.CancelFunc{13: func() { close(cancelled) }}}
	if err := executor.Cancel(context.Background(), &dockerImagePullStageRun{stageID: 13}); err != nil {
		t.Fatalf("cancel docker image pull task: %v", err)
	}
	select {
	case <-cancelled:
	default:
		t.Fatal("expected running pull cancellation")
	}
}

type dockerImagePullStageRun struct {
	input   json.RawMessage
	stageID uint64
	logs    []moduleapi.TaskLogEntry
}

func (*dockerImagePullStageRun) TaskID() uint64 { return 1 }

func (r *dockerImagePullStageRun) StageID() uint64 { return r.stageID }

func (*dockerImagePullStageRun) Attempt() int { return 1 }

func (r *dockerImagePullStageRun) Input() json.RawMessage {
	return append(json.RawMessage(nil), r.input...)
}

func (*dockerImagePullStageRun) CancellationRequested(context.Context) bool { return false }

func (r *dockerImagePullStageRun) AppendLog(_ context.Context, entry moduleapi.TaskLogEntry) error {
	r.logs = append(r.logs, entry)
	return nil
}
