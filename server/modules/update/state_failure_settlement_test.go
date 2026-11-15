package update

import (
	"testing"

	"graft/server/internal/moduleapi"
)

func TestBoundStateWriteFailureSettlesTaskAsAttentionWithoutClaimingMigration(t *testing.T) {
	for _, migrated := range []bool{false, true} {
		tasks := &stubTaskService{}
		backups := &plannedBackupHandoff{planned: true}
		operation := ComposeUpdateOperation{OperationID: "update-state-write", RunnerID: "runner-state-write", TaskID: 71}
		receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, RunnerID: operation.RunnerID,
			ExecutionMode: "local", LaunchStageID: 101, LaunchAttempt: 2, MigrationStarted: migrated,
			FailureCode: RunnerFailureCodeStateWriteFailed, FailureStage: RunnerFailureStagePermissionDenied}
		settled, err := NewComposeExecutionCoordinator(tasks, backups).SettleReceipt(t.Context(), operation, receipt)
		if err != nil || settled.Outcome != ExecutionOutcomeNeedsAttention || tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeNeedsAttention {
			t.Fatalf("state failure settlement=%#v task=%#v err=%v", settled, tasks.external, err)
		}
		if tasks.external.LaunchStageID != 101 || tasks.external.LaunchAttempt != 2 || !backups.planned || backups.cancelCalls != 0 {
			t.Fatalf("state failure lost reconciliation authority: task=%#v backup=%#v", tasks.external, backups)
		}
	}
}

func TestStateFailureVocabularyKeepsTheNewCodeBounded(t *testing.T) {
	if !validRunnerStateFailure(RunnerFailureCodeStateWriteFailed) {
		t.Fatal("the canonical state write failure cannot be persisted")
	}
	if validRunnerStateFailure("arbitrary_unapproved_failure") {
		t.Fatal("unrecognized failure text was accepted into the state journal")
	}
}
