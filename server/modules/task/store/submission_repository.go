package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"graft/server/internal/moduleapi"
	taskmodel "graft/server/modules/task/model"
)

// CreateSubmission 保存独立于 Task 的提交事实，并在 owner 锁内检查现有可执行 Task。
//
//nolint:cyclop,gocognit,gocyclo,nestif // 事务内必须按幂等、owner 容量和唯一索引竞争顺序完成创建。
func (r *SQLRepository) CreateSubmission(ctx context.Context, input CreateSubmissionInput) (taskmodel.Submission, bool, error) {
	submission := input.Submission
	if err := validateSubmission(submission); err != nil {
		return taskmodel.Submission{}, false, err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return taskmodel.Submission{}, false, fmt.Errorf("begin create submission: %w", err)
	}
	defer rollback(tx)
	if err := r.lockOwner(ctx, tx, submission.Owner); err != nil {
		return taskmodel.Submission{}, false, err
	}
	if submission.IdempotencyKeyHash != nil {
		existing, found, err := r.findIdempotentSubmission(ctx, tx, submission)
		if err != nil || found {
			if err != nil {
				return taskmodel.Submission{}, false, err
			}
			if existing.SubmissionFingerprint == nil || submission.SubmissionFingerprint == nil || *existing.SubmissionFingerprint != *submission.SubmissionFingerprint {
				return taskmodel.Submission{}, false, moduleapi.ErrTaskSubmissionConflict
			}
			return existing, true, nil
		}
	}
	if busy, err := r.ownerHasActiveTask(ctx, tx, submission.Owner); err != nil {
		return taskmodel.Submission{}, false, err
	} else if busy {
		return taskmodel.Submission{}, false, moduleapi.ErrTaskOwnerBusy
	}
	now := time.Now().UTC()
	submission.CreatedAt, submission.UpdatedAt = now, now
	_, err = tx.ExecContext(ctx, r.placeholder.rebind(`INSERT INTO task_submissions (
		id, task_type, owner_type, owner_id, requested_by, idempotency_key_hash, submission_fingerprint,
		state, submission_version, lease_ttl_ms, lease_renewable, lease_token_hash, lease_expires_at, absolute_deadline_at, prerequisite_kind, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		submission.ID, submission.Type, submission.Owner.Type, submission.Owner.ID, submission.RequestedBy,
		submission.IdempotencyKeyHash, submission.SubmissionFingerprint, submission.State, submission.Version, submission.LeaseTTL.Milliseconds(), submission.LeaseRenewable,
		submission.LeaseTokenHash, submission.LeaseExpiresAt, submission.AbsoluteDeadlineAt, submission.PrerequisiteKind, now, now)
	if err != nil {
		if isReservedOwnerConflict(err) {
			return taskmodel.Submission{}, false, moduleapi.ErrTaskOwnerBusy
		}
		return taskmodel.Submission{}, false, fmt.Errorf("insert task submission: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return taskmodel.Submission{}, false, fmt.Errorf("commit create submission: %w", err)
	}
	return submission, false, nil
}

// GetSubmission 返回一条提交聚合，不泄漏租约令牌摘要。
func (r *SQLRepository) GetSubmission(ctx context.Context, submissionID string) (taskmodel.Submission, error) {
	if strings.TrimSpace(submissionID) == "" {
		return taskmodel.Submission{}, ErrInvalidInput
	}
	item, err := scanSubmission(r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+submissionColumns()+` FROM task_submissions WHERE id = ?`), submissionID))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Submission{}, ErrNotFound
	}
	if err != nil {
		return taskmodel.Submission{}, fmt.Errorf("get task submission: %w", err)
	}
	return item, nil
}

// RenewSubmission 在 reserved 状态下以 version 和令牌摘要续租。
func (r *SQLRepository) RenewSubmission(ctx context.Context, input RenewSubmissionInput) (taskmodel.Submission, error) {
	if input.ID == "" || input.LeaseTokenHash == "" || input.Version <= 0 || input.LeaseExpiresAt.IsZero() {
		return taskmodel.Submission{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_submissions
		SET lease_expires_at = ?, submission_version = submission_version + 1, updated_at = ?
		WHERE id = ? AND state = ? AND submission_version = ? AND lease_token_hash = ?
			AND lease_expires_at > ? AND absolute_deadline_at > ? AND ? <= absolute_deadline_at`),
		input.LeaseExpiresAt.UTC(), now, input.ID, moduleapi.TaskSubmissionStateReserved, input.Version, input.LeaseTokenHash, now, now, input.LeaseExpiresAt.UTC())
	if err != nil {
		return taskmodel.Submission{}, fmt.Errorf("renew task submission: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.Submission{}, err
	}
	return r.GetSubmission(ctx, input.ID)
}

// MaterializeSubmission 在同一事务中写入调用模块前置条件和 Task 事实。
//
//nolint:cyclop,gocyclo // Task、前置条件和 Submission 必须在同一事务中完成 CAS 物化。
func (r *SQLRepository) MaterializeSubmission(ctx context.Context, input MaterializeSubmissionInput, writer moduleapi.TaskSubmissionWriter) (taskmodel.Task, bool, error) {
	if writer == nil || input.ID == "" || input.LeaseTokenHash == "" || input.Version <= 0 {
		return taskmodel.Task{}, false, ErrInvalidInput
	}
	input.Task.ActivationRequired = false
	normalized, err := r.normalizeMaterializedInput(input.Task, input.Stages)
	if err != nil {
		return taskmodel.Task{}, false, err
	}
	input.Task, input.Stages = normalized.Task, normalized.Stages
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return taskmodel.Task{}, false, fmt.Errorf("begin materialize submission: %w", err)
	}
	defer rollback(tx)
	submission, err := r.getSubmissionForUpdate(ctx, tx, input.ID)
	if err != nil {
		return taskmodel.Task{}, false, err
	}
	if submission.State == moduleapi.TaskSubmissionStateActivated && submission.TaskID != nil {
		task, err := r.getTaskWith(ctx, tx, *submission.TaskID)
		return task, true, err
	}
	if submission.State != moduleapi.TaskSubmissionStateReserved || submission.Version != input.Version || submission.LeaseTokenHash != input.LeaseTokenHash || !submission.LeaseExpiresAt.After(time.Now().UTC()) {
		return taskmodel.Task{}, false, ErrStateConflict
	}
	if err := r.lockOwner(ctx, tx, submission.Owner); err != nil {
		return taskmodel.Task{}, false, err
	}
	if busy, err := r.ownerHasActiveTask(ctx, tx, submission.Owner); err != nil {
		return taskmodel.Task{}, false, err
	} else if busy {
		return taskmodel.Task{}, false, moduleapi.ErrTaskOwnerBusy
	}
	created, err := r.insertTaskWithTx(ctx, tx, input.Task, input.Stages, submission.ID)
	if err != nil {
		return taskmodel.Task{}, false, err
	}
	// 未提交的 Task 对 Worker 不可见；先分配 ID 让前置条件可在同一事务内持有稳定外键。
	submission.TaskID = &created.ID
	prerequisiteRef, err := writer.MaterializeTaskSubmission(ctx, tx, toAPISubmission(submission))
	if err != nil {
		return taskmodel.Task{}, false, fmt.Errorf("materialize submission prerequisite: %w", err)
	}
	if strings.TrimSpace(prerequisiteRef) == "" {
		return taskmodel.Task{}, false, ErrInvalidInput
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_submissions
		SET state = ?, submission_version = submission_version + 1, prerequisite_ref = ?, task_id = ?, activated_at = ?, updated_at = ?
		WHERE id = ? AND state = ? AND submission_version = ? AND lease_token_hash = ?`),
		moduleapi.TaskSubmissionStateActivated, prerequisiteRef, created.ID, now, now, input.ID, moduleapi.TaskSubmissionStateReserved, input.Version, input.LeaseTokenHash)
	if err != nil {
		return taskmodel.Task{}, false, fmt.Errorf("activate task submission: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.Task{}, false, err
	}
	if err := tx.Commit(); err != nil {
		return taskmodel.Task{}, false, fmt.Errorf("commit materialize submission: %w", err)
	}
	return created, false, nil
}

// DiscardSubmission 终结仍处于 reserved 的提交且不删除审计事实。
func (r *SQLRepository) DiscardSubmission(ctx context.Context, input TerminalizeSubmissionInput) error {
	if input.ID == "" || input.LeaseTokenHash == "" || input.Version <= 0 || strings.TrimSpace(input.Reason) == "" {
		return ErrInvalidInput
	}
	return r.terminalizeSubmission(ctx, input, moduleapi.TaskSubmissionStateDiscarded, false)
}

// ExpireSubmissions 使用数据库时间最终终结过期 reserved 行；每行都由版本 fencing。
func (r *SQLRepository) ExpireSubmissions(ctx context.Context, limit int) (int, error) {
	if limit <= 0 {
		limit = defaultPageLimit
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT id, submission_version FROM task_submissions
		WHERE state = ? AND lease_expires_at <= CURRENT_TIMESTAMP ORDER BY lease_expires_at ASC, id ASC LIMIT ?`), moduleapi.TaskSubmissionStateReserved, limit)
	if err != nil {
		return 0, fmt.Errorf("scan expired submissions: %w", err)
	}
	type expiredSubmission struct {
		id      string
		version int64
	}
	candidates := make([]expiredSubmission, 0, limit)
	for rows.Next() {
		var item expiredSubmission
		if err := rows.Scan(&item.id, &item.version); err != nil {
			closeRows(rows)
			return 0, err
		}
		candidates = append(candidates, item)
	}
	if err := rows.Err(); err != nil {
		closeRows(rows)
		return 0, err
	}
	closeRows(rows)
	count := 0
	for _, item := range candidates {
		if err := r.terminalizeSubmission(ctx, TerminalizeSubmissionInput{ID: item.id, Version: item.version, Reason: "lease_expired"}, moduleapi.TaskSubmissionStateExpired, true); err == nil {
			count++
		} else if !errors.Is(err, ErrStateConflict) {
			return count, err
		}
	}
	return count, nil
}

