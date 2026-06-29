package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// CreateRun inserts a running job execution record.
func (r *SQLRunRepository) CreateRun(ctx context.Context, run TaskRun) (TaskRun, error) {
	if err := r.ensureAvailable(); err != nil {
		return TaskRun{}, err
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
		job_key,
		task_title,
		task_title_key,
		job_title,
		job_title_key,
		job_short_title,
		job_short_title_key,
		job_category,
		module_key,
		task_builtin,
		trigger_type,
		status,
		result_summary,
		result_json,
		error_message,
		started_at,
		finished_at,
		duration_ms,
		created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, '', '{}', '', $14, NULL, NULL, $15)
	RETURNING id`,
		run.TaskKey,
		run.JobKey,
		run.TaskTitle,
		run.TaskTitleKey,
		run.JobTitle,
		run.JobTitleKey,
		run.JobShortTitle,
		run.JobShortTitleKey,
		string(run.JobCategory),
		run.ModuleKey,
		run.TaskBuiltin,
		string(run.TriggerType),
		string(run.Status),
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

// FinishRun marks a running execution record as finished and returns the updated row.
func (r *SQLRunRepository) FinishRun(ctx context.Context, command RunFinishCommand) (TaskRun, error) {
	sqlID, err := r.sqlRunID(command.ID)
	if err != nil {
		return TaskRun{}, err
	}
	if err := validateRunFinish(command.Status, command.FinishedAt); err != nil {
		return TaskRun{}, err
	}
	durationMS, err := r.runDurationMS(ctx, sqlID, command.FinishedAt)
	if err != nil {
		return TaskRun{}, err
	}
	if err := r.updateFinishedRun(ctx, finishedRunUpdate{
		sqlID:         sqlID,
		status:        command.Status,
		finishedAt:    command.FinishedAt,
		durationMS:    durationMS,
		resultJSON:    command.ResultJSON,
		resultSummary: command.ResultSummary,
		errorMessage:  command.ErrorMessage,
	}); err != nil {
		return TaskRun{}, err
	}
	run, err := r.findRunBySQLID(ctx, sqlID)
	if err != nil {
		return TaskRun{}, fmt.Errorf("finish scheduler task run: %w", err)
	}
	return run, nil
}

type finishedRunUpdate struct {
	sqlID         int64
	status        RunStatus
	finishedAt    time.Time
	durationMS    int64
	resultJSON    string
	resultSummary string
	errorMessage  string
}

func (r *SQLRunRepository) updateFinishedRun(ctx context.Context, update finishedRunUpdate) error {
	_, err := r.db.ExecContext(ctx, `UPDATE scheduler_task_runs
	SET status = $1,
		result_summary = $2,
		result_json = $3,
		error_message = $4,
		finished_at = $5,
		duration_ms = $6
	WHERE id = $7`,
		string(update.status),
		update.resultSummary,
		defaultJSONObject(update.resultJSON),
		update.errorMessage,
		update.finishedAt.UTC(),
		update.durationMS,
		update.sqlID,
	)
	if err != nil {
		return fmt.Errorf("update scheduler task run: %w", err)
	}
	return nil
}

// ListRuns returns one page of run history for a scheduled task.
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, job_key, task_title, task_title_key, job_title, job_title_key, job_short_title, job_short_title_key, job_category, module_key, task_builtin, trigger_type, status, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
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

// LatestRunByTask returns the newest run for one scheduled task when present.
func (r *SQLRunRepository) LatestRunByTask(ctx context.Context, taskKey string) (TaskRun, bool, error) {
	if err := r.ensureAvailable(); err != nil {
		return TaskRun{}, false, err
	}
	if taskKey == "" {
		return TaskRun{}, false, errors.New("scheduler run task key is required")
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, task_title, task_title_key, job_title, job_title_key, job_short_title, job_short_title_key, job_category, module_key, task_builtin, trigger_type, status, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
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

// GetRun returns one run-history record by id.
func (r *SQLRunRepository) GetRun(ctx context.Context, id uint64) (TaskRun, error) {
	sqlID, err := r.sqlRunID(id)
	if err != nil {
		return TaskRun{}, err
	}
	run, err := r.findRunBySQLID(ctx, sqlID)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskRun{}, ErrTaskNotFound
	}
	return run, err
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
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, task_title, task_title_key, job_title, job_title_key, job_short_title, job_short_title_key, job_category, module_key, task_builtin, trigger_type, status, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
	FROM scheduler_task_runs
	WHERE id = $1`, sqlID)
	return scanTaskRun(row)
}
