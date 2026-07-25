package task

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"

	_ "github.com/mattn/go-sqlite3"
	"graft/server/internal/logger"

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
	if task.CurrentStageKey == nil || *task.CurrentStageKey != "stage-2" {
		t.Fatalf("current stage key = %v, want stage-2", task.CurrentStageKey)
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

func TestRuntimeSettlesExternalReceiptAfterCrashRecovery(t *testing.T) {
	t.Parallel()

	for _, testCase := range []struct {
		name        string
		outcome     moduleapi.ExternalReceiptOutcome
		failureCode string
		wantTask    moduleapi.TaskStatus
		wantStage   moduleapi.StageStatus
	}{
		{name: "success", outcome: moduleapi.ExternalReceiptOutcomeSuccess, wantTask: moduleapi.TaskStatusSuccess, wantStage: moduleapi.StageStatusSuccess},
		{name: "failed", outcome: moduleapi.ExternalReceiptOutcomeFailed, failureCode: "runner_failed", wantTask: moduleapi.TaskStatusFailed, wantStage: moduleapi.StageStatusFailed},
		{name: "needs attention", outcome: moduleapi.ExternalReceiptOutcomeNeedsAttention, failureCode: "healthz_failed", wantTask: moduleapi.TaskStatusNeedsAttention, wantStage: moduleapi.StageStatusUnknown},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			assertExternalReceiptSettlement(t, testCase.outcome, testCase.failureCode, testCase.wantTask, testCase.wantStage)
		})
	}
}

func TestRuntimeSubmitReplaysMatchingIdempotencyKeyAndRejectsChangedSubmission(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.RequestedBy = 42
	input.IdempotencyKey = "docker-image-pull-001"
	input.Input = json.RawMessage(`{"image":"redis:7","options":{"quiet":true}}`)
	first, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	replayed := input
	replayed.Input = json.RawMessage(`{"options":{"quiet":true},"image":"redis:7"}`)
	second, err := runtime.Submit(context.Background(), replayed)
	if err != nil {
		t.Fatalf("replay submit: %v", err)
	}
	if first != second {
		t.Fatalf("replayed receipt = %#v, want %#v", second, first)
	}
	stored := mustTask(t, repository, first.TaskID)
	if stored.IdempotencyKeyHash == nil || stored.SubmissionFingerprint == nil || *stored.IdempotencyKeyHash == input.IdempotencyKey {
		t.Fatalf("idempotency submission storage = %#v", stored)
	}
	conflict := input
	conflict.Input = json.RawMessage(`{"image":"redis:8"}`)
	if _, err := runtime.Submit(context.Background(), conflict); !errors.Is(err, moduleapi.ErrTaskSubmissionConflict) {
		t.Fatalf("changed idempotency submission error = %v", err)
	}
}

func TestRuntimeSubmitWithoutIdempotencyKeyRemainsNonIdempotent(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	first, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("first submit: %v", err)
	}
	second, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("second submit: %v", err)
	}
	if first.TaskID == second.TaskID {
		t.Fatalf("legacy submissions unexpectedly replayed: %#v", first)
	}
}

func assertExternalReceiptSettlement(t *testing.T, outcome moduleapi.ExternalReceiptOutcome, failureCode string, wantTask moduleapi.TaskStatus, wantStage moduleapi.StageStatus) {
	t.Helper()
	runtime, repository, receipt := recoveredExternalReceiptTask(t)
	settlement, err := runtime.SettleExternalReceipt(context.Background(), externalReceipt(receipt.TaskID, outcome, failureCode))
	if err != nil || settlement.Status != wantTask || settlement.Idempotent {
		t.Fatalf("settlement = %#v err=%v", settlement, err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != wantTask {
		t.Fatalf("task status = %q, want %q", task.Status, wantTask)
	}
	assertExternalReceiptStageAndEvent(t, repository, receipt.TaskID, wantStage, 3)
}

func TestRuntimeRejectsMismatchedOrConflictingExternalReceiptAndReplaysExactReceipt(t *testing.T) {
	t.Parallel()
	runtime, repository, receipt := claimedExternalReceiptTask(t)
	assertMismatchedExternalReceiptRejected(t, runtime, repository, receipt)
	assertExternalReceiptReplayRules(t, runtime, repository, receipt)
}

func assertMismatchedExternalReceiptRejected(t *testing.T, runtime *Runtime, repository *taskstore.SQLRepository, receipt moduleapi.TaskReceipt) {
	t.Helper()
	mismatch := externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, "")
	mismatch.OperationID = "unexpected-operation"
	if _, err := runtime.SettleExternalReceipt(context.Background(), mismatch); err == nil {
		t.Fatal("mismatched receipt settled")
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("mismatched receipt changed task: %#v", task)
	}
}

func assertExternalReceiptReplayRules(t *testing.T, runtime *Runtime, repository *taskstore.SQLRepository, receipt moduleapi.TaskReceipt) {
	t.Helper()
	value := externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, "")
	first, err := runtime.SettleExternalReceipt(context.Background(), value)
	if err != nil || first.Idempotent {
		t.Fatalf("first settlement = %#v err=%v", first, err)
	}
	second, err := runtime.SettleExternalReceipt(context.Background(), value)
	if err != nil || !second.Idempotent || second.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("replayed settlement = %#v err=%v", second, err)
	}
	conflict := value
	conflict.IntegritySHA256 = strings.Repeat("b", 64)
	if _, err := runtime.SettleExternalReceipt(context.Background(), conflict); !errors.Is(err, taskstore.ErrStateConflict) {
		t.Fatalf("conflicting replay error = %v", err)
	}
	events, err := repository.ListEvents(context.Background(), receipt.TaskID, 0, 10)
	if err != nil || len(events) != 2 || events[1].Type != taskmodel.EventTypeExternalReceiptSettled {
		t.Fatalf("replay events = %#v err=%v", events, err)
	}
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
	items, total, err := runtime.ListTasks(context.Background(), moduleapi.TaskListFilter{Owner: moduleapi.TaskOwner{Type: "test", ID: "allowed-one"}}, 1, 0)
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
	core, logs := observer.New(zap.ErrorLevel)
	runtime.SetAppLogger(logger.NewAppLogger(zap.New(core)))
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
	if task.CurrentStageKey == nil || *task.CurrentStageKey != "stage-1" {
		t.Fatalf("failed task current stage key = %v, want stage-1", task.CurrentStageKey)
	}
	if logs.Len() != 1 {
		t.Fatalf("expected one task panic log, got %d", logs.Len())
	}
	fields := logs.All()[0].ContextMap()
	if fields["task_id"] != uint64(receipt.TaskID) || fields["executor_type"] != string(panickingExecutorType) {
		t.Fatalf("expected task panic context, got %#v", fields)
	}
	if stacktrace, ok := fields["stacktrace"].(string); !ok || !strings.Contains(stacktrace, "panickingExecutor.Execute") {
		t.Fatalf("expected executor panic stacktrace, got %#v", fields["stacktrace"])
	}
}

