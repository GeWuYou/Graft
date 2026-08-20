package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgconn"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

const externalExecutionLeaseExpiryFailureCode = "external_execution_lease_expired"

// ListExternalExecutionCandidates 返回满足串行前置条件、且只能由 Runtime Agent 领取的 pending Stage。
func (r *SQLRepository) ListExternalExecutionCandidates(ctx context.Context, limit int, offset int) ([]StageClaim, error) {
	now := time.Now().UTC()
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE stage.status = ? AND stage.external_execution = true
			AND (stage.next_retry_at IS NULL OR stage.next_retry_at <= ?)
			AND task.status IN (?, ?) AND task.cancel_requested_at IS NULL
			AND (task.scheduled_at IS NULL OR task.scheduled_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = stage.task_id AND earlier.sequence < stage.sequence
					AND earlier.status NOT IN (?, ?, ?)
					AND (stage.coordination_group = '' OR earlier.coordination_group = '' OR earlier.coordination_group <> stage.coordination_group)
			)
			AND NOT EXISTS (
				SELECT 1 FROM task_external_execution_leases lease
				WHERE lease.task_id = stage.task_id AND lease.stage_id = stage.id AND lease.attempt = stage.attempt + 1
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		LIMIT ? OFFSET ?`), moduleapi.StageStatusPending, now, moduleapi.TaskStatusReady, moduleapi.TaskStatusRunning, now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled, normalizeLimit(limit), normalizeOffset(offset))
	if err != nil {
		return nil, fmt.Errorf("list external execution candidates: %w", err)
	}
	defer closeRows(rows)
	items := make([]StageClaim, 0)
	for rows.Next() {
		item, scanErr := scanStageClaim(rows)
		if scanErr != nil {
			return nil, fmt.Errorf("scan external execution candidate: %w", scanErr)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate external execution candidates: %w", err)
	}
	return items, nil
}

// CreateExternalExecutionLease 在同一事务中将外部 Stage 变为 running 并创建唯一的 fenced lease。
func (r *SQLRepository) CreateExternalExecutionLease(ctx context.Context, input CreateExternalExecutionLeaseInput) (taskmodel.ExternalExecutionLease, error) {
	lease := input.Lease
	if !validExternalExecutionLease(lease) {
		return taskmodel.ExternalExecutionLease{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, fmt.Errorf("begin external execution claim: %w", err)
	}
	defer rollback(tx)

	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = (SELECT stage_key FROM task_stages WHERE id = ? AND task_id = ?),
			started_at = COALESCE(started_at, ?), updated_at = ?
		WHERE id = ? AND status IN (?, ?) AND cancel_requested_at IS NULL
			AND (scheduled_at IS NULL OR scheduled_at <= ?)`),
		moduleapi.TaskStatusRunning, lease.StageID, lease.TaskID, now, now, lease.TaskID,
		moduleapi.TaskStatusReady, moduleapi.TaskStatusRunning, now)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, fmt.Errorf("mark external execution task running: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.ExternalExecutionLease{}, err
	}

	result, err = tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, attempt = ?, started_at = COALESCE(started_at, ?), updated_at = ?
		WHERE id = ? AND task_id = ? AND executor_type = ? AND external_execution = true
			AND status = ? AND attempt = ? AND attempt < max_attempts
			AND (next_retry_at IS NULL OR next_retry_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = task_stages.task_id AND earlier.sequence < task_stages.sequence
					AND earlier.status NOT IN (?, ?, ?)
					AND (task_stages.coordination_group = '' OR earlier.coordination_group = '' OR earlier.coordination_group <> task_stages.coordination_group)
			)`),
		moduleapi.StageStatusRunning, lease.Attempt, now, now, lease.StageID, lease.TaskID, lease.ExecutorType,
		moduleapi.StageStatusPending, lease.Attempt-1, now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, fmt.Errorf("mark external execution stage running: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.ExternalExecutionLease{}, err
	}

	_, err = tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_external_execution_leases (
		id, task_id, stage_id, attempt, executor_type, runtime_target_id, provider_id, capability, receipt_protocol,
		operation_id, payload_sha256, fence_token_hash, state, lease_ttl_ms, lease_expires_at, absolute_deadline_at,
		created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		lease.ID, lease.TaskID, lease.StageID, lease.Attempt, lease.ExecutorType, lease.RuntimeTargetID,
		lease.ProviderID, lease.Capability, lease.Protocol, lease.OperationID, lease.PayloadSHA256, lease.FenceTokenHash,
		lease.State, lease.LeaseTTL.Milliseconds(), lease.LeaseExpiresAt.UTC(), lease.AbsoluteDeadlineAt.UTC(), now, now)
	if err != nil {
		if externalExecutionLeaseConflict(err) {
			return taskmodel.ExternalExecutionLease{}, ErrStateConflict
		}
		return taskmodel.ExternalExecutionLease{}, fmt.Errorf("create external execution lease: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return taskmodel.ExternalExecutionLease{}, fmt.Errorf("commit external execution claim: %w", err)
	}
	lease.CreatedAt, lease.UpdatedAt = now, now
	return lease, nil
}

// GetExternalExecutionLease 返回续租所需的持久 lease 与当前取消请求事实。
func (r *SQLRepository) GetExternalExecutionLease(ctx context.Context, leaseID string) (taskmodel.ExternalExecutionLease, bool, error) {
	return r.readExternalExecutionLease(ctx, r.db, leaseID)
}

// RenewExternalExecutionLease 在 lease 与 absolute deadline 均有效时延长同一个 fenced attempt。
func (r *SQLRepository) RenewExternalExecutionLease(ctx context.Context, input RenewExternalExecutionLeaseInput) (taskmodel.ExternalExecutionLease, bool, error) {
	if strings.TrimSpace(input.ID) == "" || input.FenceTokenHash == "" || input.LeaseExpiresAt.IsZero() {
		return taskmodel.ExternalExecutionLease{}, false, ErrInvalidInput
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_external_execution_leases
		SET lease_expires_at = CASE WHEN ? < absolute_deadline_at THEN ? ELSE absolute_deadline_at END, updated_at = ?
		WHERE id = ? AND fence_token_hash = ? AND state = ? AND lease_expires_at > ? AND absolute_deadline_at > ?`),
		input.LeaseExpiresAt.UTC(), input.LeaseExpiresAt.UTC(), now, input.ID, input.FenceTokenHash,
		moduleapi.ExternalExecutionLeaseStateClaimed, now, now)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, false, fmt.Errorf("renew external execution lease: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.ExternalExecutionLease{}, false, err
	}
	if _, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_external_execution_leases
		SET cancel_observed_at = COALESCE(cancel_observed_at, ?), updated_at = ?
		WHERE id = ? AND EXISTS (SELECT 1 FROM tasks WHERE tasks.id = task_external_execution_leases.task_id AND tasks.cancel_requested_at IS NOT NULL)`),
		now, now, input.ID); err != nil {
		return taskmodel.ExternalExecutionLease{}, false, fmt.Errorf("observe external execution cancellation: %w", err)
	}
	lease, cancelRequested, err := r.readExternalExecutionLease(ctx, r.db, input.ID)
	return lease, cancelRequested, err
}

// AppendExternalExecutionLogs 在验证 lease fence 后为同一 Stage 原子分配 Task 日志序列。
func (r *SQLRepository) AppendExternalExecutionLogs(ctx context.Context, input AppendExternalExecutionLogsInput) error {
	if !validExternalExecutionLogInput(input) {
		return ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin external execution logs: %w", err)
	}
	defer rollback(tx)
	if err := r.appendExternalExecutionLogsTx(ctx, tx, input); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit external execution logs: %w", err)
	}
	return nil
}

func (r *SQLRepository) appendExternalExecutionLogsTx(ctx context.Context, tx *sql.Tx, input AppendExternalExecutionLogsInput) error {
	lease, _, err := r.readExternalExecutionLease(ctx, tx, input.LeaseID)
	if err != nil {
		return err
	}
	if lease.FenceTokenHash != input.FenceTokenHash || lease.State != moduleapi.ExternalExecutionLeaseStateClaimed || !lease.LeaseExpiresAt.After(time.Now().UTC()) {
		return ErrStateConflict
	}
	if err := r.lockSettlementTask(ctx, tx, lease.TaskID); err != nil {
		return err
	}
	var sequence int64
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COALESCE(MAX(sequence), 0) FROM task_logs WHERE task_id = ?`), lease.TaskID).Scan(&sequence); err != nil {
		return fmt.Errorf("read external execution log sequence: %w", err)
	}
	occurredAt := input.OccurredAt.UTC()
	if occurredAt.IsZero() {
		occurredAt = time.Now().UTC()
	}
	return r.insertExternalExecutionLogs(ctx, tx, lease, input.Entries, sequence, occurredAt)
}

