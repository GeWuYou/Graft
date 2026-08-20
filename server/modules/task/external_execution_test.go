package task

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
	taskstore "graft/server/modules/task/store"
)

func TestExternalExecutionSuccessAdvancesNonFinalStage(t *testing.T) {
	runtime, repository, taskID := newExternalExecutionPipeline(t)
	lease := settleFirstExternalExecutionStage(t, runtime, taskID)
	assertExternalExecutionAdvanced(t, repository, taskID, lease.StageID)
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run final local stage: %v", err)
	}
	if task := mustTask(t, repository, taskID); task.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("task after final stage = %#v", task)
	}
}

func newExternalExecutionPipeline(t *testing.T) (*Runtime, *taskstore.SQLRepository, uint64) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
		t.Fatalf("register final executor: %v", err)
	}
	input := testSubmitInput(2, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("operation-stage-1")
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit external execution task: %v", err)
	}
	if err := runtime.runOne(context.Background()); err != nil {
		t.Fatalf("run local worker with external stage pending: %v", err)
	}
	pending, err := repository.ListStages(context.Background(), receipt.TaskID)
	if err != nil || len(pending) != 2 || pending[0].Status != moduleapi.StageStatusPending || pending[0].Attempt != 0 {
		t.Fatalf("external stage after local worker = %#v err=%v", pending, err)
	}
	return runtime, repository, receipt.TaskID
}

