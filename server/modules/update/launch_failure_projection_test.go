package update

import (
	"context"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type launchFailureTaskQuery struct {
	taskQueryStub
	stages   []moduleapi.TaskStageView
	stageErr error
}

func (s *launchFailureTaskQuery) ListTaskStages(context.Context, uint64) ([]moduleapi.TaskStageView, error) {
	return s.stages, s.stageErr
}

func failedLaunchProjectionFixture() (*RolloutService, *launchFailureTaskQuery, ComposeUpdateOperation) {
	operation := ComposeUpdateOperation{OperationID: "update-launch-failure", TaskID: 81,
		Outcome: ExecutionOutcomePlanning, StartedAt: time.Now().UTC()}
	code, key := "invalid_execution_intent", "controller_launch"
	query := &launchFailureTaskQuery{taskQueryStub: taskQueryStub{tasks: map[uint64]moduleapi.TaskView{
		operation.TaskID: {ID: operation.TaskID, Type: composeUpdateTaskType,
			Owner:  moduleapi.TaskOwner{Type: platformUpdateTaskOwnerType, ID: operation.OperationID},
			Status: moduleapi.TaskStatusFailed, CurrentStageKey: &key, FailureCode: &code},
	}}, stages: []moduleapi.TaskStageView{
		{ID: 101, Key: "controller_launch", Sequence: 1, ExecutorType: composeUpdateLaunchExecutor,
			Status: moduleapi.StageStatusFailed, Attempt: 1, FailureCode: &code,
			ExecutionMode: "local", ExecutionVersion: "local-sdk/v1"},
		{ID: 102, Key: "controller_result", Sequence: 2, ExecutorType: composeUpdateExecutor,
			Status: moduleapi.StageStatusPending, ExecutionMode: "controller", ExecutionVersion: runnerProtocol},
	}}
	rollout := &RolloutService{operations: &memoryOperationStore{items: map[string]ComposeUpdateOperation{operation.OperationID: operation}},
		taskQuery: query, externalLaunches: map[string]*updateControllerLaunchSlot{operation.OperationID: {}}}
	return rollout, query, operation
}

func TestFailedLaunchIsVisibleWithoutPersistingTerminalHistoryOrDeletingRetryMaterial(t *testing.T) {
	rollout, query, operation := failedLaunchProjectionFixture()
	views, err := rollout.ListOperations(t.Context(), 10)
	if err != nil || len(views) != 1 {
		t.Fatalf("views=%#v err=%v", views, err)
	}
	if views[0].Phase != RunnerPhaseFailed || views[0].StateSource != "task_recovery" || views[0].Error != "invalid_execution_intent" {
		t.Fatalf("failed launch projection=%#v", views[0])
	}
	view, err := rollout.GetOperation(t.Context(), operation.OperationID)
	if err != nil || view.Error != views[0].Error || view.StateSource != "task_recovery" {
		t.Fatalf("view=%#v err=%v", view, err)
	}
	task := query.tasks[operation.TaskID]
	task.Status = moduleapi.TaskStatusRunning
	query.tasks[operation.TaskID] = task
	views, err = rollout.ListOperations(t.Context(), 10)
	if err != nil || len(views) != 0 {
		t.Fatalf("retry retained a stale failure projection: %#v err=%v", views, err)
	}
	stored, _ := rollout.operations.Get(t.Context(), operation.OperationID)
	if stored.Outcome != ExecutionOutcomePlanning || rollout.externalLaunches[operation.OperationID] == nil {
		t.Fatalf("read-only failure projection changed retry authority: %#v", stored)
	}
}

func TestFailedLaunchProjectionRequiresTheFrozenTaskOwnerAndStageIdentity(t *testing.T) {
	for _, field := range []string{"owner", "code", "version", "attempt", "result", "executor"} {
		t.Run(field, func(t *testing.T) {
			rollout, query, operation := failedLaunchProjectionFixture()
			task := query.tasks[operation.TaskID]
			switch field {
			case "owner":
				task.Owner.ID = "different-operation"
			case "code":
				code := "provider_operation_failed"
				task.FailureCode = &code
				query.stages[0].FailureCode = &code
			case "version":
				query.stages[0].ExecutionVersion = "unsupported-version"
			case "attempt":
				query.stages[0].Attempt = 0
			case "result":
				query.stages[1].Status = moduleapi.StageStatusRunning
			case "executor":
				query.stages[0].ExecutorType = "unrelated.executor"
			}
			query.tasks[operation.TaskID] = task
			views, err := rollout.ListOperations(t.Context(), 10)
			if err != nil || len(views) != 0 {
				t.Fatalf("unproven launch became history: %#v err=%v", views, err)
			}
		})
	}
}

func TestLaunchFailureQueriesPropagateErrorsAndRespectSettledControllerHistory(t *testing.T) {
	rollout, query, operation := failedLaunchProjectionFixture()
	query.err = errors.New("task query unavailable")
	if _, err := rollout.ListOperations(t.Context(), 10); !errors.Is(err, query.err) {
		t.Fatalf("task query failure=%v", err)
	}
	query.err = nil
	query.stageErr = errors.New("stage query unavailable")
	if _, err := rollout.ListOperations(t.Context(), 10); !errors.Is(err, query.stageErr) {
		t.Fatalf("stage query failure=%v", err)
	}
	query.err = errors.New("task query unavailable after settled receipt")
	operation.Outcome = ExecutionOutcomeSuccess
	if err := rollout.operations.Settle(t.Context(), operation); err != nil {
		t.Fatal(err)
	}
	views, err := rollout.ListOperations(t.Context(), 10)
	if err != nil || len(views) != 1 || views[0].Phase != RunnerPhaseSuccess || views[0].StateSource != "terminal_history" {
		t.Fatalf("settled Controller history was replaced: %#v err=%v", views, err)
	}
}