func (r *SQLRepository) insertExternalExecutionLogs(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, entries []moduleapi.TaskLogEntry, sequence int64, occurredAt time.Time) error {
	for _, entry := range entries {
		if !validLogStream(entry.Stream) || !validLogLevel(entry.Level) || strings.TrimSpace(entry.Line) == "" {
			return ErrInvalidInput
		}
		sequence++
		if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_logs (
			task_id, stage_id, sequence, stream, level, line, occurred_at
		) VALUES (?, ?, ?, ?, ?, ?, ?)`), lease.TaskID, lease.StageID, sequence, entry.Stream, entry.Level, entry.Line, occurredAt); err != nil {
			return fmt.Errorf("append external execution log: %w", err)
		}
	}
	return nil
}

// SettleExternalExecution 原子结算一个 fenced lease、Stage 和父 Task，并允许非最终 Stage 成功后继续执行。
func (r *SQLRepository) SettleExternalExecution(ctx context.Context, input SettleExternalExecutionInput) (ExternalReceiptSettlement, error) {
	if !validExternalExecutionSettlement(input) {
		return ExternalReceiptSettlement{}, ErrInvalidInput
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("begin external execution settlement: %w", err)
	}
	defer rollback(tx)
	lease, replay, err := r.prepareExternalExecutionSettlement(ctx, tx, input)
	if err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if replay != nil {
		return *replay, tx.Commit()
	}
	return r.persistExternalExecutionSettlement(ctx, tx, lease, input)
}

func (r *SQLRepository) prepareExternalExecutionSettlement(ctx context.Context, tx *sql.Tx, input SettleExternalExecutionInput) (taskmodel.ExternalExecutionLease, *ExternalReceiptSettlement, error) {
	lease, _, err := r.readExternalExecutionLease(ctx, tx, input.LeaseID)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, nil, err
	}
	if lease.FenceTokenHash != input.FenceTokenHash {
		return taskmodel.ExternalExecutionLease{}, nil, ErrStateConflict
	}
	if replay, found, err := r.existingLeaseSettlement(ctx, tx, lease, input); err != nil {
		return taskmodel.ExternalExecutionLease{}, nil, err
	} else if found {
		return taskmodel.ExternalExecutionLease{}, &replay, nil
	}
	if err := r.lockSettlementTask(ctx, tx, lease.TaskID); err != nil {
		return taskmodel.ExternalExecutionLease{}, nil, err
	}
	lease, _, err = r.readExternalExecutionLease(ctx, tx, input.LeaseID)
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, nil, err
	}
	if lease.FenceTokenHash != input.FenceTokenHash || (lease.State != moduleapi.ExternalExecutionLeaseStateClaimed && lease.State != moduleapi.ExternalExecutionLeaseStateExpired) {
		return taskmodel.ExternalExecutionLease{}, nil, ErrStateConflict
	}
	return lease, nil, nil
}

func (r *SQLRepository) persistExternalExecutionSettlement(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, input SettleExternalExecutionInput) (ExternalReceiptSettlement, error) {
	now := time.Now().UTC()
	stageStatus := externalExecutionStageStatus(input.Outcome)
	var failureCode any
	if input.FailureCode != "" {
		failureCode = input.FailureCode
	}
	stageResult, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, failure_code = ?, failure_message = NULL, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND task_id = ? AND attempt = ? AND executor_type = ? AND status IN (?, ?)`),
		stageStatus, failureCode, now, now, lease.StageID, lease.TaskID, lease.Attempt, lease.ExecutorType,
		moduleapi.StageStatusRunning, moduleapi.StageStatusUnknown)
	if err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("settle external execution stage: %w", err)
	}
	if err := expectOneAffected(stageResult); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	taskStatus, err := r.externalExecutionTaskStatus(ctx, tx, lease, input.Outcome)
	if err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := r.updateExternalExecutionTask(ctx, tx, lease, taskStatus, failureCode, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_external_execution_leases
		SET state = ?, settled_at = ?, updated_at = ? WHERE id = ? AND state IN (?, ?)`),
		moduleapi.ExternalExecutionLeaseStateSettled, now, now, lease.ID,
		moduleapi.ExternalExecutionLeaseStateClaimed, moduleapi.ExternalExecutionLeaseStateExpired); err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("settle external execution lease: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_external_receipts (
		lease_id, task_id, stage_id, attempt, executor_type, receipt_protocol, operation_id, outcome, failure_code,
		integrity_sha256, settled_task_status, created_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`), lease.ID, lease.TaskID, lease.StageID, lease.Attempt,
		lease.ExecutorType, lease.Protocol, lease.OperationID, input.Outcome, failureCode, input.IntegritySHA256,
		taskStatus, now); err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("insert external execution receipt: %w", err)
	}
	if err := r.appendExternalExecutionEvent(ctx, tx, lease, input, taskStatus, now); err != nil {
		return ExternalReceiptSettlement{}, err
	}
	if err := tx.Commit(); err != nil {
		return ExternalReceiptSettlement{}, fmt.Errorf("commit external execution settlement: %w", err)
	}
	return ExternalReceiptSettlement{TaskID: lease.TaskID, StageID: lease.StageID, Status: taskStatus}, nil
}

