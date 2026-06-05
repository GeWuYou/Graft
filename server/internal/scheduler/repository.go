package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"graft/server/internal/cronx"
)

const (
	defaultRunListLimit = 20
	maxSQLRunID         = 1<<63 - 1
)

// SQLRunRepository persists scheduler runtime history in scheduler_task_runs.
type SQLRunRepository struct {
	db *sql.DB
}

// NewSQLRunRepository builds a scheduler run-history repository from the shared SQL pool.
func NewSQLRunRepository(db *sql.DB) (*SQLRunRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler run repository requires a non-nil sql db")
	}

	return &SQLRunRepository{db: db}, nil
}

// CreateRun inserts a running scheduler_task_runs row.
func (r *SQLRunRepository) CreateRun(ctx context.Context, run TaskRun) (TaskRun, error) {
	if err := r.ensureAvailable(); err != nil {
		return TaskRun{}, errors.New("scheduler run repository is unavailable")
	}
	if run.TaskKey == "" {
		return TaskRun{}, errors.New("scheduler run task key is required")
	}
	if run.StartedAt.IsZero() {
		return TaskRun{}, errors.New("scheduler run started_at is required")
	}
	if run.CreatedAt.IsZero() {
		run.CreatedAt = run.StartedAt
	}
	if run.Status == "" {
		run.Status = RunStatusRunning
	}

	row := r.db.QueryRowContext(ctx, `INSERT INTO scheduler_task_runs (
		task_key,
		task_name,
		owner,
		module,
		task_type,
		trigger_type,
		status,
		error,
		started_at,
		finished_at,
		duration_ms,
		created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NULL, NULL, $10)
	RETURNING id`,
		run.TaskKey,
		run.TaskName,
		run.Owner,
		run.Module,
		string(run.TaskType),
		string(run.TriggerType),
		string(run.Status),
		run.Error,
		run.StartedAt.UTC(),
		run.CreatedAt.UTC(),
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return TaskRun{}, fmt.Errorf("create scheduler task run: %w", err)
	}
	runID, err := taskRunIDFromSQL(id)
	if err != nil {
		return TaskRun{}, fmt.Errorf("create scheduler task run: %w", err)
	}
	run.ID = runID

	return run, nil
}

// FinishRun closes a running scheduler_task_runs row.
func (r *SQLRunRepository) FinishRun(
	ctx context.Context,
	id uint64,
	status RunStatus,
	finishedAt time.Time,
	errorMessage string,
) (TaskRun, error) {
	sqlID, err := r.sqlRunID(id)
	if err != nil {
		return TaskRun{}, err
	}
	if err := validateRunFinish(status, finishedAt); err != nil {
		return TaskRun{}, err
	}

	durationMS, err := r.runDurationMS(ctx, sqlID, finishedAt)
	if err != nil {
		return TaskRun{}, err
	}

	if err := r.updateFinishedRun(ctx, sqlID, status, finishedAt, durationMS, errorMessage); err != nil {
		return TaskRun{}, err
	}

	run, err := r.findRunBySQLID(ctx, sqlID)
	if err != nil {
		return TaskRun{}, fmt.Errorf("finish scheduler task run: %w", err)
	}

	return run, nil
}

func (r *SQLRunRepository) updateFinishedRun(
	ctx context.Context,
	sqlID int64,
	status RunStatus,
	finishedAt time.Time,
	durationMS int64,
	errorMessage string,
) error {
	_, err := r.db.ExecContext(ctx, `UPDATE scheduler_task_runs
	SET status = $1,
		error = $2,
		finished_at = $3,
		duration_ms = $4
	WHERE id = $5`,
		string(status),
		errorMessage,
		finishedAt.UTC(),
		durationMS,
		sqlID,
	)
	if err != nil {
		return fmt.Errorf("update scheduler task run: %w", err)
	}

	return nil
}

// ListRuns returns one stable page of task run history.
func (r *SQLRunRepository) ListRuns(ctx context.Context, query RunListQuery) (RunListResult, error) {
	if err := r.ensureAvailable(); err != nil {
		return RunListResult{}, err
	}

	normalized, err := normalizeRunListQuery(query)
	if err != nil {
		return RunListResult{}, err
	}

	total, err := r.countRuns(ctx, normalized.TaskKey)
	if err != nil {
		return RunListResult{}, err
	}

	items, err := r.listRunItems(ctx, normalized)
	if err != nil {
		return RunListResult{}, err
	}

	return RunListResult{Items: items, Total: total}, nil
}

