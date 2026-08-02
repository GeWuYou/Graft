package update

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

// OperationStore 保存 Update 自己的编排事实，不替代 Task 或 Backup 的持久化职责。
type OperationStore interface {
	Create(context.Context, ComposeUpdateOperation) error
	Get(context.Context, string) (ComposeUpdateOperation, error)
	List(context.Context, int) ([]ComposeUpdateOperation, error)
	Settle(context.Context, ComposeUpdateOperation) error
	ClaimRecovery(context.Context, string, string) (bool, error)
	ReleaseRecoveryClaim(context.Context, string, string) error
	// RecoveryClaim 返回操作当前的恢复认领；未认领时返回空字符串，操作不存在时返回 errUpdateOperationNotFound，查询失败时返回数据库错误。
	RecoveryClaim(context.Context, string) (string, error)
}

var errUpdateOperationNotFound = errors.New("update operation not found")

type sqlOperationStore struct{ db *sql.DB }

func newSQLOperationStore(db *sql.DB) (OperationStore, error) {
	if db == nil {
		return nil, errors.New("update operation database is unavailable")
	}
	return &sqlOperationStore{db: db}, nil
}

func (s *sqlOperationStore) Create(ctx context.Context, value ComposeUpdateOperation) error {
	if s == nil || s.db == nil || !validOperation(value) || !validDeploymentStrategy(value.DeploymentStrategy) || value.TaskID == 0 {
		return errors.New("update operation is invalid")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO update_operations
	 (operation_id, runner_id, request_id, source_version, target_version, deployment_strategy, task_id, requested_by, status, created_at, started_at, updated_at)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, value.OperationID, nullableString(value.RunnerID), nullableString(value.RequestID), value.SourceVersion,
		value.TargetVersion, value.DeploymentStrategy, value.TaskID, nullableUint64(value.RequestedBy), value.Outcome)
	if err != nil {
		return fmt.Errorf("create update operation: %w", err)
	}
	return nil
}

func (s *sqlOperationStore) Get(ctx context.Context, operationID string) (ComposeUpdateOperation, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) {
		return ComposeUpdateOperation{}, errors.New("update operation identity is invalid")
	}
	return scanOperation(s.db.QueryRowContext(ctx, `SELECT operation_id, runner_id, request_id, source_version, target_version, deployment_strategy, task_id,
 backup_id, requested_by, status, receipt_integrity_sha256, failure_code, recovery_completed,
	 created_at, started_at, updated_at, finished_at,
 EXISTS(SELECT 1 FROM update_failure_diagnostics WHERE update_failure_diagnostics.operation_id = update_operations.operation_id)
 FROM update_operations WHERE operation_id = $1`, operationID))
}