func (r *SQLRepository) externalExecutionTaskStatus(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, outcome moduleapi.ExternalReceiptOutcome) (moduleapi.TaskStatus, error) {
	if outcome == moduleapi.ExternalReceiptOutcomeFailed {
		return moduleapi.TaskStatusFailed, nil
	}
	if outcome == moduleapi.ExternalReceiptOutcomeNeedsAttention {
		return moduleapi.TaskStatusNeedsAttention, nil
	}
	var cancelRequested sql.NullTime
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT cancel_requested_at FROM tasks WHERE id = ?`), lease.TaskID).Scan(&cancelRequested); err != nil {
		return "", fmt.Errorf("read external execution cancellation: %w", err)
	}
	if cancelRequested.Valid {
		if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages SET status = ?, finished_at = ?, updated_at = ?
			WHERE task_id = ? AND status = ?`), moduleapi.StageStatusCancelled, time.Now().UTC(), time.Now().UTC(), lease.TaskID, moduleapi.StageStatusPending); err != nil {
			return "", fmt.Errorf("cancel remaining external execution stages: %w", err)
		}
		return moduleapi.TaskStatusCancelled, nil
	}
	var remaining int
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COUNT(*) FROM task_stages
		WHERE task_id = ? AND status NOT IN (?, ?, ?)`), lease.TaskID,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled).Scan(&remaining); err != nil {
		return "", fmt.Errorf("count remaining external execution stages: %w", err)
	}
	if remaining == 0 {
		return moduleapi.TaskStatusSuccess, nil
	}
	return moduleapi.TaskStatusRunning, nil
}

func (r *SQLRepository) updateExternalExecutionTask(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, status moduleapi.TaskStatus, failureCode any, now time.Time) error {
	finishedAt := any(nil)
	if status != moduleapi.TaskStatusRunning {
		finishedAt = now
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = (SELECT stage_key FROM task_stages WHERE id = ?), failure_code = ?,
			failure_message = NULL, finished_at = ?, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`), status, lease.StageID, failureCode, finishedAt, now, lease.TaskID,
		moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention)
	if err != nil {
		return fmt.Errorf("settle external execution task: %w", err)
	}
	return expectOneAffected(result)
}

