package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
)

const (
	maxPurposeLength     = 64
	maxStorageRefLength  = 2048
	maxRestoreCodeLength = 128
	sha256Length         = 64
)

// SQLRepository 将 backup 模块拥有的元数据持久化到其 owner-aligned 表。
type SQLRepository struct{ db *sql.DB }

// NewSQLRepository 创建由平台 SQL 连接支持的备份仓储。
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("backup repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db}, nil
}

// Create 写入完整性已验证的配置快照和数据库 dump 元数据。
func (r *SQLRepository) Create(ctx context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	input, err := normalizeCreate(input)
	if err != nil {
		return moduleapi.Backup{}, err
	}
	var item moduleapi.Backup
	if input.TaskID != 0 {
		return r.createForTask(ctx, input)
	}
	err = r.db.QueryRowContext(ctx, `INSERT INTO backups (
		purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
	RETURNING id, task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at, restore_code, restore_at`,
		input.Purpose, moduleapi.BackupStatusAvailable, input.ConfigSnapshot.StorageRef, input.ConfigSnapshot.SHA256, input.ConfigSnapshot.SizeBytes,
		input.DatabaseDump.StorageRef, input.DatabaseDump.SHA256, input.DatabaseDump.SizeBytes, input.RetainUntil.UTC(), input.CreatedBy,
	).Scan(scanBackup(&item)...)
	if err != nil {
		return moduleapi.Backup{}, fmt.Errorf("create backup: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) createForTask(ctx context.Context, input moduleapi.CreateBackupInput) (moduleapi.Backup, error) {
	var item moduleapi.Backup
	err := r.db.QueryRowContext(ctx, `INSERT INTO backups (
		task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
	ON CONFLICT (task_id) DO NOTHING
	RETURNING id, task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at, restore_code, restore_at`,
		input.TaskID, input.Purpose, moduleapi.BackupStatusAvailable, input.ConfigSnapshot.StorageRef, input.ConfigSnapshot.SHA256, input.ConfigSnapshot.SizeBytes,
		input.DatabaseDump.StorageRef, input.DatabaseDump.SHA256, input.DatabaseDump.SizeBytes, input.RetainUntil.UTC(), input.CreatedBy,
	).Scan(scanBackup(&item)...)
	if err == nil {
		return item, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return moduleapi.Backup{}, fmt.Errorf("create task backup: %w", err)
	}
	item, err = r.getByTaskID(ctx, input.TaskID)
	if err != nil {
		return moduleapi.Backup{}, err
	}
	if !matchesCreateInput(item, input) {
		return moduleapi.Backup{}, moduleapi.ErrBackupInvalidInput
	}
	return item, nil
}

func matchesCreateInput(item moduleapi.Backup, input moduleapi.CreateBackupInput) bool {
	return matchesTaskBackupIdentity(item, input) && matchesTaskBackupArtifacts(item, input) && matchesTaskBackupCreator(item.CreatedBy, input.CreatedBy)

}

func matchesTaskBackupIdentity(item moduleapi.Backup, input moduleapi.CreateBackupInput) bool {
	return item.TaskID != nil && *item.TaskID == input.TaskID && item.Purpose == input.Purpose && item.Status == moduleapi.BackupStatusAvailable && item.RetainUntil.Equal(input.RetainUntil.UTC())
}

func matchesTaskBackupArtifacts(item moduleapi.Backup, input moduleapi.CreateBackupInput) bool {
	return item.ConfigSnapshot == input.ConfigSnapshot && item.DatabaseDump == input.DatabaseDump
}

func matchesTaskBackupCreator(createdBy *uint64, inputCreatedBy *uint64) bool {
	if createdBy == nil || inputCreatedBy == nil {
		return createdBy == nil && inputCreatedBy == nil
	}
	return *createdBy == *inputCreatedBy
}

// Get 返回一个未软删除的内部备份事实。
func (r *SQLRepository) Get(ctx context.Context, id uint64) (moduleapi.Backup, error) {
	if id == 0 {
		return moduleapi.Backup{}, moduleapi.ErrBackupInvalidInput
	}
	var item moduleapi.Backup
	err := r.db.QueryRowContext(ctx, `SELECT id, task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at, restore_code, restore_at
		FROM backups WHERE id = $1 AND deleted_at = 0`, id).Scan(scanBackup(&item)...)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.Backup{}, moduleapi.ErrBackupNotFound
	}
	if err != nil {
		return moduleapi.Backup{}, fmt.Errorf("get backup: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) getByTaskID(ctx context.Context, taskID uint64) (moduleapi.Backup, error) {
	var item moduleapi.Backup
	err := r.db.QueryRowContext(ctx, `SELECT id, task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at, restore_code, restore_at
		FROM backups WHERE task_id = $1 AND deleted_at = 0`, taskID).Scan(scanBackup(&item)...)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.Backup{}, moduleapi.ErrBackupNotFound
	}
	if err != nil {
		return moduleapi.Backup{}, fmt.Errorf("get task backup: %w", err)
	}
	return item, nil
}

// RecordRestoreEvidence 记录受控恢复流程已经形成的结果代码。
func (r *SQLRepository) RecordRestoreEvidence(ctx context.Context, input moduleapi.RecordBackupRestoreInput) (moduleapi.Backup, error) {
	input, err := normalizeRestore(input)
	if err != nil {
		return moduleapi.Backup{}, err
	}
	var item moduleapi.Backup
	err = r.db.QueryRowContext(ctx, `UPDATE backups SET status = $1, restore_code = $2, restore_at = $3, updated_at = CURRENT_TIMESTAMP
		WHERE id = $4 AND deleted_at = 0
		RETURNING id, task_id, purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
			database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at, restore_code, restore_at`,
		input.Status, input.RestoreCode, input.RecordedAt.UTC(), input.ID,
	).Scan(scanBackup(&item)...)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.Backup{}, moduleapi.ErrBackupNotFound
	}
	if err != nil {
		return moduleapi.Backup{}, fmt.Errorf("record backup restore evidence: %w", err)
	}
	return item, nil
}

func normalizeCreate(input moduleapi.CreateBackupInput) (moduleapi.CreateBackupInput, error) {
	input.Purpose = strings.TrimSpace(input.Purpose)
	if len(input.Purpose) == 0 || len(input.Purpose) > maxPurposeLength || input.RetainUntil.IsZero() || !input.RetainUntil.After(time.Now().UTC()) {
		return input, moduleapi.ErrBackupInvalidInput
	}
	if err := normalizeArtifact(&input.ConfigSnapshot); err != nil {
		return input, err
	}
	if err := normalizeArtifact(&input.DatabaseDump); err != nil {
		return input, err
	}
	return input, nil
}

func normalizeArtifact(artifact *moduleapi.BackupArtifact) error {
	artifact.StorageRef = strings.TrimSpace(artifact.StorageRef)
	artifact.SHA256 = strings.ToLower(strings.TrimSpace(artifact.SHA256))
	if len(artifact.StorageRef) == 0 || len(artifact.StorageRef) > maxStorageRefLength || artifact.SizeBytes < 0 || !validSHA256(artifact.SHA256) {
		return moduleapi.ErrBackupInvalidInput
	}
	return nil
}

func normalizeRestore(input moduleapi.RecordBackupRestoreInput) (moduleapi.RecordBackupRestoreInput, error) {
	input.RestoreCode = strings.TrimSpace(input.RestoreCode)
	if input.ID == 0 || input.RecordedAt.IsZero() || len(input.RestoreCode) == 0 || len(input.RestoreCode) > maxRestoreCodeLength {
		return input, moduleapi.ErrBackupInvalidInput
	}
	if input.Status != moduleapi.BackupStatusRestored && input.Status != moduleapi.BackupStatusExpired {
		return input, moduleapi.ErrBackupInvalidInput
	}
	return input, nil
}

func validSHA256(value string) bool {
	if len(value) != sha256Length {
		return false
	}
	for _, char := range value {
		if (char < '0' || char > '9') && (char < 'a' || char > 'f') {
			return false
		}
	}
	return true
}

func scanBackup(item *moduleapi.Backup) []any {
	return []any{
		&item.ID, &item.TaskID, &item.Purpose, &item.Status,
		&item.ConfigSnapshot.StorageRef, &item.ConfigSnapshot.SHA256, &item.ConfigSnapshot.SizeBytes,
		&item.DatabaseDump.StorageRef, &item.DatabaseDump.SHA256, &item.DatabaseDump.SizeBytes,
		&item.RetainUntil, &item.CreatedBy, &item.CreatedAt, &item.UpdatedAt, &item.RestoreCode, &item.RestoreAt,
	}
}
