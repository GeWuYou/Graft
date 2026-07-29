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
	if s == nil || s.db == nil || !validOperation(value) || value.TaskID == 0 {
		return errors.New("update operation is invalid")
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO update_operations
	 (operation_id, request_id, source_version, target_version, update_mode, task_id, requested_by, status, created_at, started_at)
	 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`, value.OperationID, nullableString(value.RequestID), value.SourceVersion,
		value.TargetVersion, value.UpdateMode, value.TaskID, nullableUint64(value.RequestedBy), value.Outcome)
	if err != nil {
		return fmt.Errorf("create update operation: %w", err)
	}
	return nil
}

func (s *sqlOperationStore) Get(ctx context.Context, operationID string) (ComposeUpdateOperation, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) {
		return ComposeUpdateOperation{}, errors.New("update operation identity is invalid")
	}
	return scanOperation(s.db.QueryRowContext(ctx, `SELECT operation_id, request_id, source_version, target_version, update_mode, task_id,
 backup_id, requested_by, status, receipt_integrity_sha256, failure_code, recovery_completed,
	 created_at, started_at, finished_at,
 EXISTS(SELECT 1 FROM update_failure_diagnostics WHERE update_failure_diagnostics.operation_id = update_operations.operation_id)
 FROM update_operations WHERE operation_id = $1`, operationID))
}

func (s *sqlOperationStore) List(ctx context.Context, limit int) ([]ComposeUpdateOperation, error) {
	if s == nil || s.db == nil || limit < 1 || limit > 100 {
		return nil, errors.New("update operation list limit is invalid")
	}
	rows, err := s.db.QueryContext(ctx, `SELECT operation_id, request_id, source_version, target_version, update_mode, task_id,
 backup_id, requested_by, status, receipt_integrity_sha256, failure_code, recovery_completed,
 created_at, started_at, finished_at,
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
	if s == nil || s.db == nil || !runnerOperationID.MatchString(value.OperationID) || !validOutcome(value.Outcome) {
		return errors.New("settled update operation is invalid")
	}
	result, err := s.db.ExecContext(ctx, `UPDATE update_operations SET backup_id = $1, status = $2,
 receipt_integrity_sha256 = $3, failure_code = $4, recovery_completed = $5, finished_at = CURRENT_TIMESTAMP
	WHERE operation_id = $6`, nullableUint64(value.BackupID), value.Outcome, value.ReceiptIntegritySHA256,
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

type operationScanner interface{ Scan(...any) error }

func scanOperation(row operationScanner) (ComposeUpdateOperation, error) {
	var item ComposeUpdateOperation
	var backupID, requestedBy sql.NullInt64
	var requestID, integrity, failure sql.NullString
	var created, started time.Time
	var finished sql.NullTime
	if err := row.Scan(&item.OperationID, &requestID, &item.SourceVersion, &item.TargetVersion, &item.UpdateMode, &item.TaskID, &backupID,
		&requestedBy, &item.Outcome, &integrity, &failure, &item.RecoveryCompleted, &created, &started, &finished, &item.FailureDiagnosticAvailable); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ComposeUpdateOperation{}, errUpdateOperationNotFound
		}
		return ComposeUpdateOperation{}, fmt.Errorf("scan update operation: %w", err)
	}
	assignOperationNullableFields(&item, operationNullableFields{requestID: requestID, backupID: backupID, requestedBy: requestedBy, integrity: integrity, failure: failure, finished: finished})
	item.CreatedAt, item.StartedAt = created.UTC(), started.UTC()
	return item, nil
}

type operationNullableFields struct {
	requestID, integrity, failure sql.NullString
	backupID, requestedBy         sql.NullInt64
	finished                      sql.NullTime
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
