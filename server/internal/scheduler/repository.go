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

// SQLTaskRepository persists scheduled task definitions in scheduled_tasks.
type SQLTaskRepository struct {
	db *sql.DB
}

// NewSQLTaskRepository builds a scheduler task-definition repository from the shared SQL pool.
func NewSQLTaskRepository(db *sql.DB) (*SQLTaskRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler task repository requires a non-nil sql db")
	}

	return &SQLTaskRepository{db: db}, nil
}

// SeedBuiltinTasks inserts system tasks from cronx.Registry without overwriting user-edited cron/enabled values.
func (r *SQLTaskRepository) SeedBuiltinTasks(ctx context.Context, tasks []TaskDefinition) error {
	if err := r.ensureTaskAvailable(); err != nil {
		return err
	}
	for _, task := range tasks {
		task.TaskType = cronx.TaskTypeSystem
		task.Builtin = true
		if task.ConfigJSON == "" {
			task.ConfigJSON = "{}"
		}
		if task.CreatedAt.IsZero() {
			task.CreatedAt = time.Now().UTC()
		}
		if task.UpdatedAt.IsZero() {
			task.UpdatedAt = task.CreatedAt
		}
		if err := validateDefinition(task); err != nil {
			return err
		}
		_, err := r.db.ExecContext(ctx, `INSERT INTO scheduled_tasks (
			task_key,
			task_type,
			title,
			description,
			cron_expression,
			enabled,
			builtin,
			config_json,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, true, $7, $8, $9)
		ON CONFLICT (task_key) DO UPDATE
		SET task_type = EXCLUDED.task_type,
			title = EXCLUDED.title,
			description = EXCLUDED.description,
			builtin = true,
			config_json = EXCLUDED.config_json,
			updated_at = EXCLUDED.updated_at
		WHERE scheduled_tasks.builtin = true`,
			task.TaskKey,
			string(task.TaskType),
			task.Title,
			task.Description,
			task.CronExpression,
			task.Enabled,
			task.ConfigJSON,
			task.CreatedAt.UTC(),
			task.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("seed builtin scheduled task %s: %w", task.TaskKey, err)
		}
	}

	return nil
}

// CreateTask inserts one user-owned HTTP task definition.
func (r *SQLTaskRepository) CreateTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	if task.TaskType == "" {
		task.TaskType = cronx.TaskTypeHTTP
	}
	task.Builtin = false
	if task.ConfigJSON == "" {
		task.ConfigJSON = "{}"
	}
	if task.CreatedAt.IsZero() {
		task.CreatedAt = time.Now().UTC()
	}
	if task.UpdatedAt.IsZero() {
		task.UpdatedAt = task.CreatedAt
	}
	if err := validateDefinition(task); err != nil {
		return TaskDefinition{}, err
	}

	row := r.db.QueryRowContext(ctx, `INSERT INTO scheduled_tasks (
		task_key,
		task_type,
		title,
		description,
		cron_expression,
		enabled,
		builtin,
		config_json,
		created_at,
		updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, false, $7, $8, $9)
	RETURNING id, task_key, task_type, title, description, cron_expression, enabled, builtin, config_json, created_at, updated_at, deleted_at`,
		task.TaskKey,
		string(task.TaskType),
		task.Title,
		task.Description,
		task.CronExpression,
		task.Enabled,
		task.ConfigJSON,
		task.CreatedAt.UTC(),
		task.UpdatedAt.UTC(),
	)

	return scanTaskDefinition(row)
}