//nolint:cyclop // 丢弃和过期共享一条带行锁、owner 锁和 version fencing 的原子路径。
func (r *SQLRepository) terminalizeSubmission(ctx context.Context, input TerminalizeSubmissionInput, target moduleapi.TaskSubmissionState, expired bool) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin terminalize submission: %w", err)
	}
	defer rollback(tx)
	submission, err := r.getSubmissionForUpdate(ctx, tx, input.ID)
	if err != nil {
		return err
	}
	if submission.State == target {
		return nil
	}
	if submission.State != moduleapi.TaskSubmissionStateReserved || submission.Version != input.Version || (!expired && submission.LeaseTokenHash != input.LeaseTokenHash) {
		return ErrStateConflict
	}
	if err := r.lockOwner(ctx, tx, submission.Owner); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_submissions SET state = ?, submission_version = submission_version + 1, terminal_reason = ?, terminal_at = ?, updated_at = ? WHERE id = ? AND state = ? AND submission_version = ?`), target, input.Reason, now, now, input.ID, moduleapi.TaskSubmissionStateReserved, input.Version)
	if err != nil {
		return fmt.Errorf("terminalize task submission: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SQLRepository) lockOwner(ctx context.Context, tx *sql.Tx, owner moduleapi.TaskOwner) error {
	if r.placeholder == placeholderQuestion {
		return nil
	}
	_, err := tx.ExecContext(ctx, r.placeholder.rebind(`SELECT pg_advisory_xact_lock(hashtextextended(?, 0))`), owner.Type+"\x00"+owner.ID)
	if err != nil {
		return wrapDatabaseOperation("task_owner_capacity_lock", err)
	}
	return nil
}

func (r *SQLRepository) ownerHasActiveTask(ctx context.Context, tx *sql.Tx, owner moduleapi.TaskOwner) (bool, error) {
	var exists bool
	err := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT EXISTS(SELECT 1 FROM tasks WHERE owner_type = ? AND owner_id = ? AND status IN (?, ?, ?, ?, ?))`), owner.Type, owner.ID, moduleapi.TaskStatusPending, moduleapi.TaskStatusReady, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention).Scan(&exists)
	return exists, err
}

