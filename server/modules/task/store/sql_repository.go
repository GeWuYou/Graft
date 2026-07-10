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
	"graft/server/modules/task/state"
)

const (
	defaultPageLimit = 100
	maxPageLimit     = 500
)

// SQLRepository persists Task Runtime facts in module-owned PostgreSQL tables.
type SQLRepository struct {
	db          *sql.DB
	placeholder placeholderStyle
}

// NewSQLRepository creates a SQL repository with explicitly selected SQL dialect semantics.
func NewSQLRepository(db *sql.DB, dialect SQLDialect) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("task repository requires a non-nil sql db")
	}
	placeholder, err := placeholderStyleForDialect(dialect)
	if err != nil {
		return nil, err
	}
	return &SQLRepository{db: db, placeholder: placeholder}, nil
}

// Create atomically stores a frozen Task, its ordered Stage plan, and the created event.
func (r *SQLRepository) Create(ctx context.Context, input CreateInput) (taskmodel.Task, []taskmodel.Stage, error) {
	input, err := normalizeCreateInput(input)
	if err != nil {
		return taskmodel.Task{}, nil, err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return taskmodel.Task{}, nil, fmt.Errorf("begin create task transaction: %w", err)
	}
	defer rollback(tx)

	now := time.Now().UTC()
	input.Task.CreatedAt = now
	input.Task.UpdatedAt = now
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO tasks (
		task_type, owner_type, owner_id, status, input_json, metadata_json, plan_json, state_json,
		current_stage_key, created_by, scheduled_at, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		input.Task.Type,
		input.Task.Owner.Type,
		input.Task.Owner.ID,
		input.Task.Status,
		input.Task.Input,
		input.Task.Metadata,
		input.Task.Plan,
		input.Task.State,
		input.Task.CurrentStageKey,
		input.Task.CreatedBy,
		input.Task.ScheduledAt,
		now,
		now,
	).Scan(&input.Task.ID); err != nil {
		return taskmodel.Task{}, nil, fmt.Errorf("insert task: %w", err)
	}

	stages := make([]taskmodel.Stage, 0, len(input.Stages))
	for _, current := range input.Stages {
		current.TaskID = input.Task.ID
		current.CreatedAt = now
		current.UpdatedAt = now
		if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_stages (
			task_id, stage_key, sequence, executor_type, status, attempt, max_attempts, retry_backoff_ms,
			next_retry_at, input_json, recovery_policy, result_json, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING id`),
			current.TaskID,
			current.Key,
			current.Sequence,
			current.ExecutorType,
			current.Status,
			current.Attempt,
			current.MaxAttempts,
			current.RetryBackoffMS,
			current.NextRetryAt,
			current.Input,
			current.RecoveryPolicy,
			current.Result,
			now,
			now,
		).Scan(&current.ID); err != nil {
			return taskmodel.Task{}, nil, fmt.Errorf("insert task stage %q: %w", current.Key, err)
		}
		stages = append(stages, current)
	}

	createdPayload := json.RawMessage(`{}`)
	if err := tx.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?) RETURNING id`),
		input.Task.ID,
		int64(1),
		taskmodel.EventTypeCreated,
		createdPayload,
		now,
	).Scan(new(uint64)); err != nil {
		return taskmodel.Task{}, nil, fmt.Errorf("insert task created event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return taskmodel.Task{}, nil, fmt.Errorf("commit create task transaction: %w", err)
	}
	return input.Task, stages, nil
}

// Get returns one Task by stable ID.
func (r *SQLRepository) Get(ctx context.Context, taskID uint64) (taskmodel.Task, error) {
	if taskID == 0 {
		return taskmodel.Task{}, ErrInvalidInput
	}
	item, err := scanTask(r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+`
		FROM tasks WHERE id = ?`), taskID))
	if errors.Is(err, sql.ErrNoRows) {
		return taskmodel.Task{}, ErrNotFound
	}
	if err != nil {
		return taskmodel.Task{}, fmt.Errorf("get task: %w", err)
	}
	return item, nil
}

