package store

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"graft/server/internal/moduleapi"
)

func TestSQLRepositoryStoresMetadataWithoutArtifactContent(t *testing.T) {
	t.Parallel()
	repository, db := newTestRepository(t)
	input := validCreateInput()
	created, err := repository.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if created.ID == 0 || created.Status != moduleapi.BackupStatusAvailable || created.ConfigSnapshot.SHA256 != input.ConfigSnapshot.SHA256 || created.DatabaseDump.SizeBytes != input.DatabaseDump.SizeBytes {
		t.Fatalf("created backup mismatch: %#v", created)
	}

	var configContent, dumpContent int
	if err := db.QueryRow(`SELECT COUNT(*), COUNT(*) FROM backups`).Scan(&configContent, &dumpContent); err != nil || configContent != 1 || dumpContent != 1 {
		t.Fatalf("expected one metadata row, counts=%d/%d err=%v", configContent, dumpContent, err)
	}
	found, err := repository.Get(context.Background(), created.ID)
	if err != nil || found.ConfigSnapshot.StorageRef != input.ConfigSnapshot.StorageRef || found.DatabaseDump.StorageRef != input.DatabaseDump.StorageRef {
		t.Fatalf("get backup mismatch: %#v err=%v", found, err)
	}
}

func TestSQLRepositoryRecordsOnlyStructuredRestoreEvidence(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	created, err := repository.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	recordedAt := time.Now().UTC().Add(time.Minute)
	updated, err := repository.RecordRestoreEvidence(context.Background(), moduleapi.RecordBackupRestoreInput{ID: created.ID, Status: moduleapi.BackupStatusRestored, RestoreCode: "manual_restore_verified", RecordedAt: recordedAt})
	if err != nil {
		t.Fatalf("record restore evidence: %v", err)
	}
	if updated.Status != moduleapi.BackupStatusRestored || updated.RestoreCode != "manual_restore_verified" || updated.RestoreAt == nil {
		t.Fatalf("restore evidence mismatch: %#v", updated)
	}
	if _, err := repository.RecordRestoreEvidence(context.Background(), moduleapi.RecordBackupRestoreInput{ID: created.ID, Status: moduleapi.BackupStatusAvailable, RestoreCode: "invalid", RecordedAt: recordedAt}); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected invalid restore status, got %v", err)
	}
}

func TestSQLRepositoryRejectsInvalidArtifactMetadata(t *testing.T) {
	t.Parallel()
	repository, _ := newTestRepository(t)
	input := validCreateInput()
	input.ConfigSnapshot.SHA256 = "not-a-checksum"
	if _, err := repository.Create(context.Background(), input); !errors.Is(err, moduleapi.ErrBackupInvalidInput) {
		t.Fatalf("expected invalid checksum, got %v", err)
	}
	if _, err := repository.Get(context.Background(), 99); !errors.Is(err, moduleapi.ErrBackupNotFound) {
		t.Fatalf("expected not found, got %v", err)
	}
}

func validCreateInput() moduleapi.CreateBackupInput {
	creator := uint64(42)
	return moduleapi.CreateBackupInput{
		Purpose:        "platform_update",
		ConfigSnapshot: moduleapi.BackupArtifact{StorageRef: "artifact://backup/config/1", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", SizeBytes: 100},
		DatabaseDump:   moduleapi.BackupArtifact{StorageRef: "artifact://backup/dump/1", SHA256: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", SizeBytes: 200},
		RetainUntil:    time.Now().UTC().Add(time.Hour),
		CreatedBy:      &creator,
	}
}

func newTestRepository(t *testing.T) (*SQLRepository, *sql.DB) {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`CREATE TABLE backups (
		id INTEGER PRIMARY KEY AUTOINCREMENT, purpose TEXT NOT NULL, status TEXT NOT NULL,
		config_snapshot_ref TEXT NOT NULL, config_snapshot_sha256 TEXT NOT NULL, config_snapshot_bytes INTEGER NOT NULL,
		database_dump_ref TEXT NOT NULL, database_dump_sha256 TEXT NOT NULL, database_dump_bytes INTEGER NOT NULL,
		retain_until DATETIME NOT NULL, created_by INTEGER NULL, created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP, restore_code TEXT NOT NULL DEFAULT '', restore_at DATETIME NULL,
		deleted_at INTEGER NOT NULL DEFAULT 0, deleted_by INTEGER NULL
	)`); err != nil {
		t.Fatalf("create backups table: %v", err)
	}
	repository, err := NewSQLRepository(db)
	if err != nil {
		t.Fatalf("new repository: %v", err)
	}
	return repository, db
}