func (s *sqlOperationStore) List(ctx context.Context, limit int) ([]ComposeUpdateOperation, error) {
	if s == nil || s.db == nil || limit < 1 || limit > 100 {
		return nil, errors.New("update operation list limit is invalid")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT operation_id, runner_id, request_id, source_version, target_version, deployment_strategy, task_id,
 backup_id, requested_by, status, receipt_integrity_sha256, failure_code, recovery_completed,
	 created_at, started_at, updated_at, finished_at,
 EXISTS(SELECT 1 FROM update_failure_diagnostics WHERE update_failure_diagnostics.operation_id = update_operations.operation_id)
 FROM update_operations ORDER BY created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, fmt.Errorf("list update operations: %w", err)
	}
	defer func() { _ = rows.Close() }()
	items := make([]ComposeUpdateOperation, 0)
	for rows.Next() {
		item, err := scanOperation(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate update operations: %w", err)
	}
	return items, nil
}

func (s *sqlOperationStore) Settle(ctx context.Context, value ComposeUpdateOperation) error {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(value.OperationID) || !runnerOperationID.MatchString(value.RunnerID) || !validOutcome(value.Outcome) {
		return errors.New("settled update operation is invalid")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE update_operations SET runner_id = $1, backup_id = $2, status = $3,
	 receipt_integrity_sha256 = $4, failure_code = $5, recovery_completed = $6, updated_at = CURRENT_TIMESTAMP, finished_at = CURRENT_TIMESTAMP
	WHERE operation_id = $7`, nullableString(value.RunnerID), nullableUint64(value.BackupID), value.Outcome, value.ReceiptIntegritySHA256,
		nullableString(value.FailureCode), value.RecoveryCompleted, value.OperationID)
	if err != nil {
		return fmt.Errorf("settle update operation: %w", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read settled update operation count: %w", err)
	}
	if n == 0 {
		return errUpdateOperationNotFound
	}
	return nil
}

// ClaimRecovery 原子保留一个未终态操作供恢复启动使用。
// 认领只是协调证据；runner 生命周期阶段仍由状态卷拥有。
func (s *sqlOperationStore) ClaimRecovery(ctx context.Context, operationID, claimID string) (bool, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) || strings.TrimSpace(claimID) == "" {
		return false, errors.New("update recovery claim is invalid")
	}
	var claimedOperationID string
	err := s.db.QueryRowContext(ctx, `UPDATE update_operations
		SET recovery_claim_id = $2, recovery_claimed_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		WHERE operation_id = $1 AND recovery_claim_id IS NULL AND finished_at IS NULL
		RETURNING operation_id`, operationID, claimID).Scan(&claimedOperationID)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("claim update operation recovery: %w", err)
	}
	return claimedOperationID == operationID, nil
}

// ReleaseRecoveryClaim 只在已证明容器尚未启动的失败后清除调用方认领。
func (s *sqlOperationStore) ReleaseRecoveryClaim(ctx context.Context, operationID, claimID string) error {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) || strings.TrimSpace(claimID) == "" {
		return errors.New("update recovery claim is invalid")
	}
	_, err := s.db.ExecContext(ctx, `UPDATE update_operations
		SET recovery_claim_id = NULL, recovery_claimed_at = NULL, updated_at = CURRENT_TIMESTAMP
		WHERE operation_id = $1 AND recovery_claim_id = $2`, operationID, claimID)
	if err != nil {
		return fmt.Errorf("release update operation recovery claim: %w", err)
	}
	return nil
}

// RecoveryClaim 返回操作当前的恢复认领，用于在进程中断后核验 claim 绑定容器是否已经创建。
func (s *sqlOperationStore) RecoveryClaim(ctx context.Context, operationID string) (string, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) {
		return "", errors.New("update recovery claim query is invalid")
	}
	var claim sql.NullString
	if err := s.db.QueryRowContext(ctx, `SELECT recovery_claim_id FROM update_operations WHERE operation_id = $1`, operationID).Scan(&claim); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errUpdateOperationNotFound
		}
		return "", fmt.Errorf("read update operation recovery claim: %w", err)
	}
	return strings.TrimSpace(claim.String), nil
}

type operationScanner interface{ Scan(...any) error }

func scanOperation(row operationScanner) (ComposeUpdateOperation, error) {
	var item ComposeUpdateOperation
	var backupID, requestedBy sql.NullInt64
	var runnerID, requestID, integrity, failure sql.NullString
	var created, started, updated time.Time
	var finished sql.NullTime
	if err := row.Scan(&item.OperationID, &runnerID, &requestID, &item.SourceVersion, &item.TargetVersion, &item.DeploymentStrategy, &item.TaskID, &backupID,
		&requestedBy, &item.Outcome, &integrity, &failure, &item.RecoveryCompleted, &created, &started, &updated, &finished, &item.FailureDiagnosticAvailable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ComposeUpdateOperation{}, errUpdateOperationNotFound
		}
		return ComposeUpdateOperation{}, fmt.Errorf("scan update operation: %w", err)
	}
	assignOperationNullableFields(&item, operationNullableFields{runnerID: runnerID, requestID: requestID, backupID: backupID, requestedBy: requestedBy, integrity: integrity, failure: failure, finished: finished})
	item.CreatedAt, item.StartedAt, item.UpdatedAt = created.UTC(), started.UTC(), updated.UTC()
	return item, nil
}

type operationNullableFields struct {
	runnerID, requestID, integrity, failure sql.NullString
	backupID, requestedBy                   sql.NullInt64
	finished                                sql.NullTime
}

//nolint:cyclop // 可空持久化字段必须独立映射，以保持零值语义。
func assignOperationNullableFields(item *ComposeUpdateOperation, values operationNullableFields) {
	if item == nil {
		return
	}
	if values.backupID.Valid && values.backupID.Int64 > 0 {
		item.BackupID = uint64(values.backupID.Int64)
	}
	if values.requestedBy.Valid && values.requestedBy.Int64 > 0 {
		item.RequestedBy = uint64(values.requestedBy.Int64)
	}
	if values.requestID.Valid {
		item.RequestID = values.requestID.String
	}
	if values.runnerID.Valid {
		item.RunnerID = values.runnerID.String
	}
	if values.integrity.Valid {
		item.ReceiptIntegritySHA256 = values.integrity.String
	}
	if values.failure.Valid {
		item.FailureCode = values.failure.String
	}
	if values.finished.Valid {
		value := values.finished.Time.UTC()
		item.FinishedAt = &value
	}
}

func nullableUint64(value uint64) any {
	if value == 0 {
		return nil
	}
	return value
}
func nullableString(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func validOutcome(value ExecutionOutcome) bool {
	switch value {
	case ExecutionOutcomePlanning, ExecutionOutcomeBackingUp, ExecutionOutcomePulling, ExecutionOutcomeMigrating, ExecutionOutcomeRecreating, ExecutionOutcomeVerifying, ExecutionOutcomeSuccess, ExecutionOutcomeFailed, ExecutionOutcomeRecovered, ExecutionOutcomeNeedsAttention:
		return true
	}
	return false
}