// List returns an owner-scoped Task history page and total using the owner index.
func (r *SQLRepository) List(ctx context.Context, owner moduleapi.TaskOwner, limit int, offset int) ([]taskmodel.Task, int64, error) {
	if strings.TrimSpace(owner.Type) == "" || strings.TrimSpace(owner.ID) == "" || offset < 0 {
		return nil, 0, ErrInvalidInput
	}
	var total int64
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`SELECT COUNT(*) FROM tasks WHERE owner_type = ? AND owner_id = ?`), owner.Type, owner.ID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count owner tasks: %w", err)
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+taskColumns()+`
		FROM tasks WHERE owner_type = ? AND owner_id = ?
		ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?`), owner.Type, owner.ID, normalizeLimit(limit), offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list owner tasks: %w", err)
	}
	defer closeRows(rows)
	items := make([]taskmodel.Task, 0)
	for rows.Next() {
		item, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, 0, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate owner tasks: %w", err)
	}
	return items, total, nil
}

// ListStages returns a Task's immutable serial stage plan in execution order.
func (r *SQLRepository) ListStages(ctx context.Context, taskID uint64) ([]taskmodel.Stage, error) {
	if taskID == 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+stageColumns()+`
		FROM task_stages WHERE task_id = ? ORDER BY sequence ASC, id ASC`), taskID)
	if err != nil {
		return nil, fmt.Errorf("list task stages: %w", err)
	}
	defer closeRows(rows)
	return scanStages(rows)
}

// ListEvents replays non-derivable Task events after a sequence cursor.
func (r *SQLRepository) ListEvents(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Event, error) {
	if taskID == 0 || afterSequence < 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+eventColumns()+`
		FROM task_events WHERE task_id = ? AND sequence > ? ORDER BY sequence ASC LIMIT ?`), taskID, afterSequence, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list task events: %w", err)
	}
	defer closeRows(rows)
	return scanEvents(rows)
}

// ListLogs replays Task logs after a sequence cursor.
func (r *SQLRepository) ListLogs(ctx context.Context, taskID uint64, afterSequence int64, limit int) ([]taskmodel.Log, error) {
	if taskID == 0 || afterSequence < 0 {
		return nil, ErrInvalidInput
	}
	rows, err := r.db.QueryContext(ctx, r.placeholder.rebind(`SELECT `+logColumns()+`
		FROM task_logs WHERE task_id = ? AND sequence > ? ORDER BY sequence ASC LIMIT ?`), taskID, afterSequence, normalizeLimit(limit))
	if err != nil {
		return nil, fmt.Errorf("list task logs: %w", err)
	}
	defer closeRows(rows)
	return scanLogs(rows)
}

// TransitionTask applies a validated compare-and-swap Task state transition.
func (r *SQLRepository) TransitionTask(ctx context.Context, input TaskTransitionInput) error {
	if input.TaskID == 0 {
		return ErrInvalidInput
	}
	if err := state.ValidateTaskTransition(input.From, input.To); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, current_stage_key = ?, failure_code = ?, failure_message = ?,
			started_at = COALESCE(?, started_at), finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ?`),
		input.To,
		input.CurrentStageKey,
		input.FailureCode,
		input.FailureMessage,
		input.StartedAt,
		input.FinishedAt,
		input.DurationMS,
		now,
		input.TaskID,
		input.From,
	)
	if err != nil {
		return fmt.Errorf("transition task: %w", err)
	}
	return expectOneAffected(result)
}

// TransitionStage applies a validated compare-and-swap Stage state transition.
func (r *SQLRepository) TransitionStage(ctx context.Context, input StageTransitionInput) error {
	if input.StageID == 0 || input.Attempt < 0 {
		return ErrInvalidInput
	}
	if err := state.ValidateStageTransition(input.From, input.To); err != nil {
		return err
	}
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, attempt = ?, next_retry_at = ?, result_json = ?, failure_code = ?, failure_message = ?,
			started_at = COALESCE(?, started_at), finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status = ?`),
		input.To,
		input.Attempt,
		input.NextRetryAt,
		normalizeJSON(input.Result),
		input.FailureCode,
		input.FailureMessage,
		input.StartedAt,
		input.FinishedAt,
		input.DurationMS,
		now,
		input.StageID,
		input.From,
	)
	if err != nil {
		return fmt.Errorf("transition task stage: %w", err)
	}
	return expectOneAffected(result)
}

