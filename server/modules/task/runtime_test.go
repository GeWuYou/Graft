package task

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
	taskstore "graft/server/modules/task/store"
	"graft/server/modules/task/testschema"
)

func TestRuntimeExecutesSerialPlanAndCompletesTask(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(2, 1))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	for range 2 {
		if err := runtime.runOne(context.Background()); err != nil {
			t.Fatalf("run stage: %v", err)
		}
	}
	task := mustTask(t, repository, receipt.TaskID)
	if task.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("task status = %q, want success", task.Status)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatalf("list stages: %v", err)
	}
	for _, stage := range stages {
		if stage.Status != moduleapi.StageStatusSuccess || stage.Attempt != 1 {
			t.Fatalf("unexpected stage after serial execution: %#v", stage)
		}
	}
	if got := executor.calls(); got != 2 {
		t.Fatalf("executor calls = %d, want 2", got)
	}
}

func TestRuntimeRetryableStageReturnsToPendingForNextAttempt(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{errors: []error{errors.New("transient"), nil}}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 2))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run failed attempt: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatalf("list after retryable failure: %v", err)
	}
	if stages[0].Status != moduleapi.StageStatusPending || stages[0].Attempt != 1 {
		t.Fatalf("stage after retryable failure = %#v", stages[0])
	}
	time.Sleep(time.Millisecond)
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run retry attempt: %v", err)
	}
	task := mustTask(t, repository, receipt.TaskID)
	if task.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("task status = %q, want success", task.Status)
	}
}

func TestRepositoryRecoversRunningStageAsUnknownAndTaskNeedsAttention(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	claim, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found {
		t.Fatalf("claim running stage: found=%t err=%v", found, err)
	}
	if claim.Stage.Status != moduleapi.StageStatusRunning {
		t.Fatalf("claimed stage status = %q", claim.Stage.Status)
	}
	count, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC())
	if err != nil || count != 1 {
		t.Fatalf("recover interrupted stages = %d, %v", count, err)
	}
	assertInterruptedTaskRecovery(t, repository, receipt.TaskID)
	if count, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil || count != 0 {
		t.Fatalf("second recovery = %d, %v", count, err)
	}
	assertInterruptedTaskRecovery(t, repository, receipt.TaskID)
}

func TestRuntimeListTasksAuthorizesBeforePagination(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	for _, ownerID := range []string{"allowed-one", "denied", "allowed-two"} {
		input := testSubmitInput(1, 1)
		input.Owner.ID = ownerID
		if _, err := runtime.Submit(context.Background(), input); err != nil {
			t.Fatalf("submit %q: %v", ownerID, err)
		}
	}
	items, total, err := runtime.ListTasks(context.Background(), moduleapi.TaskOwner{Type: "test", ID: "allowed-one"}, 1, 0)
	if err != nil {
		t.Fatalf("list owner tasks: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].Owner.ID != "allowed-one" {
		t.Fatalf("owner task page = %#v total=%d", items, total)
	}
}

func TestRuntimeConvertsExecutorPanicsToFailedStage(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(panickingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].ExecutorType = panickingExecutorType
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run panicking executor: %v", err)
	}
	task := mustTask(t, repository, receipt.TaskID)
	if task.Status != moduleapi.TaskStatusFailed || task.FailureCode == nil || *task.FailureCode != errorCodeExecutor {
		t.Fatalf("task after executor panic = %#v", task)
	}
}

func TestRuntimeReturnsErrorWhenExecutorCancellationPanics(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(panickingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].ExecutorType = panickingExecutorType
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	claim, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found {
		t.Fatalf("claim stage: found=%t err=%v", found, err)
	}
	runtime.addRunning(receipt.TaskID, runningStage{executor: panickingExecutor{}, run: &stageRun{runtime: runtime, task: claim.Task, stage: claim.Stage}, cancel: func() {}})
	if err := runtime.Cancel(context.Background(), receipt.TaskID); err == nil {
		t.Fatal("cancel with panicking executor succeeded")
	}
}

func TestTaskCapabilitiesRequireOperationAuthorization(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	if err := runtime.RegisterTaskOwnerAuthorizer(capabilityAuthorizer{}); err != nil {
		t.Fatalf("register owner authorizer: %v", err)
	}
	task := moduleapi.TaskView{Owner: moduleapi.TaskOwner{Type: "capability-test", ID: "1"}, Status: moduleapi.TaskStatusNeedsAttention}
	stages := []moduleapi.TaskStageView{{Status: moduleapi.StageStatusUnknown}}
	capabilities := taskCapabilities(context.Background(), runtime, &moduleapi.CurrentUser{}, task, stages)
	if capabilities["cancel"] || capabilities["retry"] || !capabilities["download_log"] {
		t.Fatalf("capabilities = %#v", capabilities)
	}
}

