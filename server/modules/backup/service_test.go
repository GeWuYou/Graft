package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type serviceTestRepository struct {
	created   moduleapi.CreateBackupInput
	restored  moduleapi.RecordBackupRestoreInput
	item      moduleapi.Backup
	plan      moduleapi.BackupRunnerHandoffPlan
	completed moduleapi.CompleteBackupRunnerHandoffInput
	settledID uint64
}

func (r *serviceTestRepository) Create(_ context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	r.created = input
	return r.item, nil
}
func (r *serviceTestRepository) Get(_ context.Context, _ uint64) (moduleapi.Backup, error) {
	return r.item, nil
}
func (r *serviceTestRepository) GetSummary(_ context.Context, _ uint64) (moduleapi.BackupSummary, error) {
	return ToSummary(r.item), nil
}
func (r *serviceTestRepository) ListSummaries(context.Context, int, int) ([]moduleapi.BackupSummary, int64, error) {
	return []moduleapi.BackupSummary{ToSummary(r.item)}, 1, nil
}
func (r *serviceTestRepository) RecordRestoreEvidence(_ context.Context, input moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	r.restored = input
	return r.item, nil
}
func (r *serviceTestRepository) PrepareRunnerHandoff(_ context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	r.plan = plan
	return plan, nil
}
func (*serviceTestRepository) CancelRunnerHandoff(context.Context, string, uint64) error { return nil }
func (r *serviceTestRepository) GetRunnerHandoff(_ context.Context, _ string, _ uint64) (moduleapi.BackupRunnerHandoffPlan, uint64, error) {
	return r.plan, r.settledID, nil
}
func (r *serviceTestRepository) CompleteRunnerHandoff(_ context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error) {
	r.completed = input
	return moduleapi.BackupRunnerHandoffCompletion{BackupID: 7, OperationID: input.OperationID, TaskID: input.TaskID}, nil
}

func TestServiceForwardsBackupCapabilityAndProjectsSafeSummary(t *testing.T) {
	t.Parallel()
	repository := &serviceTestRepository{item: moduleapi.Backup{
		ID:             7,
		Purpose:        "platform_update",
		Status:         moduleapi.BackupStatusAvailable,
		ConfigSnapshot: moduleapi.BackupArtifact{StorageRef: "/state/backup/config.env", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SizeBytes: 10},
		DatabaseDump:   moduleapi.BackupArtifact{StorageRef: "/state/backup/database.dump", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SizeBytes: 20},
		RetainUntil:    time.Now().UTC().Add(time.Hour),
		RestoreCode:    "not_attempted",
	}}
	service := NewService(repository)
	input := moduleapi.CreateBackupInput{Purpose: "platform_update", ConfigSnapshot: repository.item.ConfigSnapshot, DatabaseDump: repository.item.DatabaseDump, RetainUntil: repository.item.RetainUntil}
	created, err := service.Create(context.Background(), input)
	if err != nil || created.ID != repository.item.ID || repository.created.Purpose != input.Purpose {
		t.Fatalf("create capability mismatch: item=%#v input=%#v err=%v", created, repository.created, err)
	}
	restoreEvidence := moduleapi.RecordBackupRestoreInput{ID: 7, Status: moduleapi.BackupStatusRestored, RestoreCode: "manual_restore_verified", RecordedAt: time.Now().UTC()}
	if _, err := service.RecordRestoreEvidence(context.Background(), restoreEvidence); err != nil {
		t.Fatalf("record restore evidence: %v", err)
	}
	if repository.restored != restoreEvidence {
		t.Fatalf("restore evidence was not forwarded: got=%#v want=%#v", repository.restored, restoreEvidence)
	}
	summary := ToSummary(created)
	if summary.ID != created.ID || summary.Purpose != created.Purpose || summary.Status != created.Status {
		t.Fatalf("summary mismatch: %#v", summary)
	}
	encodedSummary, err := json.Marshal(summary)
	if err != nil || strings.Contains(string(encodedSummary), "manual_restore_verified") || strings.Contains(string(encodedSummary), "restore_code") {
		t.Fatalf("summary exposed restore evidence: json=%s err=%v", encodedSummary, err)
	}
}

func TestServiceRejectsUnavailableRepositoryAndInvalidID(t *testing.T) {
	t.Parallel()
	service := (*Service)(nil)
	if _, err := service.Create(context.Background(), moduleapi.CreateBackupInput{}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable create service error, got %v", err)
	}
	if _, err := service.Get(context.Background(), 1); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable get service error, got %v", err)
	}
	if _, err := service.GetSummary(context.Background(), 1); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable summary service error, got %v", err)
	}
	if _, _, err := service.ListSummaries(context.Background(), 1, 0); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable list service error, got %v", err)
	}
	if _, err := service.RecordRestoreEvidence(context.Background(), moduleapi.RecordBackupRestoreInput{}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable restore service error, got %v", err)
	}
	if _, err := NewService(&serviceTestRepository{}).Get(context.Background(), 0); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected invalid id error, got %v", err)
	}
}

