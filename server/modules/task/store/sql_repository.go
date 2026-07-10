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

// NewSQLRepository creates a Task Runtime SQL repository.
func NewSQLRepository(db *sql.DB) (*SQLRepository, error) {
	if db == nil {
		return nil, errors.New("task repository requires a non-nil sql db")
	}
	return &SQLRepository{db: db, placeholder: placeholderStyleFor(db)}, nil
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

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	if err := normalizeTaskForCreate(&input.Task, len(input.Stages)); err != nil {
		return CreateInput{}, err
	}
	if err := normalizeStagesForCreate(input.Stages); err != nil {
		return CreateInput{}, err
	}
	return input, nil
}

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

func normalizeStagesForCreate(stages []taskmodel.Stage) error {
	seenKeys := make(map[string]struct{}, len(stages))
	for index := range stages {
		if err := normalizeStageForCreate(&stages[index], index+1, seenKeys); err != nil {
			return err
		}
	}
	return nil
}

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

func normalizeJSON(value json.RawMessage) json.RawMessage {
	if len(value) == 0 {
		return json.RawMessage(`{}`)
	}
	return value
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return defaultPageLimit
	}
	if limit > maxPageLimit {
		return maxPageLimit
	}
	return limit
}

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

func rollback(tx *sql.Tx) {
	_ = tx.Rollback()
}

func closeRows(rows *sql.Rows) {
	_ = rows.Close()
}

func validEventType(value taskmodel.EventType) bool {
	switch value {
	case taskmodel.EventTypeCreated, taskmodel.EventTypeCancelRequested, taskmodel.EventTypeCancelled, taskmodel.EventTypeRetryRequested, taskmodel.EventTypeRetryScheduled, taskmodel.EventTypeRecoveryRequired, taskmodel.EventTypeRecoveryResolved:
		return true
	default:
		return false
	}
}

func validLogStream(value string) bool {
	return value == "stdout" || value == "stderr" || value == "system"
}

func validLogLevel(value string) bool {
	return value == "info" || value == "warn" || value == "error"
}