// AppendEvent persists a non-derivable history fact before realtime is introduced.
func (r *SQLRepository) AppendEvent(ctx context.Context, input AppendEventInput) (taskmodel.Event, error) {
	if input.TaskID == 0 || input.Sequence <= 0 || !validEventType(input.Type) {
		return taskmodel.Event{}, ErrInvalidInput
	}
	now := time.Now().UTC()
	item := taskmodel.Event{TaskID: input.TaskID, Sequence: input.Sequence, Type: input.Type, Payload: normalizeJSON(input.Payload), CreatedAt: now}
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) VALUES (?, ?, ?, ?, ?) RETURNING id`),
		item.TaskID, item.Sequence, item.Type, item.Payload, item.CreatedAt,
	).Scan(&item.ID); err != nil {
		return taskmodel.Event{}, fmt.Errorf("append task event: %w", err)
	}
	return item, nil
}

// AppendLog persists one executor output line. Size limits are enforced by the worker batch.
func (r *SQLRepository) AppendLog(ctx context.Context, input AppendLogInput) (taskmodel.Log, error) {
	if input.TaskID == 0 || input.Sequence <= 0 || !validLogStream(input.Stream) || !validLogLevel(input.Level) || strings.TrimSpace(input.Line) == "" {
		return taskmodel.Log{}, ErrInvalidInput
	}
	if input.OccurredAt.IsZero() {
		input.OccurredAt = time.Now().UTC()
	}
	item := taskmodel.Log{TaskID: input.TaskID, StageID: input.StageID, Sequence: input.Sequence, Stream: input.Stream, Level: input.Level, Line: input.Line, OccurredAt: input.OccurredAt.UTC()}
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(`INSERT INTO task_logs (
		task_id, stage_id, sequence, stream, level, line, occurred_at
	) VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id`),
		item.TaskID, item.StageID, item.Sequence, item.Stream, item.Level, item.Line, item.OccurredAt,
	).Scan(&item.ID); err != nil {
		return taskmodel.Log{}, fmt.Errorf("append task log: %w", err)
	}
	return item, nil
}

// ClaimNextStage atomically selects the next serially executable Stage and
// persists its running state. PostgreSQL SKIP LOCKED keeps concurrent workers
// from running the same Stage; the SQLite path retains equivalent test semantics.
func (r *SQLRepository) ClaimNextStage(ctx context.Context, now time.Time) (StageClaim, bool, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	if r.placeholder == placeholderQuestion {
		return r.claimNextStageSQLite(ctx, now.UTC())
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("begin stage claim transaction: %w", err)
	}
	defer rollback(tx)

	claim, found, err := r.claimNextStagePostgres(ctx, tx, now.UTC())
	if err != nil || !found {
		return claim, found, err
	}
	if err := tx.Commit(); err != nil {
		return StageClaim{}, false, fmt.Errorf("commit stage claim transaction: %w", err)
	}
	return claim, true, nil
}

func (r *SQLRepository) claimNextStagePostgres(ctx context.Context, tx *sql.Tx, now time.Time) (StageClaim, bool, error) {
	row := tx.QueryRowContext(ctx, r.placeholder.rebind(`SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE stage.status = ?
			AND (stage.next_retry_at IS NULL OR stage.next_retry_at <= ?)
			AND task.status IN (?, ?, ?)
			AND (task.scheduled_at IS NULL OR task.scheduled_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = stage.task_id AND earlier.sequence < stage.sequence
					AND earlier.status NOT IN (?, ?, ?)
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		FOR UPDATE OF task, stage SKIP LOCKED
		LIMIT 1`),
		moduleapi.StageStatusPending,
		now,
		moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning,
		now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled,
	)
	return r.transitionClaimedStage(ctx, tx, row, now)
}

func (r *SQLRepository) claimNextStageSQLite(ctx context.Context, now time.Time) (StageClaim, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("begin stage claim transaction: %w", err)
	}
	defer rollback(tx)
	// #nosec G202 -- projection fragments are static module-owned SQL identifiers, never caller input.
	row := tx.QueryRowContext(ctx, `SELECT `+taskColumnsFor("task")+`, `+stageColumnsFor("stage")+`
		FROM task_stages stage
		JOIN tasks task ON task.id = stage.task_id
		WHERE stage.status = ?
			AND (stage.next_retry_at IS NULL OR stage.next_retry_at <= ?)
			AND task.status IN (?, ?, ?)
			AND (task.scheduled_at IS NULL OR task.scheduled_at <= ?)
			AND NOT EXISTS (
				SELECT 1 FROM task_stages earlier
				WHERE earlier.task_id = stage.task_id AND earlier.sequence < stage.sequence
					AND earlier.status NOT IN (?, ?, ?)
			)
		ORDER BY task.created_at ASC, stage.sequence ASC, stage.id ASC
		LIMIT 1`,
		moduleapi.StageStatusPending,
		now,
		moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning,
		now,
		moduleapi.StageStatusSuccess, moduleapi.StageStatusSkipped, moduleapi.StageStatusCancelled,
	)
	claim, found, err := r.transitionClaimedStage(ctx, tx, row, now)
	if err != nil || !found {
		return claim, found, err
	}
	if err := tx.Commit(); err != nil {
		return StageClaim{}, false, fmt.Errorf("commit stage claim transaction: %w", err)
	}
	return claim, true, nil
}

func (r *SQLRepository) transitionClaimedStage(ctx context.Context, tx *sql.Tx, row *sql.Row, now time.Time) (StageClaim, bool, error) {
	claim, err := scanStageClaim(row)
	if errors.Is(err, sql.ErrNoRows) {
		return StageClaim{}, false, nil
	}
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("select claimable task stage: %w", err)
	}
	if claim.Task.Status != moduleapi.TaskStatusRunning {
		if err := state.ValidateTaskTransition(claim.Task.Status, moduleapi.TaskStatusRunning); err != nil {
			return StageClaim{}, false, err
		}
		result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
			SET status = ?, current_stage_key = ?, started_at = COALESCE(started_at, ?), updated_at = ?
			WHERE id = ? AND status = ?`), moduleapi.TaskStatusRunning, claim.Stage.Key, now, now, claim.Task.ID, claim.Task.Status)
		if err != nil {
			return StageClaim{}, false, fmt.Errorf("mark claimed task running: %w", err)
		}
		if err := expectOneAffected(result); err != nil {
			return StageClaim{}, false, err
		}
		claim.Task.Status = moduleapi.TaskStatusRunning
		claim.Task.CurrentStageKey = stringPointer(claim.Stage.Key)
		claim.Task.StartedAt = &now
	}
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, attempt = ?, started_at = COALESCE(started_at, ?), updated_at = ?
		WHERE id = ? AND status = ?`), moduleapi.StageStatusRunning, claim.Stage.Attempt+1, now, now, claim.Stage.ID, moduleapi.StageStatusPending)
	if err != nil {
		return StageClaim{}, false, fmt.Errorf("mark claimed stage running: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return StageClaim{}, false, err
	}
	claim.Stage.Status = moduleapi.StageStatusRunning
	claim.Stage.Attempt++
	claim.Stage.StartedAt = &now
	return claim, true, nil
}

// RequestCancellation records a cooperative cancellation request. It leaves a
// running Stage intact so the worker can invoke its consumer-owned Cancel hook.
func (r *SQLRepository) RequestCancellation(ctx context.Context, taskID uint64, requestedAt time.Time) (taskmodel.Task, error) {
	if taskID == 0 {
		return taskmodel.Task{}, ErrInvalidInput
	}
	if requestedAt.IsZero() {
		requestedAt = time.Now().UTC()
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET cancel_requested_at = COALESCE(cancel_requested_at, ?), updated_at = ?
		WHERE id = ? AND status IN (?, ?, ?, ?)`),
		requestedAt.UTC(), requestedAt.UTC(), taskID,
		moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled, moduleapi.TaskStatusRunning, moduleapi.TaskStatusNeedsAttention,
	)
	if err != nil {
		return taskmodel.Task{}, fmt.Errorf("request task cancellation: %w", err)
	}
	if err := expectOneAffected(result); err != nil {
		return taskmodel.Task{}, err
	}
	return r.Get(ctx, taskID)
}