func (r *SQLRepository) getSubmissionForUpdate(ctx context.Context, tx *sql.Tx, id string) (taskmodel.Submission, error) {
	query := `SELECT ` + submissionColumns() + ` FROM task_submissions WHERE id = ?`
	if r.placeholder != placeholderQuestion {
		query += ` FOR UPDATE`
	}
	item, err := scanSubmission(tx.QueryRowContext(ctx, r.placeholder.rebind(query), id))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Submission{}, ErrNotFound
	}
	if err != nil {
		return taskmodel.Submission{}, fmt.Errorf("lock task submission: %w", err)
	}
	return item, nil
}

func (r *SQLRepository) getTaskWith(ctx context.Context, queryer taskQueryer, id uint64) (taskmodel.Task, error) {
	item, err := scanTask(queryer.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+` FROM tasks WHERE id = ?`), id))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Task{}, ErrNotFound
	}
	return item, err
}

//nolint:dupl // Task 与 Submission 的幂等查询故意保持同构，便于审计两种聚合的键语义。
func (r *SQLRepository) findIdempotentSubmission(ctx context.Context, queryer taskQueryer, candidate taskmodel.Submission) (taskmodel.Submission, bool, error) {
	requestedBy := uint64(0)
	if candidate.RequestedBy != nil {
		requestedBy = *candidate.RequestedBy
	}
	item, err := scanSubmission(queryer.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+submissionColumns()+` FROM task_submissions WHERE task_type = ? AND owner_type = ? AND owner_id = ? AND COALESCE(requested_by, 0) = ? AND idempotency_key_hash = ?`), candidate.Type, candidate.Owner.Type, candidate.Owner.ID, requestedBy, candidate.IdempotencyKeyHash))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Submission{}, false, nil
	}
	if err != nil {
		return taskmodel.Submission{}, false, fmt.Errorf("find idempotent task submission: %w", err)
	}
	return item, true, nil
}

