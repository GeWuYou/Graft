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

// OperationProgressStore 是 OperationStore 的可选扩展，供 runner 日志恢复推进非终态阶段。
type OperationProgressStore interface {
	Advance(context.Context, string, ExecutionOutcome) (ComposeUpdateOperation, bool, error)
}

var errUpdateOperationNotFound = errors.New("update operation not found")

const updateOperationStatusPlaceholderStart = 3

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

// Advance 持久化 runner 已证明的非终态阶段，并拒绝倒退、重复和任何终态覆盖。
func (s *sqlOperationStore) Advance(ctx context.Context, operationID string, outcome ExecutionOutcome) (ComposeUpdateOperation, bool, error) {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(operationID) || !isProgressOutcome(outcome) {
		return ComposeUpdateOperation{}, false, errors.New("update operation progress is invalid")
	}
	allowed := progressPredecessors(outcome)
	query := `UPDATE update_operations SET status = $1, updated_at = CURRENT_TIMESTAMP WHERE operation_id = $2 AND status IN (` + placeholders(len(allowed), updateOperationStatusPlaceholderStart) + `) RETURNING operation_id, runner_id, request_id, source_version, target_version, deployment_strategy, task_id, backup_id, requested_by, status, receipt_integrity_sha256, failure_code, recovery_completed, created_at, started_at, updated_at, finished_at, EXISTS(SELECT 1 FROM update_failure_diagnostics WHERE update_failure_diagnostics.operation_id = update_operations.operation_id)`
	args := []any{outcome, operationID}
	for _, previous := range allowed {
		args = append(args, previous)
	}
	item, err := scanOperation(s.db.QueryRowContext(ctx, query, args...))
	if errors.Is(err, errUpdateOperationNotFound) {
		return ComposeUpdateOperation{}, false, nil
	}
	if err != nil {
		return ComposeUpdateOperation{}, false, err
	}
	return item, true, nil
}

func (s *sqlOperationStore) Settle(ctx context.Context, value ComposeUpdateOperation) error {
	if s == nil || s.db == nil || !runnerOperationID.MatchString(value.OperationID) || !validOutcome(value.Outcome) {
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

func isProgressOutcome(outcome ExecutionOutcome) bool {
	return outcome == ExecutionOutcomeBackingUp || outcome == ExecutionOutcomePulling || outcome == ExecutionOutcomeMigrating || outcome == ExecutionOutcomeRecreating || outcome == ExecutionOutcomeVerifying
}
func progressPredecessors(outcome ExecutionOutcome) []ExecutionOutcome {
	switch outcome {
	case ExecutionOutcomeBackingUp:
		return []ExecutionOutcome{ExecutionOutcomePlanning}
	case ExecutionOutcomePulling:
		return []ExecutionOutcome{ExecutionOutcomePlanning, ExecutionOutcomeBackingUp}
	case ExecutionOutcomeMigrating:
		return []ExecutionOutcome{ExecutionOutcomePlanning, ExecutionOutcomeBackingUp, ExecutionOutcomePulling}
	case ExecutionOutcomeRecreating:
		return []ExecutionOutcome{ExecutionOutcomePlanning, ExecutionOutcomeBackingUp, ExecutionOutcomePulling, ExecutionOutcomeMigrating}
	case ExecutionOutcomeVerifying:
		return []ExecutionOutcome{ExecutionOutcomePlanning, ExecutionOutcomeBackingUp, ExecutionOutcomePulling, ExecutionOutcomeMigrating, ExecutionOutcomeRecreating}
	default:
		return nil
	}
}
func placeholders(count, start int) string {
	values := make([]string, count)
	for index := range values {
		values[index] = fmt.Sprintf("$%d", index+start)
	}
	return strings.Join(values, ", ")
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