// CancelPendingTask finalizes an unclaimed task without invoking a Stage executor.
func (r *SQLRepository) CancelPendingTask(ctx context.Context, taskID uint64, finishedAt time.Time, durationMS *int64) error {
	if taskID == 0 || finishedAt.IsZero() {
		return ErrInvalidInput
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, finished_at = ?, duration_ms = ?, updated_at = ?
		WHERE id = ? AND status IN (?, ?)`), moduleapi.TaskStatusCancelled, finishedAt.UTC(), durationMS, finishedAt.UTC(), taskID, moduleapi.TaskStatusPending, moduleapi.TaskStatusScheduled)
	if err != nil {
		return fmt.Errorf("cancel pending task: %w", err)
	}
	return expectOneAffected(result)
}

// RetryStage returns an operator-approved failed or unknown Stage to pending.
// The runtime only accepts retries while the parent Task is needs_attention.
func (r *SQLRepository) RetryStage(ctx context.Context, taskID uint64, stageID uint64, retryAt time.Time) (taskmodel.Stage, error) {
	if taskID == 0 || stageID == 0 || retryAt.IsZero() {
		return taskmodel.Stage{}, ErrInvalidInput
	}
	if err := r.resumeRetryableTask(ctx, taskID); err != nil {
		return taskmodel.Stage{}, err
	}
	if err := r.markStagePendingForRetry(ctx, taskID, stageID, retryAt); err != nil {
		return taskmodel.Stage{}, err
	}
	return r.getStage(ctx, taskID, stageID)
}

func (r *SQLRepository) resumeRetryableTask(ctx context.Context, taskID uint64) error {
	task, err := r.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Status != moduleapi.TaskStatusNeedsAttention && task.Status != moduleapi.TaskStatusFailed {
		return ErrStateConflict
	}
	if err := r.TransitionTask(ctx, TaskTransitionInput{TaskID: taskID, From: task.Status, To: moduleapi.TaskStatusRunning}); err != nil {
		return fmt.Errorf("resume task for stage retry: %w", err)
	}
	return nil
}

func (r *SQLRepository) markStagePendingForRetry(ctx context.Context, taskID uint64, stageID uint64, retryAt time.Time) error {
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, failure_code = NULL, failure_message = NULL,
			finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND task_id = ? AND status IN (?, ?)`),
		moduleapi.StageStatusPending, retryAt.UTC(), retryAt.UTC(), stageID, taskID, moduleapi.StageStatusFailed, moduleapi.StageStatusUnknown,
	)
	if err != nil {
		return fmt.Errorf("retry task stage: %w", err)
	}
	return expectOneAffected(result)
}