func TestManualRetentionDeadlineUsesOnlyManualBackupPolicy(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
	cases := map[ManualRetention]time.Duration{
		ManualRetentionOneDay:     24 * time.Hour,
		ManualRetentionSevenDays:  7 * 24 * time.Hour,
		ManualRetentionThirtyDays: 30 * 24 * time.Hour,
	}
	for retention, duration := range cases {
		deadline, err := ManualRetentionDeadline(retention, now)
		if err != nil || !deadline.Equal(now.Add(duration)) {
			t.Fatalf("retention %q deadline=%s err=%v", retention, deadline, err)
		}
	}
	if _, err := ManualRetentionDeadline("14d", now); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected invalid retention error, got %v", err)
	}
}

func TestServiceCompletesRunnerHandoffAfterVerifyingFrozenArtifacts(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	configRef := filepath.Join(root, "config.snapshot")
	dumpRef := filepath.Join(root, "database.dump")
	configContent := []byte("CONFIG=redacted\n")
	dumpContent := []byte("postgres dump")
	writeRunnerArtifact(t, configRef, configContent)
	writeRunnerArtifact(t, dumpRef, dumpContent)
	repository := &serviceTestRepository{plan: moduleapi.BackupRunnerHandoffPlan{
		OperationID: "update-42", TaskID: 42, Purpose: "platform_update", RetainUntil: time.Now().UTC().Add(time.Hour),
		ArtifactRoot: root, ConfigSnapshotRef: configRef, DatabaseDumpRef: dumpRef,
	}}
	service := NewService(repository)
	configSHA, configSize := runnerArtifactDigest(configContent)
	dumpSHA, dumpSize := runnerArtifactDigest(dumpContent)
	completion, err := service.CompleteRunnerHandoff(context.Background(), moduleapi.CompleteBackupRunnerHandoffInput{
		OperationID: "update-42", TaskID: 42, ConfigSnapshotSHA256: "  " + strings.ToUpper(configSHA) + "  ", ConfigSnapshotBytes: configSize,
		DatabaseDumpSHA256: "\t" + strings.ToUpper(dumpSHA), DatabaseDumpBytes: dumpSize,
	})
	if err != nil || completion.BackupID != 7 {
		t.Fatalf("complete verified handoff: completion=%#v err=%v", completion, err)
	}
	if repository.completed.ConfigSnapshotSHA256 != configSHA || repository.completed.ConfigSnapshotBytes != configSize || repository.completed.DatabaseDumpSHA256 != dumpSHA || repository.completed.DatabaseDumpBytes != dumpSize {
		t.Fatalf("completion did not use server-verified metadata: %#v", repository.completed)
	}
	if _, err := service.CompleteRunnerHandoff(context.Background(), moduleapi.CompleteBackupRunnerHandoffInput{
		OperationID: "update-42", TaskID: 42, ConfigSnapshotSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ConfigSnapshotBytes: configSize,
		DatabaseDumpSHA256: dumpSHA, DatabaseDumpBytes: dumpSize,
	}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected forged checksum rejection, got %v", err)
	}
}

func TestServiceRejectsRunnerArtifactOutsideFrozenRoot(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.dump")
	writeRunnerArtifact(t, outside, []byte("dump"))
	service := NewService(&serviceTestRepository{})
	if _, err := service.PrepareRunnerHandoff(context.Background(), moduleapi.BackupRunnerHandoffPlan{
		OperationID: "update-43", TaskID: 43, Purpose: "platform_update", RetainUntil: time.Now().UTC().Add(time.Hour),
		ArtifactRoot: root, ConfigSnapshotRef: filepath.Join(root, "config"), DatabaseDumpRef: outside,
	}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected outside artifact rejection, got %v", err)
	}
}

func TestServiceReplaysSettledRunnerHandoffWithoutReadingExpiredArtifacts(t *testing.T) {
	t.Parallel()
	repository := &serviceTestRepository{settledID: 7}
	service := NewService(repository)
	input := moduleapi.CompleteBackupRunnerHandoffInput{
		OperationID: "update-45", TaskID: 45,
		ConfigSnapshotSHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", ConfigSnapshotBytes: 10,
		DatabaseDumpSHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", DatabaseDumpBytes: 20,
	}
	completion, err := service.CompleteRunnerHandoff(context.Background(), input)
	if err != nil || completion.BackupID != 7 || repository.completed.OperationID != input.OperationID {
		t.Fatalf("replay settled handoff: completion=%#v completed=%#v err=%v", completion, repository.completed, err)
	}
}

func writeRunnerArtifact(t *testing.T, path string, content []byte) {
	t.Helper()
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write runner artifact: %v", err)
	}
}

func runnerArtifactDigest(content []byte) (string, int64) {
	digest := sha256.Sum256(content)
	return hex.EncodeToString(digest[:]), int64(len(content))
}