func TestRuntimeWorkerIterationRecoversAndContinuesAfterPanic(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	runtime.repository = &panicOnceRepository{Repository: repository}
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	if _, err := runtime.Submit(context.Background(), testSubmitInput(1, 1)); err != nil {
		t.Fatalf("submit task: %v", err)
	}

	if err := runtime.runWorkerIteration(context.Background()); err == nil || !strings.Contains(err.Error(), "task worker panicked") {
		t.Fatalf("first worker iteration error = %v, want recovered panic", err)
	}
	if err := runtime.runWorkerIteration(context.Background()); err != nil {
		t.Fatalf("second worker iteration: %v", err)
	}
	if calls := executor.calls(); calls != 1 {
		t.Fatalf("executor calls after worker panic = %d, want 1", calls)
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
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusCancelled || task.CurrentStageKey == nil || *task.CurrentStageKey != "stage-1" {
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

func externalReceiptSubmitInput() moduleapi.SubmitTaskInput {
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalReceipt = &moduleapi.ExternalReceiptExpectation{Protocol: "compose-runner/v1", OperationID: "operation-123"}
	return input
}

func externalReceipt(taskID uint64, outcome moduleapi.ExternalReceiptOutcome, failureCode string) moduleapi.ExternalTaskReceipt {
	return moduleapi.ExternalTaskReceipt{TaskID: taskID, ExecutorType: "test.executor", Protocol: "compose-runner/v1", OperationID: "operation-123", Outcome: outcome, FailureCode: failureCode, IntegritySHA256: strings.Repeat("a", 64)}
}

func claimedExternalReceiptTask(t *testing.T) (*Runtime, *taskstore.SQLRepository, moduleapi.TaskReceipt) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), externalReceiptSubmitInput())
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || !found {
		t.Fatalf("claim external stage: found=%t err=%v", found, err)
	}
	return runtime, repository, receipt
}

func recoveredExternalReceiptTask(t *testing.T) (*Runtime, *taskstore.SQLRepository, moduleapi.TaskReceipt) {
	t.Helper()
	runtime, repository, receipt := claimedExternalReceiptTask(t)
	if _, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil {
		t.Fatalf("recover interrupted stage: %v", err)
	}
	return runtime, repository, receipt
}

func assertExternalReceiptStageAndEvent(t *testing.T, repository *taskstore.SQLRepository, taskID uint64, wantStage moduleapi.StageStatus, wantEventCount int) {
	t.Helper()
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil || len(stages) != 1 || stages[0].Status != wantStage {
		t.Fatalf("settled stages = %#v err=%v", stages, err)
	}
	events, err := repository.ListEvents(context.Background(), taskID, 0, 10)
	if err != nil || len(events) != wantEventCount || events[len(events)-1].Type != taskmodel.EventTypeExternalReceiptSettled {
		t.Fatalf("settlement events = %#v err=%v", events, err)
	}
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

type panicOnceRepository struct {
	taskstore.Repository
	mu       sync.Mutex
	panicked bool
}

func (r *panicOnceRepository) ClaimNextStage(ctx context.Context, now time.Time) (taskstore.StageClaim, bool, error) {
	r.mu.Lock()
	if !r.panicked {
		r.panicked = true
		r.mu.Unlock()
		panic("claim next stage")
	}
	r.mu.Unlock()
	return r.Repository.ClaimNextStage(ctx, now)
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