func (r *SQLRepository) getStage(ctx context.Context, taskID uint64, stageID uint64) (taskmodel.Stage, error) {
	stages, err := r.ListStages(ctx, taskID)
	if err != nil {
		return taskmodel.Stage{}, err
	}
	for _, stage := range stages {
		if stage.ID == stageID {
			return stage, nil
		}
	}
	return taskmodel.Stage{}, ErrNotFound
}

// RescheduleStage converts a retryable failed attempt into the next pending
// attempt without rewriting the attempt counter or terminal history details.
func (r *SQLRepository) RescheduleStage(ctx context.Context, stageID uint64, retryAt time.Time) error {
	if stageID == 0 || retryAt.IsZero() {
		return ErrInvalidInput
	}
	result, err := r.db.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE id = ? AND status = ?`), moduleapi.StageStatusPending, retryAt.UTC(), retryAt.UTC(), stageID, moduleapi.StageStatusFailed)
	if err != nil {
		return fmt.Errorf("reschedule task stage: %w", err)
	}
	return expectOneAffected(result)
}

// NextEventSequence returns the next append-only sequence for one Task.
func (r *SQLRepository) NextEventSequence(ctx context.Context, taskID uint64) (int64, error) {
	return r.nextSequence(ctx, "task_events", taskID)
}

// NextLogSequence returns the next append-only log sequence for one Task.
func (r *SQLRepository) NextLogSequence(ctx context.Context, taskID uint64) (int64, error) {
	return r.nextSequence(ctx, "task_logs", taskID)
}

func (r *SQLRepository) nextSequence(ctx context.Context, table string, taskID uint64) (int64, error) {
	if taskID == 0 {
		return 0, ErrInvalidInput
	}
	var sequence int64
	query := `SELECT COALESCE(MAX(sequence), 0) + 1 FROM ` + table + ` WHERE task_id = ?`
	if err := r.db.QueryRowContext(ctx, r.placeholder.rebind(query), taskID).Scan(&sequence); err != nil {
		return 0, fmt.Errorf("next task sequence: %w", err)
	}
	return sequence, nil
}

// RecoverInterruptedStages marks manual-reconcile running Stages unknown because
// a restarted process cannot prove an external side effect. Explicitly
// idempotent Stages return to pending for a controlled retry attempt.
func (r *SQLRepository) RecoverInterruptedStages(ctx context.Context, now time.Time) (int, error) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("begin interrupted stage recovery: %w", err)
	}
	defer rollback(tx)
	unknownCount, err := r.markManualRecoveryUnknown(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	if err := r.markRecoveryTasks(ctx, tx, now); err != nil {
		return 0, err
	}
	if err := r.appendRecoveryEvents(ctx, tx, now); err != nil {
		return 0, err
	}
	retriedCount, err := r.rescheduleIdempotentRecovery(ctx, tx, now)
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit interrupted stage recovery: %w", err)
	}
	return unknownCount + retriedCount, nil
}

func (r *SQLRepository) markManualRecoveryUnknown(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	recoveryMessage := "stage outcome is unknown after task runtime restart"
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, failure_code = ?, failure_message = ?, finished_at = ?,
			duration_ms = NULL,
			updated_at = ?
		WHERE status = ? AND recovery_policy = ?`), moduleapi.StageStatusUnknown, "runner_interrupted", recoveryMessage, now.UTC(), now.UTC(), moduleapi.StageStatusRunning, moduleapi.StageRecoveryManualReconcile)
	if err != nil {
		return 0, fmt.Errorf("mark interrupted stages unknown: %w", err)
	}
	count64, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count interrupted stages: %w", err)
	}
	return int(count64), nil
}