func (r *SQLRepository) insertTaskWithTx(ctx context.Context, tx *sql.Tx, task taskmodel.Task, stages []taskmodel.Stage, submissionID string) (taskmodel.Task, error) {
	now := time.Now().UTC()
	err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO tasks (task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json, activation_required, current_stage_key, created_by, idempotency_key_hash, submission_fingerprint, scheduled_at, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`), task.Type, task.Owner.Type, task.Owner.ID, task.Status, task.Input, task.Metadata, task.Plan, task.State, false, task.CurrentStageKey, task.CreatedBy, task.IdempotencyKeyHash, task.SubmissionFingerprint, task.ScheduledAt, now, now).Scan(&task.ID)
	if err != nil {
		return taskmodel.Task{}, mapCreateTaskError(err)
	}
	for _, stage := range stages {
		stage.TaskID = task.ID
		if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_stages (task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms, next_retry_at, input_json, coordination_group, leg_id, recovery_policy, result_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`), stage.TaskID, stage.Key, stage.Sequence, stage.ExecutorType, stage.Status, stage.Attempt, stage.MaxAttempts, stage.RetryBackoffMS, stage.NextRetryAt, stage.Input, stage.CoordinationGroup, stage.LegID, stage.RecoveryPolicy, stage.Result, now, now).Scan(&stage.ID); err != nil {
			return taskmodel.Task{}, fmt.Errorf("insert materialized stage %q: %w", stage.Key, err)
		}
	}
	payload := []byte(`{}`)
	if submissionID != "" {
		var payloadErr error
		payload, payloadErr = json.Marshal(struct {
			SubmissionID string `json:"submission_id"`
		}{SubmissionID: submissionID})
		if payloadErr != nil {
			return taskmodel.Task{}, fmt.Errorf("marshal task submission event: %w", payloadErr)
		}
	}
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (task_id, sequence, event_type, payload_json, created_at) VALUES (?, ?, ?, ?, ?) RETURNING id`), task.ID, int64(1), taskmodel.EventTypeCreated, payload, now).Scan(new(uint64)); err != nil {
		return taskmodel.Task{}, fmt.Errorf("insert materialized task event: %w", err)
	}
	return task, nil
}

func (r *SQLRepository) normalizeMaterializedInput(task taskmodel.Task, stages []taskmodel.Stage) (CreateInput, error) {
	if task.Status != moduleapi.TaskStatusReady && task.Status != moduleapi.TaskStatusScheduled {
		return CreateInput{}, ErrInvalidInput
	}
	return normalizeCreateInput(CreateInput{Task: task, Stages: stages})
}

//nolint:cyclop // 这里集中校验提交聚合的所有数据库约束，避免部分有效对象进入事务。
func validateSubmission(s taskmodel.Submission) error {
	if strings.TrimSpace(s.ID) == "" || strings.TrimSpace(string(s.Type)) == "" || strings.TrimSpace(s.Owner.Type) == "" || strings.TrimSpace(s.Owner.ID) == "" || s.State != moduleapi.TaskSubmissionStateReserved || s.Version != 1 || s.LeaseTTL <= 0 || s.LeaseTokenHash == "" || s.LeaseExpiresAt.IsZero() || s.AbsoluteDeadlineAt.IsZero() || s.LeaseExpiresAt.After(s.AbsoluteDeadlineAt) || strings.TrimSpace(s.PrerequisiteKind) == "" {
		return ErrInvalidInput
	}
	return nil
}

func isReservedOwnerConflict(err error) bool {
	return strings.Contains(err.Error(), "uq_task_submissions_reserved_owner") || strings.Contains(err.Error(), "UNIQUE constraint failed: task_submissions.owner_type, task_submissions.owner_id")
}

func toAPISubmission(s taskmodel.Submission) moduleapi.TaskSubmission {
	return moduleapi.TaskSubmission{ID: s.ID, TaskType: s.Type, Owner: s.Owner, RequestedBy: s.RequestedBy, State: s.State, SubmissionVersion: s.Version, LeaseTTL: s.LeaseTTL, LeaseRenewable: s.LeaseRenewable, LeaseExpiresAt: s.LeaseExpiresAt, AbsoluteDeadlineAt: s.AbsoluteDeadlineAt, PrerequisiteKind: s.PrerequisiteKind, PrerequisiteRef: s.PrerequisiteRef, TaskID: s.TaskID, TerminalReason: s.TerminalReason, CreatedAt: s.CreatedAt, UpdatedAt: s.UpdatedAt, ActivatedAt: s.ActivatedAt, TerminalAt: s.TerminalAt}
}