func (r *SQLRepository) appendExternalExecutionEvent(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, input SettleExternalExecutionInput, status moduleapi.TaskStatus, now time.Time) error {
	payload, err := json.Marshal(map[string]any{"lease_id": lease.ID, "stage_id": lease.StageID, "attempt": lease.Attempt, "operation_id": lease.OperationID, "protocol": lease.Protocol, "outcome": input.Outcome, "task_status": status})
	if err != nil {
		return fmt.Errorf("marshal external execution event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, ?, ? FROM task_events WHERE task_id = ?`),
		lease.TaskID, taskmodel.EventTypeExternalReceiptSettled, payload, now, lease.TaskID); err != nil {
		return fmt.Errorf("append external execution event: %w", err)
	}
	return nil
}

func (r *SQLRepository) existingLeaseSettlement(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, input SettleExternalExecutionInput) (ExternalReceiptSettlement, bool, error) {
	var outcome, failureCode, digest, status string
	err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT outcome, COALESCE(failure_code, ''), integrity_sha256, settled_task_status
		FROM task_external_receipts WHERE lease_id = ?`), lease.ID).Scan(&outcome, &failureCode, &digest, &status)
	if errors.Is(err, sql.ErrNoRows) {
		return ExternalReceiptSettlement{}, false, nil
	}
	if err != nil {
		return ExternalReceiptSettlement{}, false, fmt.Errorf("find external execution receipt: %w", err)
	}
	if outcome != string(input.Outcome) || failureCode != input.FailureCode || digest != input.IntegritySHA256 {
		return ExternalReceiptSettlement{}, false, ErrStateConflict
	}
	return ExternalReceiptSettlement{TaskID: lease.TaskID, StageID: lease.StageID, Status: moduleapi.TaskStatus(status), Idempotent: true}, true, nil
}

// ExpireExternalExecutionLeases 将超时 running attempt 收敛为 unknown/needs_attention，且不重新分配外部副作用。
func (r *SQLRepository) ExpireExternalExecutionLeases(ctx context.Context, now time.Time, limit int) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin external execution expiry: %w", err)
	}
	defer rollback(tx)
	ids, err := r.listExpiredExternalExecutionLeaseIDs(ctx, tx, now.UTC(), limit)
	if err != nil {
		return 0, err
	}
	expired, err := r.expireExternalExecutionLeaseBatch(ctx, tx, ids, now.UTC())
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit external execution expiry: %w", err)
	}
	return expired, nil
}

func (r *SQLRepository) listExpiredExternalExecutionLeaseIDs(ctx context.Context, tx *sql.Tx, now time.Time, limit int) ([]string, error) {
	query := `SELECT id FROM task_external_execution_leases WHERE state = ? AND (lease_expires_at <= ? OR absolute_deadline_at <= ?)
		ORDER BY lease_expires_at ASC, id ASC LIMIT ?`
	rows, err := tx.QueryContext(ctx, r.placeholder.rebind(query), moduleapi.ExternalExecutionLeaseStateClaimed, now, now, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("select expired external execution leases: %w", err)
	}
	defer closeRows(rows)
	ids := make([]string, 0)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan expired external execution lease: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired external execution leases: %w", err)
	}
	return ids, nil
}

func (r *SQLRepository) expireExternalExecutionLeaseBatch(ctx context.Context, tx *sql.Tx, ids []string, now time.Time) (int, error) {
	expired := 0
	for _, id := range ids {
		lease, _, err := r.readExternalExecutionLease(ctx, tx, id)
		if err != nil {
			return 0, err
		}
		settled, err := r.expireExternalExecutionLease(ctx, tx, lease, now)
		if err != nil {
			return 0, err
		}
		if settled {
			expired++
		}
	}
	return expired, nil
}

func (r *SQLRepository) expireExternalExecutionLease(ctx context.Context, tx *sql.Tx, lease taskmodel.ExternalExecutionLease, now time.Time) (bool, error) {
	if err := r.lockSettlementTask(ctx, tx, lease.TaskID); err != nil {
		return false, err
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_external_execution_leases SET state = ?, updated_at = ?
		WHERE id = ? AND state = ? AND (lease_expires_at <= ? OR absolute_deadline_at <= ?)`),
		moduleapi.ExternalExecutionLeaseStateExpired, now, lease.ID, moduleapi.ExternalExecutionLeaseStateClaimed, now, now)
	if err != nil {
		return false, fmt.Errorf("expire external execution lease: %w", err)
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("count expired external execution lease: %w", err)
	}
	if affected == 0 {
		return false, nil
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages SET status = ?, failure_code = ?, failure_message = NULL,
		finished_at = ?, duration_ms = NULL, updated_at = ? WHERE id = ? AND task_id = ? AND attempt = ? AND status = ?`),
		moduleapi.StageStatusUnknown, externalExecutionLeaseExpiryFailureCode, now, now, lease.StageID, lease.TaskID, lease.Attempt, moduleapi.StageStatusRunning); err != nil {
		return false, fmt.Errorf("mark expired external execution stage unknown: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks SET status = ?, current_stage_key = (SELECT stage_key FROM task_stages WHERE id = ?),
		failure_code = ?, failure_message = NULL, finished_at = ?, duration_ms = NULL, updated_at = ? WHERE id = ? AND status = ?`),
		moduleapi.TaskStatusNeedsAttention, lease.StageID, externalExecutionLeaseExpiryFailureCode, now, now, lease.TaskID, moduleapi.TaskStatusRunning); err != nil {
		return false, fmt.Errorf("mark expired external execution task needs attention: %w", err)
	}
	payload, err := json.Marshal(map[string]any{"lease_id": lease.ID, "stage_id": lease.StageID, "attempt": lease.Attempt, "reason": externalExecutionLeaseExpiryFailureCode})
	if err != nil {
		return false, fmt.Errorf("marshal external execution expiry event: %w", err)
	}
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at)
		SELECT ?, COALESCE(MAX(sequence), 0) + 1, ?, ?, ? FROM task_events WHERE task_id = ?`),
		lease.TaskID, taskmodel.EventTypeRecoveryRequired, payload, now, lease.TaskID); err != nil {
		return false, fmt.Errorf("append external execution expiry event: %w", err)
	}
	return true, nil
}

func (r *SQLRepository) readExternalExecutionLease(ctx context.Context, queryer interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, id string) (taskmodel.ExternalExecutionLease, bool, error) {
	var lease taskmodel.ExternalExecutionLease
	var executorType, state string
	var leaseTTLMS int64
	var cancelObservedAt, settledAt sql.NullTime
	var cancelRequestedAt sql.NullTime
	err := queryer.QueryRowContext(ctx, r.placeholder.rebind(`SELECT lease.id, lease.task_id, lease.stage_id, lease.attempt,
		lease.executor_type, lease.runtime_target_id, lease.provider_id, lease.capability, lease.receipt_protocol,
		lease.operation_id, lease.payload_sha256, lease.fence_token_hash, lease.state, lease.lease_ttl_ms,
		lease.lease_expires_at, lease.absolute_deadline_at, lease.cancel_observed_at, lease.settled_at,
		lease.created_at, lease.updated_at, task.cancel_requested_at
		FROM task_external_execution_leases lease JOIN tasks task ON task.id = lease.task_id WHERE lease.id = ?`), id).Scan(
		&lease.ID, &lease.TaskID, &lease.StageID, &lease.Attempt, &executorType, &lease.RuntimeTargetID, &lease.ProviderID,
		&lease.Capability, &lease.Protocol, &lease.OperationID, &lease.PayloadSHA256, &lease.FenceTokenHash, &state,
		&leaseTTLMS, &lease.LeaseExpiresAt, &lease.AbsoluteDeadlineAt, &cancelObservedAt, &settledAt,
		&lease.CreatedAt, &lease.UpdatedAt, &cancelRequestedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.ExternalExecutionLease{}, false, ErrNotFound
	}
	if err != nil {
		return taskmodel.ExternalExecutionLease{}, false, fmt.Errorf("read external execution lease: %w", err)
	}
	lease.ExecutorType = moduleapi.StageExecutorType(executorType)
	lease.State = moduleapi.ExternalExecutionLeaseState(state)
	lease.LeaseTTL = time.Duration(leaseTTLMS) * time.Millisecond
	lease.CancelObservedAt = nullableTime(cancelObservedAt)
	lease.SettledAt = nullableTime(settledAt)
	return lease, cancelRequestedAt.Valid, nil
}

func validExternalExecutionLease(lease taskmodel.ExternalExecutionLease) bool {
	return externalExecutionLeaseIdentityValid(lease) && externalExecutionLeaseTimingValid(lease)
}

func externalExecutionLeaseIdentityValid(lease taskmodel.ExternalExecutionLease) bool {
	return externalExecutionLeasePrimaryIdentityValid(lease) && externalExecutionLeaseBindingValid(lease) &&
		lowercaseReceiptSHA256(lease.PayloadSHA256) && lowercaseReceiptSHA256(lease.FenceTokenHash)
}

func externalExecutionLeasePrimaryIdentityValid(lease taskmodel.ExternalExecutionLease) bool {
	return strings.TrimSpace(lease.ID) != "" && lease.TaskID != 0 && lease.StageID != 0 && lease.Attempt > 0 &&
		strings.TrimSpace(string(lease.ExecutorType)) != "" && lease.RuntimeTargetID > 0
}

func externalExecutionLeaseBindingValid(lease taskmodel.ExternalExecutionLease) bool {
	return strings.TrimSpace(lease.ProviderID) != "" && strings.TrimSpace(lease.Capability) != "" &&
		strings.TrimSpace(lease.Protocol) != "" && strings.TrimSpace(lease.OperationID) != ""
}

func externalExecutionLeaseTimingValid(lease taskmodel.ExternalExecutionLease) bool {
	return lease.State == moduleapi.ExternalExecutionLeaseStateClaimed && lease.LeaseTTL > 0 && !lease.LeaseExpiresAt.IsZero() &&
		!lease.AbsoluteDeadlineAt.IsZero() && !lease.LeaseExpiresAt.After(lease.AbsoluteDeadlineAt)
}

func validExternalExecutionLogInput(input AppendExternalExecutionLogsInput) bool {
	return strings.TrimSpace(input.LeaseID) != "" && input.FenceTokenHash != "" && len(input.Entries) > 0
}

func validExternalExecutionSettlement(input SettleExternalExecutionInput) bool {
	if strings.TrimSpace(input.LeaseID) == "" || !lowercaseReceiptSHA256(input.FenceTokenHash) || !lowercaseReceiptSHA256(input.IntegritySHA256) {
		return false
	}
	switch input.Outcome {
	case moduleapi.ExternalReceiptOutcomeSuccess:
		return input.FailureCode == ""
	case moduleapi.ExternalReceiptOutcomeFailed, moduleapi.ExternalReceiptOutcomeNeedsAttention:
		return strings.TrimSpace(input.FailureCode) != "" && len(input.FailureCode) <= 128
	default:
		return false
	}
}

func externalExecutionStageStatus(outcome moduleapi.ExternalReceiptOutcome) moduleapi.StageStatus {
	switch outcome {
	case moduleapi.ExternalReceiptOutcomeSuccess:
		return moduleapi.StageStatusSuccess
	case moduleapi.ExternalReceiptOutcomeFailed:
		return moduleapi.StageStatusFailed
	default:
		return moduleapi.StageStatusUnknown
	}
}

func externalExecutionLeaseConflict(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return strings.Contains(err.Error(), "UNIQUE constraint failed")
}