func assertInterruptedTaskRecovery(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) {
	t.Helper()
	task := mustTask(t, repository, taskID)
	if task.Status != moduleapi.TaskStatusNeedsAttention || task.FailureCode == nil || *task.FailureCode != "runner_interrupted" {
		t.Fatalf("recovered task = %#v", task)
	}
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil {
		t.Fatalf("list recovered stages: %v", err)
	}
	if stages[0].Status != moduleapi.StageStatusUnknown || stages[0].FailureCode == nil || *stages[0].FailureCode != "runner_interrupted" {
		t.Fatalf("recovered stage = %#v", stages[0])
	}
	events, err := repository.ListEvents(context.Background(), taskID, 0, 10)
	if err != nil {
		t.Fatalf("list recovery event: %v", err)
	}
	if len(events) != 2 || events[1].Type != taskmodel.EventTypeRecoveryRequired {
		t.Fatalf("recovery events = %#v", events)
	}
}

func TestRuntimeRetriesOperatorApprovedFailedStage(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{errors: []error{errors.New("failed"), nil}}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 1))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run failed stage: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatalf("list failed stage: %v", err)
	}
	if err := runtime.RetryStage(context.Background(), receipt.TaskID, stages[0].ID); err != nil {
		t.Fatalf("request stage retry: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run approved retry: %v", err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("task after retry = %#v", task)
	}
}

func TestRuntimeRetryResolvesRecoveryEvent(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || !found {
		t.Fatalf("claim stage: found=%t err=%v", found, err)
	}
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("recover stage: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatalf("list stages: %v", err)
	}
	if err := runtime.RetryStage(context.Background(), receipt.TaskID, stages[0].ID); err != nil {
		t.Fatalf("retry recovered stage: %v", err)
	}
	events, err := repository.ListEvents(context.Background(), receipt.TaskID, 0, 10)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 4 || events[2].Type != taskmodel.EventTypeRecoveryResolved || events[3].Type != taskmodel.EventTypeRetryRequested {
		t.Fatalf("events after recovery retry = %#v", events)
	}
}

func TestRuntimeCancelsNeedsAttentionTask(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || !found {
		t.Fatalf("claim stage: found=%t err=%v", found, err)
	}
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("recover interrupted stage: %v", err)
	}
	if err := runtime.Cancel(context.Background(), receipt.TaskID); err != nil {
		t.Fatalf("cancel needs attention task: %v", err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("task after cancel = %#v", task)
	}
}

func newRuntimeForTest(t *testing.T) (*Runtime, *taskstore.SQLRepository) {
	t.Helper()
	dsn := fmt.Sprintf("file:task-runtime-%s?mode=memory&cache=private", t.Name())
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := testschema.CreateSQLite(db); err != nil {
		t.Fatalf("create runtime schema: %v", err)
	}
	repository, err := taskstore.NewSQLRepository(db, taskstore.SQLDialectSQLite)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return NewRuntime(repository), repository
}

func testSubmitInput(stageCount int, attempts int) moduleapi.SubmitTaskInput {
	stages := make([]moduleapi.StagePlan, 0, stageCount)
	for index := 0; index < stageCount; index++ {
		stages = append(stages, moduleapi.StagePlan{Key: fmt.Sprintf("stage-%d", index+1), ExecutorType: "test.executor", RetryPolicy: moduleapi.StageRetryPolicy{MaxAttempts: attempts}, RecoveryPolicy: moduleapi.StageRecoveryRetryIfIdempotent})
	}
	return moduleapi.SubmitTaskInput{Type: "test.runtime", Owner: moduleapi.TaskOwner{Type: "test", ID: fmt.Sprintf("owner-%d", time.Now().UnixNano())}, Plan: moduleapi.TaskPlan{Stages: stages}}
}

func mustTask(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) taskmodel.Task {
	t.Helper()
	task, err := repository.Get(context.Background(), taskID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	return task
}

type recordingExecutor struct {
	mu     sync.Mutex
	errors []error
	count  int
}

func (e *recordingExecutor) Type() moduleapi.StageExecutorType { return "test.executor" }

func (e *recordingExecutor) Execute(_ context.Context, _ moduleapi.StageRun) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	result := error(nil)
	if e.count < len(e.errors) {
		result = e.errors[e.count]
	}
	e.count++
	return result
}

func (e *recordingExecutor) Cancel(_ context.Context, _ moduleapi.StageRun) error { return nil }

func (e *recordingExecutor) calls() int {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.count
}

const panickingExecutorType moduleapi.StageExecutorType = "test.panicking"

type panickingExecutor struct{}

func (panickingExecutor) Type() moduleapi.StageExecutorType { return panickingExecutorType }

func (panickingExecutor) Execute(context.Context, moduleapi.StageRun) error { panic("execute") }

func (panickingExecutor) Cancel(context.Context, moduleapi.StageRun) error { panic("cancel") }

type capabilityAuthorizer struct{}

func (capabilityAuthorizer) OwnerType() string { return "capability-test" }

func (capabilityAuthorizer) AuthorizeTaskOwner(_ context.Context, _ *moduleapi.CurrentUser, action moduleapi.TaskOwnerAction, _ moduleapi.TaskOwner) error {
	if action == moduleapi.TaskOwnerActionView {
		return nil
	}
	return errors.New("operation access denied")
}