func settleFirstExternalExecutionStage(t *testing.T, runtime *Runtime, taskID uint64) moduleapi.ExternalExecutionLease {
	t.Helper()
	lease := claimTestExternalExecution(t, runtime)
	if lease.TaskID != taskID || lease.StageID == 0 || lease.Attempt != 1 || lease.FenceToken == "" {
		t.Fatalf("claimed lease = %#v", lease)
	}
	if err := runtime.AppendExternalExecutionLogs(context.Background(), moduleapi.ExternalExecutionLogBatch{
		Handle:  externalExecutionHandle(lease),
		Entries: []moduleapi.TaskLogEntry{{Stream: "stdout", Level: "info", Line: "compose operation started"}},
	}); err != nil {
		t.Fatalf("append external logs: %v", err)
	}
	settlement, err := runtime.SettleExternalExecution(context.Background(), testExternalExecutionReceipt(lease, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil {
		t.Fatalf("settle external execution: %v", err)
	}
	if settlement.Status != moduleapi.TaskStatusRunning {
		t.Fatalf("non-final settlement status = %s", settlement.Status)
	}
	return lease
}

func assertExternalExecutionAdvanced(t *testing.T, repository *taskstore.SQLRepository, taskID uint64, stageID uint64) {
	t.Helper()
	stages, err := repository.ListStages(context.Background(), taskID)
	if err != nil || len(stages) != 2 || stages[0].Status != moduleapi.StageStatusSuccess || stages[1].Status != moduleapi.StageStatusPending {
		t.Fatalf("stages after non-final receipt = %#v err=%v", stages, err)
	}
	logs, err := repository.ListLogs(context.Background(), taskID, 0, 10)
	if err != nil || len(logs) != 1 || logs[0].StageID == nil || *logs[0].StageID != stageID {
		t.Fatalf("external execution logs = %#v err=%v", logs, err)
	}
}

func TestExternalExecutionReceiptReplayAndFenceConflict(t *testing.T) {
	runtime, _, lease := claimedExternalExecutionTask(t, 1)
	receipt := testExternalExecutionReceipt(lease, moduleapi.ExternalReceiptOutcomeSuccess, "")
	first, err := runtime.SettleExternalExecution(context.Background(), receipt)
	if err != nil || first.Idempotent {
		t.Fatalf("first settlement = %#v err=%v", first, err)
	}
	second, err := runtime.SettleExternalExecution(context.Background(), receipt)
	if err != nil || !second.Idempotent {
		t.Fatalf("replayed settlement = %#v err=%v", second, err)
	}
	conflict := receipt
	conflict.IntegritySHA256 = strings.Repeat("b", 64)
	if _, err := runtime.SettleExternalExecution(context.Background(), conflict); !errors.Is(err, taskstore.ErrStateConflict) {
		t.Fatalf("conflicting receipt error = %v", err)
	}
	wrongFence := receipt
	wrongFence.Handle.FenceToken = strings.Repeat("f", 64)
	if _, err := runtime.SettleExternalExecution(context.Background(), wrongFence); !errors.Is(err, taskstore.ErrStateConflict) {
		t.Fatalf("wrong fence error = %v", err)
	}
}

func TestExternalExecutionRenewalObservesCancellationAndStopsRemainingStages(t *testing.T) {
	runtime, repository, lease := claimedExternalExecutionTask(t, 2)
	if err := runtime.Cancel(context.Background(), lease.TaskID); err != nil {
		t.Fatalf("request external execution cancellation: %v", err)
	}
	renewed, err := runtime.RenewExternalExecution(context.Background(), externalExecutionHandle(lease))
	if err != nil || !renewed.CancellationRequested {
		t.Fatalf("renewed lease = %#v err=%v", renewed, err)
	}
	persisted, cancelRequested, err := repository.GetExternalExecutionLease(context.Background(), lease.ID)
	if err != nil || !cancelRequested || persisted.CancelObservedAt == nil {
		t.Fatalf("persisted cancellation observation = %#v requested=%t err=%v", persisted, cancelRequested, err)
	}
	settlement, err := runtime.SettleExternalExecution(context.Background(), testExternalExecutionReceipt(lease, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil || settlement.Status != moduleapi.TaskStatusCancelled {
		t.Fatalf("cancelled settlement = %#v err=%v", settlement, err)
	}
	stages, err := repository.ListStages(context.Background(), lease.TaskID)
	if err != nil || len(stages) != 2 || stages[0].Status != moduleapi.StageStatusSuccess || stages[1].Status != moduleapi.StageStatusCancelled {
		t.Fatalf("cancelled stages = %#v err=%v", stages, err)
	}
}

func TestExternalExecutionLeaseSurvivesRestartUntilExpiry(t *testing.T) {
	runtime, repository, lease := claimedExternalExecutionTask(t, 1)
	if count, err := repository.RecoverInterruptedStages(context.Background(), time.Now().UTC()); err != nil || count != 0 {
		t.Fatalf("recover with active lease = count:%d err:%v", count, err)
	}
	stages, err := repository.ListStages(context.Background(), lease.TaskID)
	if err != nil || len(stages) != 1 || stages[0].Status != moduleapi.StageStatusRunning {
		t.Fatalf("active leased stage = %#v err=%v", stages, err)
	}
	if count, err := repository.ExpireExternalExecutionLeases(context.Background(), lease.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 1 {
		t.Fatalf("expire external lease = count:%d err:%v", count, err)
	}
	if task := mustTask(t, repository, lease.TaskID); task.Status != moduleapi.TaskStatusNeedsAttention {
		t.Fatalf("task after lease expiry = %#v", task)
	}
	stages, err = repository.ListStages(context.Background(), lease.TaskID)
	if err != nil || stages[0].Status != moduleapi.StageStatusUnknown || stages[0].FailureCode == nil || *stages[0].FailureCode != "external_execution_lease_expired" {
		t.Fatalf("expired leased stage = %#v err=%v", stages, err)
	}
	late, err := runtime.SettleExternalExecution(context.Background(), testExternalExecutionReceipt(lease, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil || late.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("late fenced receipt = %#v err=%v", late, err)
	}
}

func TestExternalExecutionRetryCreatesNewFencedAttempt(t *testing.T) {
	runtime, repository, first := claimedExternalExecutionTaskWithAttempts(t, 1, 2)
	if count, err := repository.ExpireExternalExecutionLeases(context.Background(), first.AbsoluteDeadlineAt.Add(time.Second), 10); err != nil || count != 1 {
		t.Fatalf("expire first external lease = count:%d err:%v", count, err)
	}
	if err := runtime.RetryStage(context.Background(), first.TaskID, first.StageID); err != nil {
		t.Fatalf("retry expired external stage: %v", err)
	}
	second := claimTestExternalExecution(t, runtime)
	if second.ID == first.ID || second.Attempt != first.Attempt+1 || second.OperationID != first.OperationID {
		t.Fatalf("retried external lease = first:%#v second:%#v", first, second)
	}
	settlement, err := runtime.SettleExternalExecution(context.Background(), testExternalExecutionReceipt(second, moduleapi.ExternalReceiptOutcomeSuccess, ""))
	if err != nil || settlement.Status != moduleapi.TaskStatusSuccess {
		t.Fatalf("settle retried external stage = %#v err=%v", settlement, err)
	}
}

func TestExternalExecutionClaimScansPastUnmatchedPage(t *testing.T) {
	runtime, _ := newRuntimeForTest(t)
	for index := 0; index < externalExecutionCandidateLimit; index++ {
		submitExternalExecutionForTarget(t, runtime, int64(1000+index), "unmatched-operation")
	}
	matchedTaskID := submitExternalExecutionForTarget(t, runtime, 42, "matched-operation")
	lease := claimTestExternalExecution(t, runtime)
	if lease.TaskID != matchedTaskID {
		t.Fatalf("claimed paged external execution task = %d, want %d", lease.TaskID, matchedTaskID)
	}
}

func TestExternalExecutionRejectsIncompleteExpectationAndOversizedLogs(t *testing.T) {
	runtime, _, lease := claimedExternalExecutionTask(t, 1)
	if err := runtime.AppendExternalExecutionLogs(context.Background(), moduleapi.ExternalExecutionLogBatch{
		Handle: externalExecutionHandle(lease), Entries: []moduleapi.TaskLogEntry{{Stream: "stdout", Level: "info", Line: strings.Repeat("x", externalExecutionLogLineMaxRunes+1)}},
	}); !errors.Is(err, taskstore.ErrInvalidInput) {
		t.Fatalf("oversized log error = %v", err)
	}
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].ExternalExecution = &moduleapi.ExternalExecutionExpectation{RuntimeTargetID: 1}
	if _, err := runtime.Submit(context.Background(), input); err == nil {
		t.Fatal("incomplete external execution expectation was accepted")
	}
	duplicate := testSubmitInput(2, 1)
	duplicate.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("duplicate-operation")
	duplicate.Plan.Stages[1].ExternalExecution = testExternalExecutionExpectation("duplicate-operation")
	if _, err := runtime.Submit(context.Background(), duplicate); err == nil {
		t.Fatal("duplicate external operation identity was accepted")
	}
}

func claimedExternalExecutionTask(t *testing.T, stageCount int) (*Runtime, *taskstore.SQLRepository, moduleapi.ExternalExecutionLease) {
	return claimedExternalExecutionTaskWithAttempts(t, stageCount, 1)
}

func claimedExternalExecutionTaskWithAttempts(t *testing.T, stageCount int, attempts int) (*Runtime, *taskstore.SQLRepository, moduleapi.ExternalExecutionLease) {
	t.Helper()
	runtime, repository := newRuntimeForTest(t)
	if stageCount > 1 {
		if err := runtime.RegisterStageExecutor(&recordingExecutor{}); err != nil {
			t.Fatalf("register remaining executor: %v", err)
		}
	}
	input := testSubmitInput(stageCount, attempts)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation("operation-stage-1")
	if _, err := runtime.Submit(context.Background(), input); err != nil {
		t.Fatalf("submit external execution task: %v", err)
	}
	return runtime, repository, claimTestExternalExecution(t, runtime)
}

func submitExternalExecutionForTarget(t *testing.T, runtime *Runtime, runtimeTargetID int64, operationID string) uint64 {
	t.Helper()
	input := testSubmitInput(1, 1)
	input.Plan.Stages[0].RecoveryPolicy = moduleapi.StageRecoveryManualReconcile
	input.Plan.Stages[0].ExternalExecution = testExternalExecutionExpectation(operationID)
	input.Plan.Stages[0].ExternalExecution.RuntimeTargetID = runtimeTargetID
	receipt, err := runtime.Submit(context.Background(), input)
	if err != nil {
		t.Fatalf("submit external execution for target %d: %v", runtimeTargetID, err)
	}
	return receipt.TaskID
}

func claimTestExternalExecution(t *testing.T, runtime *Runtime) moduleapi.ExternalExecutionLease {
	t.Helper()
	lease, err := runtime.ClaimExternalExecution(context.Background(), moduleapi.ExternalExecutionClaimRequest{
		RuntimeTargetID: 42, ProviderID: "docker", Capability: "compose_execution",
	})
	if err != nil {
		t.Fatalf("claim external execution: %v", err)
	}
	return lease
}

func testExternalExecutionExpectation(operationID string) *moduleapi.ExternalExecutionExpectation {
	return &moduleapi.ExternalExecutionExpectation{
		RuntimeTargetID: 42, ProviderID: "docker", Capability: "compose_execution", Protocol: "runtime-agent/v1",
		OperationID: operationID, LeaseTTL: time.Minute, AbsoluteDeadline: 10 * time.Minute,
	}
}

func externalExecutionHandle(lease moduleapi.ExternalExecutionLease) moduleapi.ExternalExecutionLeaseHandle {
	return moduleapi.ExternalExecutionLeaseHandle{LeaseID: lease.ID, FenceToken: lease.FenceToken}
}

func testExternalExecutionReceipt(lease moduleapi.ExternalExecutionLease, outcome moduleapi.ExternalReceiptOutcome, failureCode string) moduleapi.ExternalExecutionReceipt {
	return moduleapi.ExternalExecutionReceipt{Handle: externalExecutionHandle(lease), Outcome: outcome, FailureCode: failureCode, IntegritySHA256: strings.Repeat("c", 64)}
}
