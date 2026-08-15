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

func TestRuntimeStopAcceptsNilContext(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	if err := runtime.Start(context.Background()); err != nil {
		t.Fatalf("start runtime: %v", err)
	}
	if err := runtime.Stop(nil); err != nil {
		t.Fatalf("stop runtime with nil context: %v", err)
	}
}

func TestRepositoryClaimsCoordinatedLegsWithoutSerialBlocking(t *testing.T) {
	t.Parallel()
	_, repository := newRuntimeForTest(t)
	created, err := createCoordinatedTestTask(repository, "amd64", "arm64")
	if err != nil {
		t.Fatalf("create coordinated task: %v", err)
	}
	first, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found || first.Task.ID != created.ID {
		t.Fatalf("claim first coordinated leg: claim=%#v found=%t err=%v", first, found, err)
	}
	second, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found || second.Task.ID != created.ID || second.Stage.ID == first.Stage.ID {
		t.Fatalf("claim second coordinated leg: claim=%#v found=%t err=%v", second, found, err)
	}
}

func TestRepositoryRejectsDuplicateCoordinatedLegIdentity(t *testing.T) {
	t.Parallel()
	_, repository := newRuntimeForTest(t)
	_, err := createCoordinatedTestTask(repository, "linux-amd64", "linux-amd64")
	if !errors.Is(err, taskstore.ErrInvalidInput) {
		t.Fatalf("duplicate coordinated leg error = %v, want %v", err, taskstore.ErrInvalidInput)
	}
}

func TestRuntimeCancelsMultipleUntrackedCoordinatedLegs(t *testing.T) {
	t.Parallel()
	_, repository := newRuntimeForTest(t)
	created, err := createCoordinatedTestTask(repository, "amd64", "arm64")
	if err != nil {
		t.Fatalf("create coordinated task: %v", err)
	}
	for range 2 {
		if _, found, claimErr := repository.ClaimNextStage(context.Background(), time.Now().UTC()); claimErr != nil || !found {
			t.Fatalf("claim coordinated leg: found=%t err=%v", found, claimErr)
		}
	}
	runtime := NewRuntime(repository)
	if err := runtime.Cancel(context.Background(), created.ID); err != nil {
		t.Fatalf("cancel untracked coordinated task: %v", err)
	}
	if task := mustTask(t, repository, created.ID); task.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("task status = %q, want cancelled", task.Status)
	}
	stages, err := repository.ListStages(context.Background(), created.ID)
	if err != nil {
		t.Fatalf("list cancelled coordinated legs: %v", err)
	}
	for _, stage := range stages {
		if stage.Status != moduleapi.StageStatusCancelled {
			t.Fatalf("stage status = %q, want cancelled", stage.Status)
		}
	}
}

func createCoordinatedTestTask(repository *taskstore.SQLRepository, legIDs ...string) (taskmodel.Task, error) {
	stages := make([]taskmodel.Stage, 0, len(legIDs))
	for index, legID := range legIDs {
		stages = append(stages, taskmodel.Stage{Key: fmt.Sprintf("build-%d", index+1), Sequence: index + 1, ExecutorType: "test.executor", CoordinationGroup: "build-platforms", LegID: legID, Status: moduleapi.StageStatusPending, MaxAttempts: 1, Input: json.RawMessage(`{}`), RecoveryPolicy: moduleapi.StageRecoveryManualReconcile, Result: json.RawMessage(`{}`)})
	}
	created, _, _, err := repository.Create(context.Background(), taskstore.CreateInput{Task: taskmodel.Task{Type: "test.coordinated", Owner: moduleapi.TaskOwner{Type: "test", ID: fmt.Sprintf("coordinated-%d", time.Now().UnixNano())}, Status: moduleapi.TaskStatusReady, Input: json.RawMessage(`{}`), Metadata: json.RawMessage(`{}`), Plan: json.RawMessage(`{}`), State: json.RawMessage(`{}`)}, Stages: stages})
	return created, err
}

