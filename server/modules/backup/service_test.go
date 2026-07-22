package backup

import (
	"context"
	"errors"
	"testing"
	"time"

	"graft/server/internal/moduleapi"
)

type serviceTestRepository struct {
	created  moduleapi.CreateBackupInput
	restored moduleapi.RecordBackupRestoreInput
	item     moduleapi.Backup
}

func (r *serviceTestRepository) Create(_ context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	r.created = input
	return r.item, nil
}
func (r *serviceTestRepository) Get(_ context.Context, _ uint64) (moduleapi.Backup, error) {
	return r.item, nil
}
func (r *serviceTestRepository) RecordRestoreEvidence(_ context.Context, input moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	r.restored = input
	return r.item, nil
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
	if _, err := service.RecordRestoreEvidence(context.Background(), moduleapi.RecordBackupRestoreInput{ID: 7, Status: moduleapi.BackupStatusRestored, RestoreCode: "manual_restore_verified", RecordedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("record restore evidence: %v", err)
	}
	summary := ToSummary(created)
	if summary.ID != created.ID || summary.Purpose != created.Purpose || summary.Status != created.Status {
		t.Fatalf("summary mismatch: %#v", summary)
	}
}

func TestServiceRejectsUnavailableRepositoryAndInvalidID(t *testing.T) {
	t.Parallel()
	if _, err := (*Service)(nil).Create(context.Background(), moduleapi.CreateBackupInput{}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected unavailable service error, got %v", err)
	}
	if _, err := NewService(&serviceTestRepository{}).Get(context.Background(), 0); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected invalid id error, got %v", err)
	}
}
