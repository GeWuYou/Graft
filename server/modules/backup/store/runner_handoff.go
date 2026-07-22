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

const maxRunnerOperationLength = 128

// PrepareRunnerHandoff 持久化一次性 runner 的冻结备份范围，阻止完成回执更换任务、工件路径或保留期。
func (r *SQLRepository) PrepareRunnerHandoff(ctx context.Context, plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	plan, err := normalizeRunnerPlan(plan)
	if err != nil {
		return moduleapi.BackupRunnerHandoffPlan{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO backup_runner_handoffs (
		operation_id, task_id, purpose, retain_until, created_by, artifact_root, config_snapshot_ref, database_dump_ref, status, created_at
	) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,'PLANNED',CURRENT_TIMESTAMP)`,
		plan.OperationID, plan.TaskID, plan.Purpose, plan.RetainUntil.UTC(), plan.CreatedBy, plan.ArtifactRoot, plan.ConfigSnapshotRef, plan.DatabaseDumpRef)
	if err != nil {
		return moduleapi.BackupRunnerHandoffPlan{}, fmt.Errorf("prepare backup runner handoff: %w", err)
	}
	return plan, nil
}

// GetRunnerHandoff 返回冻结的 handoff 计划及已结算备份标识；不返回任何工件内容。
func (r *SQLRepository) GetRunnerHandoff(ctx context.Context, operationID string, taskID uint64) (moduleapi.BackupRunnerHandoffPlan, uint64, error) {
	if strings.TrimSpace(operationID) == "" || taskID == 0 {
		return moduleapi.BackupRunnerHandoffPlan{}, 0, moduleapi.ErrBackupInvalidInput
	}
	var plan moduleapi.BackupRunnerHandoffPlan
	var backupID sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT operation_id, task_id, purpose, retain_until, created_by, artifact_root, config_snapshot_ref, database_dump_ref, backup_id
		FROM backup_runner_handoffs WHERE operation_id = $1 AND task_id = $2`, operationID, taskID).Scan(
		&plan.OperationID, &plan.TaskID, &plan.Purpose, &plan.RetainUntil, &plan.CreatedBy, &plan.ArtifactRoot, &plan.ConfigSnapshotRef, &plan.DatabaseDumpRef, &backupID)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.BackupRunnerHandoffPlan{}, 0, moduleapi.ErrBackupNotFound
	}
	if err != nil {
		return moduleapi.BackupRunnerHandoffPlan{}, 0, fmt.Errorf("get backup runner handoff: %w", err)
	}
	if backupID.Valid && backupID.Int64 > 0 {
		return plan, uint64(backupID.Int64), nil
	}
	return plan, 0, nil
}

// CompleteRunnerHandoff 原子创建 backup 事实并结算 handoff；相同 operation 的第二次提交返回既有备份证据。
func (r *SQLRepository) CompleteRunnerHandoff(ctx context.Context, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.BackupRunnerHandoffCompletion, error) {
	input, err := normalizeRunnerCompletion(input)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("begin backup runner completion: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	plan, backupID, err := getRunnerHandoffTx(ctx, tx, input.OperationID, input.TaskID)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	if backupID != 0 {
		return r.completedHandoff(ctx, tx, backupID, input, true)
	}
	backup, err := createRunnerBackup(ctx, tx, plan, input)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, err
	}
	result, err := tx.ExecContext(ctx, `UPDATE backup_runner_handoffs SET status = 'COMPLETED', backup_id = $1, completed_at = CURRENT_TIMESTAMP
		WHERE operation_id = $2 AND task_id = $3 AND status = 'PLANNED'`, backup.ID, input.OperationID, input.TaskID)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("complete backup runner handoff: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("count completed backup runner handoff: %w", err)
	}
	if updated != 1 {
		return moduleapi.BackupRunnerHandoffCompletion{}, moduleapi.ErrBackupInvalidInput
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("commit backup runner handoff: %w", err)
	}
	return runnerCompletion(backup.ID, input, false), nil
}

func getRunnerHandoffTx(ctx context.Context, tx *sql.Tx, operationID string, taskID uint64) (moduleapi.BackupRunnerHandoffPlan, uint64, error) {
	var plan moduleapi.BackupRunnerHandoffPlan
	var backupID sql.NullInt64
	err := tx.QueryRowContext(ctx, `SELECT operation_id, task_id, purpose, retain_until, created_by, artifact_root, config_snapshot_ref, database_dump_ref, backup_id
		FROM backup_runner_handoffs WHERE operation_id = $1 AND task_id = $2`, operationID, taskID).Scan(
		&plan.OperationID, &plan.TaskID, &plan.Purpose, &plan.RetainUntil, &plan.CreatedBy, &plan.ArtifactRoot, &plan.ConfigSnapshotRef, &plan.DatabaseDumpRef, &backupID)
	if errors.Is(err, sql.ErrNoRows) {
		return moduleapi.BackupRunnerHandoffPlan{}, 0, moduleapi.ErrBackupNotFound
	}
	if err != nil {
		return moduleapi.BackupRunnerHandoffPlan{}, 0, fmt.Errorf("lock backup runner handoff: %w", err)
	}
	if backupID.Valid && backupID.Int64 > 0 {
		return plan, uint64(backupID.Int64), nil
	}
	return plan, 0, nil
}

