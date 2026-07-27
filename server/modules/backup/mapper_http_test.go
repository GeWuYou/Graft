package backup

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

func TestBackupDetailResponseExposesSafeArtifactMetadataAndRestoreEvidence(t *testing.T) {
	t.Parallel()
	taskID := uint64(75)
	creatorID := uint64(42)
	recordedAt := time.Date(2026, time.July, 27, 7, 17, 0, 0, time.UTC)
	item := moduleapi.Backup{
		ID: 2, TaskID: &taskID, Purpose: "manual_backup", Status: moduleapi.BackupStatusRestored,
		ConfigSnapshot: moduleapi.BackupArtifact{StorageRef: "/private/backups/2/config.snapshot", SHA256: strings.Repeat("a", 64), SizeBytes: 128},
		DatabaseDump:   moduleapi.BackupArtifact{StorageRef: "/private/backups/2/database.dump", SHA256: strings.Repeat("b", 64), SizeBytes: 256},
		RetainUntil:    recordedAt.Add(24 * time.Hour), CreatedBy: &creatorID, CreatedAt: recordedAt,
		RestoreCode: "manual_restore_verified", RestoreAt: &recordedAt,
	}

	response := toBackupDetailResponse(item)
	if response.ConfigSnapshot.SizeBytes != 128 || response.ConfigSnapshot.SHA256 != strings.Repeat("a", 64) || response.DatabaseDump.SizeBytes != 256 || response.RestoreEvidence.Status != "RECORDED" || response.RestoreEvidence.ResultCode == nil || *response.RestoreEvidence.ResultCode != "manual_restore_verified" || response.RestoreEvidence.RecordedAt == nil {
		t.Fatalf("detail response mismatch: %#v", response)
	}
	encoded, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("marshal detail response: %v", err)
	}
	for _, prohibited := range []string{"storage_ref", "/private/backups", "created_by", "42"} {
		if strings.Contains(string(encoded), prohibited) {
			t.Fatalf("detail response exposed prohibited value %q: %s", prohibited, encoded)
		}
	}
}

func TestBackupDetailResponseMarksMissingRestoreEvidence(t *testing.T) {
	t.Parallel()
	response := toBackupDetailResponse(moduleapi.Backup{RestoreCode: "not_attempted"})
	if response.RestoreEvidence.Status != "NOT_VERIFIED" || response.RestoreEvidence.ResultCode != nil || response.RestoreEvidence.RecordedAt != nil {
		t.Fatalf("expected missing restore evidence to remain unverified: %#v", response.RestoreEvidence)
	}
}

func TestBackupSummaryResponseDoesNotExposeAssociatedTask(t *testing.T) {
	t.Parallel()
	encoded, err := json.Marshal(toBackupSummaryResponse(moduleapi.BackupSummary{ID: 2, Purpose: "manual_backup"}))
	if err != nil {
		t.Fatalf("marshal summary response: %v", err)
	}
	if strings.Contains(string(encoded), "task_id") {
		t.Fatalf("summary response exposed an associated task: %s", encoded)
	}
}