// UpdateTask updates mutable fields for an existing scheduled task.
func (r *SQLTaskRepository) UpdateTask(ctx context.Context, key string, patch TaskMutation) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	existing, err := r.GetTask(ctx, key)
	if err != nil {
		return TaskDefinition{}, err
	}
	if err := validateTaskPatch(key, existing, patch); err != nil {
		return TaskDefinition{}, err
	}
	next := applyTaskPatch(existing, patch)
	next.UpdatedAt = time.Now().UTC()
	if err := validateDefinition(next); err != nil {
		return TaskDefinition{}, err
	}

	row := r.db.QueryRowContext(ctx, `UPDATE scheduled_tasks
	SET title = $1,
		description = $2,
		cron_expression = $3,
		enabled = $4,
		config_json = $5,
		updated_at = $6
	WHERE task_key = $7 AND deleted_at IS NULL
	RETURNING id, task_key, task_type, title, description, cron_expression, enabled, builtin, config_json, created_at, updated_at, deleted_at`,
		next.Title,
		next.Description,
		next.CronExpression,
		next.Enabled,
		next.ConfigJSON,
		next.UpdatedAt,
		key,
	)

	return scanTaskDefinition(row)
}

func validateTaskPatch(key string, existing TaskDefinition, patch TaskMutation) error {
	if patch.TaskKey != "" && patch.TaskKey != key {
		return ErrTaskImmutable
	}
	if err := validateTaskTypePatch(existing, patch); err != nil {
		return err
	}
	if err := validateBuiltinTaskPatch(existing, patch); err != nil {
		return err
	}
	return nil
}

func validateTaskTypePatch(existing TaskDefinition, patch TaskMutation) error {
	if patch.TaskType == "" {
		return nil
	}
	if existing.Builtin && patch.TaskType != cronx.TaskTypeSystem {
		return ErrTaskImmutable
	}
	if !existing.Builtin && patch.TaskType != cronx.TaskTypeHTTP {
		return ErrTaskImmutable
	}
	return nil
}

func validateBuiltinTaskPatch(existing TaskDefinition, patch TaskMutation) error {
	if !existing.Builtin {
		return nil
	}
	if patch.Title != "" || patch.Description != "" || patch.ConfigJSON != "" {
		return ErrTaskImmutable
	}
	return nil
}

func applyTaskPatch(existing TaskDefinition, patch TaskMutation) TaskDefinition {
	next := existing
	if patch.Title != "" {
		next.Title = patch.Title
	}
	if patch.Description != "" {
		next.Description = patch.Description
	}
	if patch.CronExpression != "" {
		next.CronExpression = patch.CronExpression
	}
	if patch.EnabledSet {
		next.Enabled = patch.Enabled
	}
	if patch.ConfigJSON != "" {
		next.ConfigJSON = patch.ConfigJSON
	}
	return next
}

// DeleteTask soft-deletes a user HTTP task definition.
func (r *SQLTaskRepository) DeleteTask(ctx context.Context, key string) error {
	if err := r.ensureTaskAvailable(); err != nil {
		return err
	}
	existing, err := r.GetTask(ctx, key)
	if err != nil {
		return err
	}
	if existing.Builtin {
		return ErrTaskImmutable
	}
	result, err := r.db.ExecContext(ctx, `UPDATE scheduled_tasks
	SET deleted_at = $1,
		updated_at = $1
	WHERE task_key = $2 AND deleted_at IS NULL`, time.Now().UTC(), key)
	if err != nil {
		return fmt.Errorf("delete scheduled task: %w", err)
	}
	return requireAffectedScheduledTask(result)
}

// SetTaskEnabled toggles one scheduled task without changing its cron expression.
func (r *SQLTaskRepository) SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	row := r.db.QueryRowContext(ctx, `UPDATE scheduled_tasks
	SET enabled = $1,
		updated_at = $2
	WHERE task_key = $3 AND deleted_at IS NULL
	RETURNING id, task_key, task_type, title, description, cron_expression, enabled, builtin, config_json, created_at, updated_at, deleted_at`,
		enabled,
		time.Now().UTC(),
		key,
	)
	task, err := scanTaskDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, err
}