func (r *SQLRunRepository) listRunItems(ctx context.Context, query RunListQuery) ([]TaskRun, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, started_at, finished_at, duration_ms, created_at
	FROM scheduler_task_runs
	WHERE task_key = $1
	ORDER BY started_at DESC, id DESC
	LIMIT $2 OFFSET $3`, query.TaskKey, query.Limit, query.Offset)
	if err != nil {
		return nil, fmt.Errorf("list scheduler task runs: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	items := make([]TaskRun, 0, query.Limit)
	for rows.Next() {
		run, scanErr := scanTaskRun(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scheduler task runs: %w", err)
	}

	return items, nil
}

// LatestRunByTask returns the latest persisted run for one task key.
func (r *SQLRunRepository) LatestRunByTask(ctx context.Context, taskKey string) (TaskRun, bool, error) {
	if err := r.ensureAvailable(); err != nil {
		return TaskRun{}, false, err
	}
	if taskKey == "" {
		return TaskRun{}, false, errors.New("scheduler run task key is required")
	}

	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, started_at, finished_at, duration_ms, created_at
	FROM scheduler_task_runs
	WHERE task_key = $1
	ORDER BY started_at DESC, id DESC
	LIMIT 1`, taskKey)

	run, err := scanTaskRun(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskRun{}, false, nil
	}
	if err != nil {
		return TaskRun{}, false, err
	}

	return run, true, nil
}

func (r *SQLRunRepository) ensureAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler run repository is unavailable")
	}

	return nil
}

func (r *SQLRunRepository) sqlRunID(id uint64) (int64, error) {
	if err := r.ensureAvailable(); err != nil {
		return 0, err
	}
	if id == 0 {
		return 0, errors.New("scheduler run id is required")
	}
	if id > maxSQLRunID {
		return 0, errors.New("scheduler run id is too large")
	}

	sqlID, err := strconv.ParseInt(strconv.FormatUint(id, 10), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert scheduler run id: %w", err)
	}

	return sqlID, nil
}

func validateRunFinish(status RunStatus, finishedAt time.Time) error {
	if status == "" {
		return errors.New("scheduler run status is required")
	}
	if finishedAt.IsZero() {
		return errors.New("scheduler run finished_at is required")
	}

	return nil
}

func normalizeRunListQuery(query RunListQuery) (RunListQuery, error) {
	if query.TaskKey == "" {
		return RunListQuery{}, errors.New("scheduler run task key is required")
	}
	if query.Limit <= 0 {
		query.Limit = defaultRunListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}

	return query, nil
}

func (r *SQLRunRepository) countRuns(ctx context.Context, taskKey string) (int, error) {
	var total int
	if err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scheduler_task_runs WHERE task_key = $1`, taskKey).Scan(&total); err != nil {
		return 0, fmt.Errorf("count scheduler task runs: %w", err)
	}

	return total, nil
}

func (r *SQLRunRepository) runDurationMS(ctx context.Context, sqlID int64, finishedAt time.Time) (int64, error) {
	var startedAt time.Time
	if err := r.db.QueryRowContext(ctx, `SELECT started_at FROM scheduler_task_runs WHERE id = $1`, sqlID).Scan(&startedAt); err != nil {
		return 0, fmt.Errorf("read scheduler task run start: %w", err)
	}

	durationMS := finishedAt.UTC().Sub(startedAt.UTC()).Milliseconds()
	if durationMS < 0 {
		return 0, nil
	}

	return durationMS, nil
}

func (r *SQLRunRepository) findRunBySQLID(ctx context.Context, sqlID int64) (TaskRun, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, started_at, finished_at, duration_ms, created_at
	FROM scheduler_task_runs
	WHERE id = $1`, sqlID)

	return scanTaskRun(row)
}

type taskRunScanner interface {
	Scan(dest ...any) error
}

func scanTaskRun(scanner taskRunScanner) (TaskRun, error) {
	var run TaskRun
	var id int64
	var taskType string
	var triggerType string
	var status string
	var finishedAt sql.NullTime
	var durationMS sql.NullInt64

	if err := scanner.Scan(
		&id,
		&run.TaskKey,
		&run.TaskName,
		&run.Owner,
		&run.Module,
		&taskType,
		&triggerType,
		&status,
		&run.Error,
		&run.StartedAt,
		&finishedAt,
		&durationMS,
		&run.CreatedAt,
	); err != nil {
		return TaskRun{}, err
	}

	runID, err := taskRunIDFromSQL(id)
	if err != nil {
		return TaskRun{}, err
	}
	run.ID = runID
	run.TaskType = cronTaskType(taskType)
	run.TriggerType = TriggerType(triggerType)
	run.Status = RunStatus(status)
	if finishedAt.Valid {
		finished := finishedAt.Time
		run.FinishedAt = &finished
	}
	if durationMS.Valid {
		duration := durationMS.Int64
		run.DurationMS = &duration
	}

	return run, nil
}

func taskRunIDFromSQL(id int64) (uint64, error) {
	if id <= 0 {
		return 0, errors.New("scheduler run id from database is invalid")
	}

	runID, err := strconv.ParseUint(strconv.FormatInt(id, 10), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert scheduler run id from database: %w", err)
	}

	return runID, nil
}

func cronTaskType(value string) cronx.TaskType {
	if value == "" {
		return cronx.TaskTypeCron
	}
	return cronx.TaskType(value)
}
