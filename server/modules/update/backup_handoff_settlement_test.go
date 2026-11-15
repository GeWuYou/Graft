package update

import (
	"context"
	"errors"
	"testing"

	"graft/server/internal/moduleapi"
)

type plannedBackupHandoff struct {
	stubBackupService
	planned     bool
	cancelError error
	cancelCalls int
}

func (s *plannedBackupHandoff) CancelRunnerHandoff(ctx context.Context, operation string, task uint64) error {
	s.cancelCalls++
	if s.cancelError != nil {
		return s.cancelError
	}
	s.planned = false
	return s.stubBackupService.CancelRunnerHandoff(ctx, operation, task)
}

func failedBackupSettlement() (ComposeUpdateOperation, RunnerReceipt) {
	operation := ComposeUpdateOperation{OperationID: "update-backup-cleanup", TaskID: 71}
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID,
		FailureCode: runnerFailureBackup, FailureStage: string(RunnerBackupFailureStageConfigSnapshot),
		FailureDetail: string(RunnerBackupFailureDetailPermissionDenied)}
	return operation, receipt
}

func TestKnownBackupFailureReleasesPreparedHandoffAfterTrustedTaskSettlement(t *testing.T) {
	tasks := &stubTaskService{}
	backups := &plannedBackupHandoff{planned: true}
	operation, receipt := failedBackupSettlement()
	settled, err := NewComposeExecutionCoordinator(tasks, backups).SettleReceipt(t.Context(), operation, receipt)
	if err != nil {
		t.Fatal(err)
	}
	if backups.planned || backups.cancelCalls != 1 || settled.Outcome != ExecutionOutcomeFailed {
		t.Fatalf("settled=%#v planned=%t cancelCalls=%d", settled, backups.planned, backups.cancelCalls)
	}
	if tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeFailed {
		t.Fatalf("task receipt=%#v", tasks.external)
	}
}

func TestBackupCleanupFailureRetainsOperationForIdempotentReceiptRetry(t *testing.T) {
	cleanupError := errors.New("backup handoff cancellation temporarily unavailable")
	tasks := &stubTaskService{}
	backups := &plannedBackupHandoff{planned: true, cancelError: cleanupError}
	coordinator := NewComposeExecutionCoordinator(tasks, backups)
	operation, receipt := failedBackupSettlement()
	settled, err := coordinator.SettleReceipt(t.Context(), operation, receipt)
	if !errors.Is(err, cleanupError) || settled.OperationID != "" || !backups.planned {
		t.Fatalf("settled=%#v planned=%t err=%v", settled, backups.planned, err)
	}
	backups.cancelError = nil
	for range 2 {
		settled, err = coordinator.SettleReceipt(t.Context(), operation, receipt)
		if err != nil || settled.Outcome != ExecutionOutcomeFailed || backups.planned {
			t.Fatalf("settled=%#v planned=%t err=%v", settled, backups.planned, err)
		}
	}
}

func TestBackupCleanupPreservesUncertainAndCompletedHandoffs(t *testing.T) {
	operation, receipt := failedBackupSettlement()
	cases := map[string]RunnerReceipt{
		"migration started": receipt,
		"unknown failure":   receipt,
		"unknown stage":     receipt,
		"unknown detail":    receipt,
		"successful":        receipt,
		"completed backup":  receipt,
	}
	value := cases["migration started"]
	value.MigrationStarted = true
	cases["migration started"] = value
	value = cases["unknown failure"]
	value.FailureCode = "runner_interrupted"
	cases["unknown failure"] = value
	value = cases["unknown stage"]
	value.FailureStage = "unknown"
	cases["unknown stage"] = value
	value = cases["unknown detail"]
	value.FailureDetail = "unknown"
	cases["unknown detail"] = value
	value = cases["successful"]
	value.Succeeded = true
	cases["successful"] = value
	value = cases["completed backup"]
	value.BackupCompletion = &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: operation.OperationID, TaskID: operation.TaskID}
	cases["completed backup"] = value
	for name, evidence := range cases {
		t.Run(name, func(t *testing.T) {
			backups := &plannedBackupHandoff{planned: true}
			_, err := NewComposeExecutionCoordinator(&stubTaskService{}, backups).SettleReceipt(t.Context(), operation, evidence)
			if err != nil || backups.cancelCalls != 0 {
				t.Fatalf("cancelCalls=%d err=%v", backups.cancelCalls, err)
			}
		})
	}
}

func TestBackupCleanupPreservesAnExistingBackupFact(t *testing.T) {
	operation, receipt := failedBackupSettlement()
	operation.BackupID = 42
	backups := &plannedBackupHandoff{}
	settled, err := NewComposeExecutionCoordinator(&stubTaskService{}, backups).SettleReceipt(t.Context(), operation, receipt)
	if err != nil || settled.BackupID != operation.BackupID || backups.cancelCalls != 0 {
		t.Fatalf("settled=%#v cancelCalls=%d err=%v", settled, backups.cancelCalls, err)
	}
}

type rejectedReceiptTaskService struct{ stubTaskService }

func (*rejectedReceiptTaskService) SettleExternalReceipt(context.Context, moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	return moduleapi.ExternalReceiptSettlement{}, errors.New("stale launch attempt")
}

func TestStaleTaskReceiptCannotCancelPreparedBackupHandoff(t *testing.T) {
	operation, receipt := failedBackupSettlement()
	backups := &plannedBackupHandoff{planned: true}
	_, err := NewComposeExecutionCoordinator(&rejectedReceiptTaskService{}, backups).SettleReceipt(t.Context(), operation, receipt)
	if err == nil || backups.cancelCalls != 0 || !backups.planned {
		t.Fatalf("cancelCalls=%d planned=%t err=%v", backups.cancelCalls, backups.planned, err)
	}
}