// ListTasks returns non-deleted scheduled task definitions in stable creation order.
func (r *SQLTaskRepository) ListTasks(ctx context.Context) ([]TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, task_type, title, description, cron_expression, enabled, builtin, config_json, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE deleted_at IS NULL
	ORDER BY builtin DESC, created_at ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list scheduled tasks: %w", err)
	}
	defer func() {
		_ = rows.Close()
	}()

	tasks := make([]TaskDefinition, 0)
	for rows.Next() {
		task, scanErr := scanTaskDefinition(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		tasks = append(tasks, task)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scheduled tasks: %w", err)
	}
	return tasks, nil
}

// GetTask returns one non-deleted scheduled task definition by key.
func (r *SQLTaskRepository) GetTask(ctx context.Context, key string) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	if key == "" {
		return TaskDefinition{}, errors.New("scheduler task key is required")
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, task_type, title, description, cron_expression, enabled, builtin, config_json, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE task_key = $1 AND deleted_at IS NULL
	LIMIT 1`, key)
	task, err := scanTaskDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, err
}

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
		result_summary,
		error_message,
		started_at,
		finished_at,
		duration_ms,
		created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, '', '', $9, NULL, NULL, $10)
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
	resultSummary string,
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

	if err := r.updateFinishedRun(ctx, finishedRunUpdate{
		sqlID:         sqlID,
		status:        status,
		finishedAt:    finishedAt,
		durationMS:    durationMS,
		resultSummary: resultSummary,
		errorMessage:  errorMessage,
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
	resultSummary string
	errorMessage  string
}

func (r *SQLRunRepository) updateFinishedRun(ctx context.Context, update finishedRunUpdate) error {
	_, err := r.db.ExecContext(ctx, `UPDATE scheduler_task_runs
	SET status = $1,
		error = $2,
		result_summary = $3,
		error_message = $4,
		finished_at = $5,
		duration_ms = $6
	WHERE id = $7`,
		string(update.status),
		update.errorMessage,
		update.resultSummary,
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, result_summary, error_message, started_at, finished_at, duration_ms, created_at
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

	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, result_summary, error_message, started_at, finished_at, duration_ms, created_at
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

// GetRun returns one persisted scheduler task run by id.
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
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, task_name, owner, module, task_type, trigger_type, status, error, result_summary, error_message, started_at, finished_at, duration_ms, created_at
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
	var legacyError string
	var resultSummary string
	var errorMessage string
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
		&legacyError,
		&resultSummary,
		&errorMessage,
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
	run.Result = resultSummary
	if errorMessage != "" {
		run.Error = errorMessage
	} else {
		run.Error = legacyError
	}
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
		return cronx.TaskTypeSystem
	}
	return cronx.TaskType(value)
}

type taskDefinitionScanner interface {
	Scan(dest ...any) error
}

func scanTaskDefinition(scanner taskDefinitionScanner) (TaskDefinition, error) {
	var task TaskDefinition
	var id int64
	var taskType string
	var deletedAt sql.NullTime
	if err := scanner.Scan(
		&id,
		&task.TaskKey,
		&taskType,
		&task.Title,
		&task.Description,
		&task.CronExpression,
		&task.Enabled,
		&task.Builtin,
		&task.ConfigJSON,
		&task.CreatedAt,
		&task.UpdatedAt,
		&deletedAt,
	); err != nil {
		return TaskDefinition{}, err
	}
	taskID, err := taskRunIDFromSQL(id)
	if err != nil {
		return TaskDefinition{}, err
	}
	task.ID = taskID
	task.TaskType = cronx.TaskType(taskType)
	if deletedAt.Valid {
		deleted := deletedAt.Time
		task.DeletedAt = &deleted
	}
	return task, nil
}

func (r *SQLTaskRepository) ensureTaskAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	return nil
}

func requireAffectedScheduledTask(result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read scheduled task affected rows: %w", err)
	}
	if affected == 0 {
		return ErrTaskNotFound
	}
	return nil
}