func createRunnerBackup(ctx context.Context, tx *sql.Tx, plan moduleapi.BackupRunnerHandoffPlan, input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.Backup, error) {
	var backup moduleapi.Backup
	err := tx.QueryRowContext(ctx, `INSERT INTO backups (
		purpose, status, config_snapshot_ref, config_snapshot_sha256, config_snapshot_bytes,
		database_dump_ref, database_dump_sha256, database_dump_bytes, retain_until, created_by, created_at, updated_at
	) VALUES ($1,'AVAILABLE',$2,$3,$4,$5,$6,$7,$8,$9,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP)
	RETURNING id`, plan.Purpose, plan.ConfigSnapshotRef, input.ConfigSnapshotSHA256, input.ConfigSnapshotBytes,
		plan.DatabaseDumpRef, input.DatabaseDumpSHA256, input.DatabaseDumpBytes, plan.RetainUntil.UTC(), plan.CreatedBy).Scan(&backup.ID)
	if err != nil {
		return moduleapi.Backup{}, fmt.Errorf("create runner backup: %w", err)
	}
	return backup, nil
}

func (r *SQLRepository) completedHandoff(ctx context.Context, tx *sql.Tx, backupID uint64, input moduleapi.CompleteBackupRunnerHandoffInput, idempotent bool) (moduleapi.BackupRunnerHandoffCompletion, error) {
	var configSHA, dumpSHA string
	var configBytes, dumpBytes int64
	err := tx.QueryRowContext(ctx, `SELECT config_snapshot_sha256, config_snapshot_bytes, database_dump_sha256, database_dump_bytes
		FROM backups WHERE id = $1 AND deleted_at = 0`, backupID).Scan(&configSHA, &configBytes, &dumpSHA, &dumpBytes)
	if err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("read completed runner backup: %w", err)
	}
	if configSHA != input.ConfigSnapshotSHA256 || configBytes != input.ConfigSnapshotBytes || dumpSHA != input.DatabaseDumpSHA256 || dumpBytes != input.DatabaseDumpBytes {
		return moduleapi.BackupRunnerHandoffCompletion{}, moduleapi.ErrBackupInvalidInput
	}
	if err := tx.Commit(); err != nil {
		return moduleapi.BackupRunnerHandoffCompletion{}, fmt.Errorf("commit replayed backup runner handoff: %w", err)
	}
	return runnerCompletion(backupID, input, idempotent), nil
}

func runnerCompletion(backupID uint64, input moduleapi.CompleteBackupRunnerHandoffInput, idempotent bool) moduleapi.BackupRunnerHandoffCompletion {
	return moduleapi.BackupRunnerHandoffCompletion{BackupID: backupID, OperationID: input.OperationID, TaskID: input.TaskID, ConfigSnapshotSHA256: input.ConfigSnapshotSHA256, ConfigSnapshotBytes: input.ConfigSnapshotBytes, DatabaseDumpSHA256: input.DatabaseDumpSHA256, DatabaseDumpBytes: input.DatabaseDumpBytes, Idempotent: idempotent}
}

func normalizeRunnerPlan(plan moduleapi.BackupRunnerHandoffPlan) (moduleapi.BackupRunnerHandoffPlan, error) {
	plan.OperationID = strings.TrimSpace(plan.OperationID)
	plan.Purpose = strings.TrimSpace(plan.Purpose)
	plan.ArtifactRoot = strings.TrimSpace(plan.ArtifactRoot)
	plan.ConfigSnapshotRef = strings.TrimSpace(plan.ConfigSnapshotRef)
	plan.DatabaseDumpRef = strings.TrimSpace(plan.DatabaseDumpRef)
	if !validRunnerPlanIdentity(plan) || !validRunnerPlanPolicy(plan) || !validRunnerPlanRefs(plan) {
		return plan, moduleapi.ErrBackupInvalidInput
	}
	return plan, nil
}

func validRunnerPlanIdentity(plan moduleapi.BackupRunnerHandoffPlan) bool {
	return plan.TaskID != 0 && len(plan.OperationID) > 0 && len(plan.OperationID) <= maxRunnerOperationLength
}

func validRunnerPlanPolicy(plan moduleapi.BackupRunnerHandoffPlan) bool {
	return len(plan.Purpose) > 0 && len(plan.Purpose) <= maxPurposeLength && !plan.RetainUntil.IsZero() && plan.RetainUntil.After(time.Now().UTC())
}

func validRunnerPlanRefs(plan moduleapi.BackupRunnerHandoffPlan) bool {
	return validRunnerRef(plan.ArtifactRoot) && validRunnerRef(plan.ConfigSnapshotRef) && validRunnerRef(plan.DatabaseDumpRef)
}

func validRunnerRef(ref string) bool {
	return len(ref) > 0 && len(ref) <= maxStorageRefLength
}

func normalizeRunnerCompletion(input moduleapi.CompleteBackupRunnerHandoffInput) (moduleapi.CompleteBackupRunnerHandoffInput, error) {
	input.OperationID = strings.TrimSpace(input.OperationID)
	input.ConfigSnapshotSHA256 = strings.ToLower(strings.TrimSpace(input.ConfigSnapshotSHA256))
	input.DatabaseDumpSHA256 = strings.ToLower(strings.TrimSpace(input.DatabaseDumpSHA256))
	if input.TaskID == 0 || len(input.OperationID) == 0 || len(input.OperationID) > maxRunnerOperationLength || input.ConfigSnapshotBytes < 0 || input.DatabaseDumpBytes < 0 || !validSHA256(strings.ToLower(input.ConfigSnapshotSHA256)) || !validSHA256(strings.ToLower(input.DatabaseDumpSHA256)) {
		return input, moduleapi.ErrBackupInvalidInput
	}
	return input, nil
}