func (r *SQLRepository) markRecoveryTasks(ctx context.Context, tx *sql.Tx, now time.Time) error {
	recoveryMessage := "stage outcome is unknown after task runtime restart"
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE tasks
		SET status = ?, failure_code = ?, failure_message = ?, finished_at = ?,
			duration_ms = NULL,
			updated_at = ?
		WHERE status = ? AND id IN (SELECT DISTINCT task_id FROM task_stages WHERE status = ? AND failure_code = ?)`),
		moduleapi.TaskStatusNeedsAttention, "runner_interrupted", recoveryMessage, now.UTC(), now.UTC(),
		moduleapi.TaskStatusRunning, moduleapi.StageStatusUnknown, "runner_interrupted",
	)
	if err != nil {
		return fmt.Errorf("mark interrupted tasks needs attention: %w", err)
	}
	if _, err := result.RowsAffected(); err != nil {
		return fmt.Errorf("count interrupted tasks: %w", err)
	}
	return nil
}

func (r *SQLRepository) appendRecoveryEvents(ctx context.Context, tx *sql.Tx, now time.Time) error {
	query := fmt.Sprintf(`INSERT INTO task_events (
		task_id, sequence, event_type, payload_json, created_at
	) SELECT DISTINCT stage.task_id,
		COALESCE((SELECT MAX(event.sequence) FROM task_events event WHERE event.task_id = stage.task_id), 0) + 1,
		?, %s, %s
	FROM task_stages stage
	JOIN tasks task ON task.id = stage.task_id
	WHERE task.status = ? AND stage.status = ? AND stage.failure_code = ?
		AND NOT EXISTS (
			SELECT 1 FROM task_events required
			WHERE required.task_id = stage.task_id AND required.event_type = ?
				AND NOT EXISTS (
					SELECT 1 FROM task_events resolved
					WHERE resolved.task_id = required.task_id
						AND resolved.event_type = ?
						AND resolved.sequence > required.sequence
				)
		)`, r.jsonValuePlaceholder(), r.timestampValuePlaceholder())
	if _, err := tx.ExecContext(ctx, r.placeholder.rebind(query),
		taskmodel.EventTypeRecoveryRequired, json.RawMessage(`{}`), now.UTC(),
		moduleapi.TaskStatusNeedsAttention, moduleapi.StageStatusUnknown, "runner_interrupted", taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved,
	); err != nil {
		return fmt.Errorf("append interrupted stage recovery events: %w", err)
	}
	return nil
}

func (r *SQLRepository) rescheduleIdempotentRecovery(ctx context.Context, tx *sql.Tx, now time.Time) (int, error) {
	result, err := tx.ExecContext(ctx, r.placeholder.rebind(`UPDATE task_stages
		SET status = ?, next_retry_at = ?, failure_code = NULL, failure_message = NULL,
			finished_at = NULL, duration_ms = NULL, updated_at = ?
		WHERE status = ? AND recovery_policy = ?`),
		moduleapi.StageStatusPending, now.UTC(), now.UTC(), moduleapi.StageStatusRunning, moduleapi.StageRecoveryRetryIfIdempotent,
	)
	if err != nil {
		return 0, fmt.Errorf("reschedule idempotent interrupted stages: %w", err)
	}
	retriedCount, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("count idempotent interrupted stages: %w", err)
	}
	return int(retriedCount), nil
}

// normalizeCreateInput 校验并规范化任务及其初始阶段配置。
// 返回规范化后的创建输入；输入无效时返回错误。
func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	if err := normalizeTaskForCreate(&input.Task, len(input.Stages)); err != nil {
		return CreateInput{}, err
	}
	if err := normalizeStagesForCreate(input.Stages); err != nil {
		return CreateInput{}, err
	}
	return input, nil
}

// normalizeTaskForCreate validates and normalizes a task before creation.
// It requires a task type, owner, and at least one stage, and permits pending or scheduled tasks.
// Scheduled tasks must include a scheduled time.
func normalizeTaskForCreate(task *taskmodel.Task, stageCount int) error {
	if task == nil || strings.TrimSpace(string(task.Type)) == "" || strings.TrimSpace(task.Owner.Type) == "" || strings.TrimSpace(task.Owner.ID) == "" || stageCount == 0 {
		return ErrInvalidInput
	}
	if task.Status != moduleapi.TaskStatusPending && task.Status != moduleapi.TaskStatusScheduled {
		return ErrInvalidInput
	}
	if task.Status == moduleapi.TaskStatusScheduled && task.ScheduledAt == nil {
		return ErrInvalidInput
	}
	task.Input = normalizeJSON(task.Input)
	task.Metadata = normalizeJSON(task.Metadata)
	task.Plan = normalizeJSON(task.Plan)
	task.State = normalizeJSON(task.State)
	return nil
}

// normalizeStagesForCreate validates and normalizes the stages in a task's initial execution plan.
func normalizeStagesForCreate(stages []taskmodel.Stage) error {
	seenKeys := make(map[string]struct{}, len(stages))
	for index := range stages {
		if err := normalizeStageForCreate(&stages[index], index+1, seenKeys); err != nil {
			return err
		}
	}
	return nil
}

// normalizeStageForCreate 验证并规范化创建任务所需的阶段数据，同时检查阶段序号和键的唯一性。
// 无效阶段或重复键返回 ErrInvalidInput；有效阶段的输入和结果 JSON 为空时规范化为 {}。
func normalizeStageForCreate(stage *taskmodel.Stage, sequence int, seenKeys map[string]struct{}) error {
	if !validInitialStage(stage, sequence) {
		return ErrInvalidInput
	}
	if _, exists := seenKeys[stage.Key]; exists {
		return ErrInvalidInput
	}
	seenKeys[stage.Key] = struct{}{}
	stage.Input = normalizeJSON(stage.Input)
	stage.Result = normalizeJSON(stage.Result)
	return nil
}

// validInitialStage reports whether a stage satisfies the constraints for an initial pending stage at the specified sequence.
func validInitialStage(stage *taskmodel.Stage, sequence int) bool {
	return stage != nil &&
		strings.TrimSpace(stage.Key) != "" &&
		strings.TrimSpace(string(stage.ExecutorType)) != "" &&
		stage.Sequence == sequence &&
		stage.Status == moduleapi.StageStatusPending &&
		stage.Attempt == 0 &&
		stage.MaxAttempts >= 1 &&
		stage.RetryBackoffMS >= 0 &&
		stage.RecoveryPolicy != ""
}

// normalizeJSON 将空 JSON 值规范化为空对象。
func normalizeJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

// normalizeLimit 将分页限制规范化到允许的范围内。
//
// 小于等于零的值使用默认分页限制，超过最大值的值使用最大分页限制。
func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}

// expectOneAffected verifies that a database operation affected exactly one row.
func expectOneAffected(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read affected rows: %w", err)
	}
	if affected != 1 {
		return ErrStateConflict
	}
	return nil
}

// rollback 回滚事务并忽略回滚错误。
func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

// closeRows 关闭数据库查询结果集并忽略关闭错误。
func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

// validEventType reports whether the event type is supported by the task repository.
func validEventType(value taskmodel.EventType) bool {
	switch value {
	case taskmodel.EventTypeCreated, taskmodel.EventTypeCancelRequested, taskmodel.EventTypeCancelled, taskmodel.EventTypeRetryRequested, taskmodel.EventTypeRetryScheduled, taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved:
		return true
	default:
		return false
	}
}

// validLogStream reports whether value identifies a supported log stream.
func validLogStream(value string) bool {
	return value == "stdout" || value == "stderr" || value == "system"
}

// validLogLevel reports whether value is a supported log level.
func validLogLevel(value string) bool {
	return value == "info" || value == "warn" || value == "error"
}
