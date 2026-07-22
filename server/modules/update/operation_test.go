package update

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestComposeExecutionCoordinatorSettlesReceiptAndDoesNotExposeBackupRefs(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 41, Status: moduleapi.TaskStatusPending}}
	backups := &stubBackupService{}
	coordinator := NewComposeExecutionCoordinator(tasks, backups)
	operation, input, err := coordinator.Start(context.Background(), ComposeUpdateOperation{OperationID: "update-41", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0"}, 9, testBackupPlan("update-41"))
	if err != nil {
		t.Fatalf("start compose update: %v", err)
	}
	if input.TaskID != 41 || backups.plan.TaskID != 41 || tasks.plan.Plan.Stages[0].ExternalReceipt.OperationID != operation.OperationID {
		t.Fatalf("operation linkage was not frozen: %#v %#v", operation, backups.plan)
	}
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, Succeeded: true, BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: operation.OperationID, TaskID: operation.TaskID, ConfigSnapshotSHA256: testDigest('a'), ConfigSnapshotBytes: 3, DatabaseDumpSHA256: testDigest('b'), DatabaseDumpBytes: 5}}
	settled, err := coordinator.SettleReceipt(context.Background(), operation, receipt)
	if err != nil {
		t.Fatalf("settle compose receipt: %v", err)
	}
	if settled.Outcome != ExecutionOutcomeSuccess || settled.BackupID != 8 || tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeSuccess || tasks.external.IntegritySHA256 == "" {
		t.Fatalf("unexpected settled operation: %#v / %#v", settled, tasks.external)
	}
	if backups.completion.OperationID != operation.OperationID || backups.completion.TaskID != operation.TaskID {
		t.Fatalf("backup completion binding lost: %#v", backups.completion)
	}
}

func TestComposeExecutionCoordinatorMarksPostMigrationFailureNeedsAttention(t *testing.T) {
	tasks := &stubTaskService{receipt: moduleapi.TaskReceipt{TaskID: 52, Status: moduleapi.TaskStatusPending}}
	coordinator := NewComposeExecutionCoordinator(tasks, &stubBackupService{})
	operation := ComposeUpdateOperation{OperationID: "update-52", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 52}
	settled, err := coordinator.SettleReceipt(context.Background(), operation, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: operation.OperationID, MigrationStarted: true, FailureCode: "healthz_failed"})
	if err != nil {
		t.Fatalf("settle post-migration failure: %v", err)
	}
	if settled.Outcome != ExecutionOutcomeNeedsAttention || tasks.external.Outcome != moduleapi.ExternalReceiptOutcomeNeedsAttention {
		t.Fatalf("post-migration receipt was not retained for attention: %#v %#v", settled, tasks.external)
	}
}

func TestComposeExecutionCoordinatorRejectsForgedBackupReceiptBinding(t *testing.T) {
	coordinator := NewComposeExecutionCoordinator(&stubTaskService{}, &stubBackupService{})
	_, err := coordinator.SettleReceipt(context.Background(), ComposeUpdateOperation{OperationID: "update-53", SourceVersion: "v1.0.0", TargetVersion: "v1.1.0", TaskID: 53}, RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-53", FailureCode: "pull_failed", BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "other", TaskID: 53}})
	if err == nil {
		t.Fatal("expected forged backup completion rejection")
	}
}

func TestRunnerReceiptDoesNotSerializeBackupStorageReferences(t *testing.T) {
	receipt := RunnerReceipt{ProtocolVersion: runnerProtocolVersion, OperationID: "update-54", BackupCompletion: &moduleapi.CompleteBackupRunnerHandoffInput{OperationID: "update-54", TaskID: 54, ConfigSnapshotSHA256: testDigest('a'), DatabaseDumpSHA256: testDigest('b')}}
	encoded, err := json.Marshal(receipt)
	if err != nil {
		t.Fatalf("marshal runner receipt: %v", err)
	}
	if string(encoded) == "" || containsAny(string(encoded), "storage_ref", "database_dump_ref", "config_snapshot_ref", "/var/") {
		t.Fatalf("runner receipt exposes backup storage data: %s", encoded)
	}
}

func containsAny(value string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(value, needle) {
			return true
		}
	}
	return false
}

type stubTaskService struct {
	receipt  moduleapi.TaskReceipt
	plan     moduleapi.SubmitTaskInput
	external moduleapi.ExternalTaskReceipt
}

func (s *stubTaskService) Submit(_ context.Context, input moduleapi.SubmitTaskInput) (moduleapi.TaskReceipt, error) {
	s.plan = input
	return s.receipt, nil
}
func (s *stubTaskService) SettleExternalReceipt(_ context.Context, receipt moduleapi.ExternalTaskReceipt) (moduleapi.ExternalReceiptSettlement, error) {
	s.external = receipt
	return moduleapi.ExternalReceiptSettlement{TaskID: receipt.TaskID}, nil
}
func (*stubTaskService) Cancel(context.Context, uint64) error             { return nil }
func (*stubTaskService) RetryStage(context.Context, uint64, uint64) error { return nil }

type stubBackupService struct {
	plan       moduleapi.BackupRunnerHandoffPlan
	completion moduleapi.CompleteBackupRunnerHandoffInput
}

func (s *stubBackupService) Create(context.Context, moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}
func (s *stubBackupService) PrepareRunnerHandoff(_ context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	s.plan = plan
	return plan, nil
}
func (s *stubBackupService) CompleteRunnerHandoff(_ context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error) {
	s.completion = input
	return moduleapi.BackupRunnerHandoffCompletion{BackupID: 8, OperationID: input.OperationID, TaskID: input.TaskID}, nil
}
func (*stubBackupService) Get(context.Context, uint64) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}
func (*stubBackupService) RecordRestoreEvidence(context.Context, moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	return moduleapi.Backup{}, errors.New("unused")
}

func testBackupPlan(operationID string) moduleapi.BackupRunnerHandoffPlan {
	return moduleapi.BackupRunnerHandoffPlan{OperationID: operationID, Purpose: "before-update", RetainUntil: time.Now().Add(time.Hour), ArtifactRoot: "/var/lib/graft/backups", ConfigSnapshotRef: "/var/lib/graft/backups/config", DatabaseDumpRef: "/var/lib/graft/backups/dump"}
}
func testDigest(character rune) string { return string(makeDigest(character)) }
func makeDigest(character rune) []rune {
	value := make([]rune, 64)
	for index := range value {
		value[index] = character
	}
	return value
}