func TestRuntimeSubmissionCannotBeClaimedBeforeMaterialization(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	submission, err := runtime.BeginSubmission(context.Background(), moduleapi.BeginTaskSubmissionInput{Task: input, Policy: testSubmissionPolicy()})
	if err != nil {
		t.Fatalf("begin submission: %v", err)
	}
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || found {
		t.Fatalf("claim before materialization = found:%t err:%v", found, err)
	}
	receipt, err := runtime.MaterializeSubmission(context.Background(), submission, input, taskSubmissionWriterFunc(func(_ context.Context, _ *sql.Tx, got moduleapi.TaskSubmission) (string, error) {
		if got.TaskID == nil || *got.TaskID == 0 {
			t.Fatalf("writer task id = %v", got.TaskID)
		}
		return "snapshot:test", nil
	}))
	if err != nil {
		t.Fatalf("materialize task: %v", err)
	}
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || !found {
		t.Fatalf("claim after materialization = found:%t err:%v", found, err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("materialized task status = %q", task.Status)
	}
}

func TestRuntimeDiscardedSubmissionReleasesOwnerForNewSubmission(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	submission, err := runtime.BeginSubmission(context.Background(), moduleapi.BeginTaskSubmissionInput{Task: input, Policy: testSubmissionPolicy()})
	if err != nil {
		t.Fatalf("begin submission: %v", err)
	}
	if err := runtime.DiscardSubmission(context.Background(), submission, "snapshot_failed"); err != nil {
		t.Fatalf("discard task submission: %v", err)
	}
	stored, err := runtime.GetSubmission(context.Background(), submission.Submission.ID)
	if err != nil || stored.State != moduleapi.TaskSubmissionStateDiscarded {
		t.Fatalf("discarded submission = %#v err=%v", stored, err)
	}
	if _, err := runtime.Submit(context.Background(), input); err != nil {
		t.Fatalf("submit after discard: %v", err)
	}
}

type taskSubmissionWriterFunc func(context.Context, *sql.Tx, moduleapi.TaskSubmission) (string, error)

func (f taskSubmissionWriterFunc) MaterializeTaskSubmission(ctx context.Context, tx *sql.Tx, submission moduleapi.TaskSubmission) (string, error) {
	return f(ctx, tx, submission)
}

func testSubmissionPolicy() moduleapi.TaskSubmissionPolicy {
	return moduleapi.TaskSubmissionPolicy{LeaseTTL: time.Minute, AbsoluteDeadline: 5 * time.Minute, RenewBefore: 15 * time.Second, AllowRenew: true, PrerequisiteKind: "test.snapshot"}
}

func TestRuntimeUpdatesCurrentStageWhenClaimingLaterStage(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(2, 1))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("complete first stage: %v", err)
	}
	claim, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC())
	if err != nil || !found || claim.Stage.Key != "stage-2" {
		t.Fatalf("claim second stage: claim=%#v found=%t err=%v", claim, found, err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.CurrentStageKey == nil || *task.CurrentStageKey != "stage-2" {
		t.Fatalf("task current stage after second claim = %v, want stage-2", task.CurrentStageKey)
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

func TestRuntimeStructuredNeedsAttentionFailureSkipsAutomaticRetry(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{errors: []error{&moduleapi.ExecutionFailure{
		Code:        "credential_cleanup_unverified",
		Class:       moduleapi.ExecutionFailureClassInternal,
		Disposition: moduleapi.RecoveryDispositionNeedsAttention,
		Cause:       errors.New("credential cleanup could not be verified"),
	}}}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 2))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run structured failure: %v", err)
	}
	task := mustTask(t, repository, receipt.TaskID)
	if task.Status != moduleapi.TaskStatusNeedsAttention || task.FailureCode == nil || *task.FailureCode != "credential_cleanup_unverified" {
		t.Fatalf("task after cleanup failure = %#v", task)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil {
		t.Fatalf("list stages: %v", err)
	}
	if len(stages) != 1 || stages[0].Status != moduleapi.StageStatusUnknown || stages[0].Attempt != 1 {
		t.Fatalf("stage after cleanup failure = %#v", stages)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run after needs attention: %v", err)
	}
	if calls := executor.calls(); calls != 1 {
		t.Fatalf("executor calls = %d, want 1", calls)
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

func TestRepositoryCancelsInterruptedStageWhenCancellationWasRequested(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Owner = moduleapi.TaskOwner{Type: "application", ID: "app-recovery-cancel"}
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	claimTaskStage(t, repository)
	if _, err := repository.RequestCancellation(context.Background(), receipt.TaskID, time.Now().UTC()); err != nil {
		t.Fatalf("request cancellation: %v", err)
	}
	if count, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil || count != 1 {
		t.Fatalf("recover requested cancellation = %d, %v", count, err)
	}
	assertInterruptedCancellation(t, repository, receipt.TaskID)
	assertTaskCanBeResubmitted(t, runtime, input)
}

func assertInterruptedCancellation(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) {
	t.Helper()
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("task after interrupted cancellation = %#v", task)
	}
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil || len(stages) != 1 || stages[0].Status != moduleapi.StageStatusCancelled {
		t.Fatalf("stages after interrupted cancellation = %#v err=%v", stages, err)
	}
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

func TestRuntimeLeavesUnregisteredExternalReceiptStageRunningUntilSettlement(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), externalReceiptSubmitInput())
	if err != nil {
		t.Fatalf("submit unregistered external stage: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("claim external stage: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil || len(stages) != 1 || stages[0].Status != moduleapi.StageStatusRunning {
		t.Fatalf("external stage after worker claim = %#v err=%v", stages, err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("external task after worker claim = %#v", task)
	}
	settlement, err := runtime.SettleExternalReceipt(context.Background(), externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil || settlement.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("settle external receipt = %#v err=%v", settlement, err)
	}
}

func TestRuntimePreservesClaimedExternalReceiptForLateSettlementAfterCancel(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), externalReceiptSubmitInput())
	if err != nil {
		t.Fatalf("submit unregistered external stage: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("claim external stage: %v", err)
	}
	if err := runtime.Cancel(context.Background(), receipt.TaskID); err != nil {
		t.Fatalf("cancel external stage: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil || len(stages) != 1 || stages[0].Status != moduleapi.StageStatusRunning {
		t.Fatalf("external stage after cancel = %#v err=%v", stages, err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("external task after cancel = %#v", task)
	}
	settlement, err := runtime.SettleExternalReceipt(context.Background(), externalReceipt(receipt.TaskID, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil || settlement.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("late external receipt settlement = %#v err=%v", settlement, err)
	}
}

func TestRuntimeCancelsUntrackedRunningStageAndReleasesActiveOwner(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Owner = moduleapi.TaskOwner{Type: "application", ID: "app-untracked-cancel"}
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	claimTaskStage(t, repository)
	assertActiveOwnerBusy(t, runtime, input)
	if err := runtime.Cancel(context.Background(), receipt.TaskID); err != nil {
		t.Fatalf("cancel untracked running task: %v", err)
	}
	assertUntrackedCancellation(t, repository, receipt.TaskID)
	assertTaskCanBeResubmitted(t, runtime, input)
}

func TestRuntimeCancelsUntrackedRunningStageWithDistinctTaskAndStageDurations(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	now := time.Now().UTC()
	capturingRepository := &captureCancellationRepository{
		Repository:     repository,
		taskStartedAt:  now.Add(-10 * time.Second),
		stageStartedAt: now.Add(-time.Second),
	}
	runtime.repository = capturingRepository
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 1))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	claimTaskStage(t, repository)
	if err := runtime.Cancel(context.Background(), receipt.TaskID); err != nil {
		t.Fatalf("cancel untracked running task: %v", err)
	}
	if capturingRepository.taskDurationMS == nil || capturingRepository.stageDurationMS == nil {
		t.Fatal("untracked cancellation did not receive both durations")
	}
	if *capturingRepository.taskDurationMS < 9_000 || *capturingRepository.stageDurationMS < 900 || *capturingRepository.taskDurationMS-*capturingRepository.stageDurationMS < 7_000 {
		t.Fatalf("durations must use independent start times: task=%d stage=%d", *capturingRepository.taskDurationMS, *capturingRepository.stageDurationMS)
	}
}

func claimTaskStage(t *testing.T, repository *taskstore.SQLRepository) {
	t.Helper()
	if _, found, err := repository.ClaimNextStage(context.Background(), time.Now().UTC()); err != nil || !found {
		t.Fatalf("claim stage: found=%t err=%v", found, err)
	}
}

func assertActiveOwnerBusy(t *testing.T, runtime *Runtime, input moduleapi.SubmitTaskInput) {
	t.Helper()
	if _, err := runtime.Submit(context.Background(), input); !errors.Is(err, moduleapi.ErrTaskOwnerBusy) {
		t.Fatalf("active owner submission error = %v", err)
	}
}

func assertUntrackedCancellation(t *testing.T, repository *taskstore.SQLRepository, taskID uint64) {
	t.Helper()
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("task after untracked cancellation = %#v", task)
	}
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil || len(stages) != 1 || stages[0].Status != moduleapi.StageStatusCancelled {
		t.Fatalf("stages after untracked cancellation = %#v err=%v", stages, err)
	}
	events, err := repository.ListEvents(context.Background(), taskID, 0, 10)
	if err != nil || len(events) != 3 || events[1].Type != taskmodel.EventTypeCancelRequested || events[2].Type != taskmodel.EventTypeCancelled {
		t.Fatalf("untracked cancellation events = %#v err=%v", events, err)
	}
}

func assertTaskCanBeResubmitted(t *testing.T, runtime *Runtime, input moduleapi.SubmitTaskInput) {
	t.Helper()
	if _, err := runtime.Submit(context.Background(), input); err != nil {
		t.Fatalf("submit after untracked cancellation: %v", err)
	}
}

func TestRuntimeRejectsUnregisteredOrdinaryStageExecutor(t *testing.T) {
	t.Parallel()
	runtime, _ := newRuntimeForTest(t)
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].ExecutorType = "platform.update.compose-runner"
	if _, err := runtime.Submit(context.Background(), input); err == nil || !strings.Contains(err.Error(), "unregistered stage executor") {
		t.Fatalf("submit unregistered ordinary stage error = %v", err)
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
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("complete first task: %v", err)
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
	_, err := runtime.Submit(context.Background(), testSubmitInput(1, 1))
	if err != nil {
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

func TestRuntimeRemovesRunningStageWhenFinishClaimPanics(t *testing.T) {
	t.Parallel()
	runtime, repository := newRuntimeForTest(t)
	runtime.repository = &panicOnGetRepository{Repository: repository}
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	receipt, err := runtime.Submit(context.Background(), testSubmitInput(1, 1))
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runWorkerIteration(context.Background()); err == nil || !strings.Contains(err.Error(), "task worker panicked") {
		t.Fatalf("worker iteration error = %v, want recovered panic", err)
	}
	runtime.mu.RLock()
	_, running := runtime.running[receipt.TaskID]
	runtime.mu.RUnlock()
	if running {
		t.Fatal("running stage remained registered after finishClaim panic")
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

func TestRuntimeRejectsCoordinatedPlanUntilDistributedLegCapabilityExists(t *testing.T) {
	runtime, _ := newRuntimeForTest(t)
	input := testSubmitInput(1, 1)
	input.Plan.Coordination = &moduleapi.CoordinatedTaskPlan{Version: "build-legs/v1", AggregateStageKey: "build-aggregate", Legs: []moduleapi.CoordinatedLegPlan{{ID: "amd64", Platform: "linux/amd64", BuilderInstanceID: "builder-a", RuntimeTargetID: 1}, {ID: "arm64", Platform: "linux/arm64", BuilderInstanceID: "builder-b", RuntimeTargetID: 2}}}
	if _, err := runtime.Submit(context.Background(), input); !errors.Is(err, moduleapi.ErrCoordinatedTaskUnsupported) {
		t.Fatalf("error = %v, want distributed leg capability gate", err)
	}
}

func TestRuntimeSubmitCoordinatedMaterializesParallelLegStages(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Coordination = &moduleapi.CoordinatedTaskPlan{Version: "build-legs/v1", AggregateStageKey: "build-aggregate", Legs: []moduleapi.CoordinatedLegPlan{{ID: "amd64", Platform: "linux/amd64", BuilderInstanceID: "builder-a", RuntimeTargetID: 1}, {ID: "arm64", Platform: "linux/arm64", BuilderInstanceID: "builder-b", RuntimeTargetID: 2}}}
	receipt, err := runtime.SubmitCoordinated(context.Background(), input)
	if err != nil {
		t.Fatalf("SubmitCoordinated: %v", err)
	}
	stages, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil || len(stages) != 3 {
		t.Fatalf("coordinated stages = %#v err=%v", stages, err)
	}
	for _, stage := range stages[:2] {
		if stage.CoordinationGroup != "build-aggregate" || stage.LegID == "" {
			t.Fatalf("coordinated stage = %#v", stage)
		}
	}
	aggregate := stages[2]
	if aggregate.Key != "build-aggregate" || aggregate.CoordinationGroup != "" || aggregate.LegID != "" {
		t.Fatalf("aggregate stage = %#v", aggregate)
	}
}

func TestRuntimeCompletesCoordinatedTaskOnlyAfterAggregateStage(t *testing.T) {
	runtime, repository := newRuntimeForTest(t)
	executor := &recordingExecutor{}
	if err := runtime.RegisterStageExecutor(executor); err != nil {
		t.Fatalf("register executor: %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Coordination = &moduleapi.CoordinatedTaskPlan{Version: "build-legs/v1", AggregateStageKey: "build-aggregate", Legs: []moduleapi.CoordinatedLegPlan{{ID: "amd64", Platform: "linux/amd64", BuilderInstanceID: "builder-a", RuntimeTargetID: 1}, {ID: "arm64", Platform: "linux/arm64", BuilderInstanceID: "builder-b", RuntimeTargetID: 2}}}
	receipt, err := runtime.SubmitCoordinated(context.Background(), input)
	if err != nil {
		t.Fatalf("submit coordinated task: %v", err)
	}
	for range 2 {
		if err := runtime.runOne(context.Background()); err != nil {
			t.Fatalf("run coordinated leg: %v", err)
		}
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("task completed before aggregate stage: %#v", task)
	}
	if executor.calls() != 2 {
		t.Fatalf("leg calls = %d, want 2", executor.calls())
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run aggregate stage: %v", err)
	}
	if task := mustTask(t, repository, receipt.TaskID); task.Status != moduleapi.TaskStatusSuccess || executor.calls() != 3 {
		t.Fatalf("aggregate completion = %#v calls=%d", task, executor.calls())
	}
}

func externalReceiptSubmitInput() moduleapi.SubmitTaskInput {
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].ExecutorType = "platform.update.compose-runner"
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalReceipt = &moduleapi.ExternalReceiptExpectation{Protocol: "compose-runner/v2", OperationID: "operation-123"}
	return input
}

func externalReceipt(taskID uint64, outcome moduleapi.ExternalReceiptOutcome, failureCode string) moduleapi.ExternalTaskReceipt {
	return moduleapi.ExternalTaskReceipt{TaskID: taskID, ExecutorType: "platform.update.compose-runner", Protocol: "compose-runner/v2", OperationID: "operation-123", Outcome: outcome, FailureCode: failureCode, IntegritySHA256: strings.Repeat("a", 64)}
}

func claimedExternalReceiptTask(t *testing.T) (*Runtime, *taskstore.SQLRepository, moduleapi.TaskReceipt) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	receipt, err := runtime.Submit(context.Background(), externalReceiptSubmitInput())
	if err != nil {
		t.Fatalf("submit task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("claim external stage: %v", err)
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

type panicOnGetRepository struct {
	taskstore.Repository
}

func (r *panicOnGetRepository) Get(context.Context, uint64) (taskmodel.Task, error) {
	panic("finish claim get")
}

type captureCancellationRepository struct {
	taskstore.Repository
	taskStartedAt   time.Time
	stageStartedAt  time.Time
	stageDurationMS *int64
	taskDurationMS  *int64
}

func (r *captureCancellationRepository) RequestCancellation(ctx context.Context, taskID uint64, requestedAt time.Time) (taskmodel.Task, error) {
	task, err := r.Repository.RequestCancellation(ctx, taskID, requestedAt)
	if err == nil {
		task.StartedAt = &r.taskStartedAt
	}
	return task, err
}

func (r *captureCancellationRepository) ListStages(ctx context.Context, taskID uint64) ([]taskmodel.Stage, error) {
	stages, err := r.Repository.ListStages(ctx, taskID)
	if err == nil && len(stages) == 1 {
		stages[0].StartedAt = &r.stageStartedAt
	}
	return stages, err
}

func (r *captureCancellationRepository) CancelUntrackedRunningStage(ctx context.Context, taskID uint64, stageID uint64, finishedAt time.Time, stageDurationMS *int64, taskDurationMS *int64) error {
	r.stageDurationMS = stageDurationMS
	r.taskDurationMS = taskDurationMS
	return r.Repository.CancelUntrackedRunningStage(ctx, taskID, stageID, finishedAt, stageDurationMS, taskDurationMS)
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
