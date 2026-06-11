// Copyright (c) 2025-2026 GeWuYou
// SPDX-License-Identifier: Apache-2.0

package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"
)

const (
	defaultRunListLimit  = 20
	defaultTaskListLimit = 20
	maxTaskListLimit     = 100
	maxSQLRunID          = 1<<63 - 1
	schedulerRunTypeJob  = "job"
)

// SQLJobDefinitionRepository stores scheduler job definitions in SQL.
type SQLJobDefinitionRepository struct {
	db *sql.DB
}

// NewSQLJobDefinitionRepository creates the SQL-backed job definition repository.
func NewSQLJobDefinitionRepository(db *sql.DB) (*SQLJobDefinitionRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler job definition repository requires a non-nil sql db")
	}
	return &SQLJobDefinitionRepository{db: db}, nil
}

// SyncJobDefinitions upserts module-registered job definitions into persistence.
func (r *SQLJobDefinitionRepository) SyncJobDefinitions(ctx context.Context, definitions []JobDefinition) error {
	if err := r.ensureAvailable(); err != nil {
		return err
	}
	for _, definition := range definitions {
		if definition.ConfigSchema == "" {
			definition.ConfigSchema = "{}"
		}
		if definition.DefaultConfig == "" {
			definition.DefaultConfig = "{}"
		}
		if definition.CreatedAt.IsZero() {
			definition.CreatedAt = time.Now().UTC()
		}
		if definition.UpdatedAt.IsZero() {
			definition.UpdatedAt = definition.CreatedAt
		}
		if err := validateJobDefinition(definition); err != nil {
			return err
		}
		_, err := r.db.ExecContext(ctx, `INSERT INTO scheduler_job_definitions (
			job_key,
			module_key,
			title_key,
			title,
			description_key,
			description,
			config_schema,
			default_config,
			default_cron,
			enabled,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, true, $10, $11)
		ON CONFLICT (job_key) DO UPDATE
		SET module_key = EXCLUDED.module_key,
			title_key = EXCLUDED.title_key,
			title = EXCLUDED.title,
			description_key = EXCLUDED.description_key,
			description = EXCLUDED.description,
			config_schema = EXCLUDED.config_schema,
			default_config = EXCLUDED.default_config,
			default_cron = EXCLUDED.default_cron,
			updated_at = EXCLUDED.updated_at
		WHERE scheduler_job_definitions.deleted_at IS NULL`,
			definition.JobKey,
			definition.ModuleKey,
			definition.TitleKey,
			definition.Title,
			definition.DescriptionKey,
			definition.Description,
			definition.ConfigSchema,
			definition.DefaultConfig,
			definition.DefaultCron,
			definition.CreatedAt.UTC(),
			definition.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("sync scheduler job definition %s: %w", definition.JobKey, err)
		}
	}
	return nil
}

// ListJobDefinitions returns active persisted job definitions.
func (r *SQLJobDefinitionRepository) ListJobDefinitions(ctx context.Context) ([]JobDefinition, error) {
	if err := r.ensureAvailable(); err != nil {
		return nil, err
	}
	rows, err := r.db.QueryContext(ctx, `SELECT id, job_key, module_key, title_key, title, description_key, description, config_schema, default_config, default_cron, enabled, created_at, updated_at, deleted_at
	FROM scheduler_job_definitions
	WHERE deleted_at IS NULL
	ORDER BY module_key ASC, title ASC, id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list scheduler job definitions: %w", err)
	}
	return collectRows(rows, scanJobDefinition, "iterate scheduler job definitions")
}

// GetJobDefinition returns one active persisted job definition by key.
func (r *SQLJobDefinitionRepository) GetJobDefinition(ctx context.Context, key string) (JobDefinition, error) {
	if err := r.ensureAvailable(); err != nil {
		return JobDefinition{}, err
	}
	if key == "" {
		return JobDefinition{}, errors.New("scheduler job key is required")
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, job_key, module_key, title_key, title, description_key, description, config_schema, default_config, default_cron, enabled, created_at, updated_at, deleted_at
	FROM scheduler_job_definitions
	WHERE job_key = $1 AND deleted_at IS NULL
	LIMIT 1`, key)
	item, err := scanJobDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return JobDefinition{}, ErrJobDefinitionNotFound
	}
	return item, err
}

func (r *SQLJobDefinitionRepository) ensureAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler job definition repository is unavailable")
	}
	return nil
}

// SQLTaskRepository stores scheduled task instances in SQL.
type SQLTaskRepository struct {
	db *sql.DB
}

// NewSQLTaskRepository creates the SQL-backed scheduled task repository.
func NewSQLTaskRepository(db *sql.DB) (*SQLTaskRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler task repository requires a non-nil sql db")
	}
	return &SQLTaskRepository{db: db}, nil
}

// SeedBuiltinTasks upserts builtin scheduled task instances declared by modules.
func (r *SQLTaskRepository) SeedBuiltinTasks(ctx context.Context, tasks []TaskDefinition) error {
	if err := r.ensureTaskAvailable(); err != nil {
		return err
	}
	for _, task := range tasks {
		task.Builtin = true
		if task.ConfigJSON == "" {
			task.ConfigJSON = "{}"
		}
		if task.ConfigSource == "" {
			task.ConfigSource = taskConfigSourceSystem
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
		_, err := r.db.ExecContext(ctx, `WITH existing AS (
			SELECT cron_expression, enabled
			FROM scheduled_tasks
			WHERE task_key = $1 AND builtin = true AND deleted_at IS NULL
		)
			INSERT INTO scheduled_tasks (
				task_key,
				job_key,
				module_key,
				title,
				description,
				cron_expression,
				enabled,
				builtin,
				task_type,
				config_json,
				config_source,
				created_at,
				updated_at
			) VALUES (
				$1,
				$2,
				$3,
				$4,
				$5,
				COALESCE((SELECT cron_expression FROM existing), $6),
				COALESCE((SELECT enabled FROM existing), $7),
				true,
				'job',
				$8,
				$9,
				$10,
				$11
			)
			ON CONFLICT (task_key) DO UPDATE
			SET job_key = EXCLUDED.job_key,
				module_key = EXCLUDED.module_key,
				title = EXCLUDED.title,
				description = EXCLUDED.description,
				builtin = true,
				config_json = EXCLUDED.config_json,
				config_source = EXCLUDED.config_source,
				updated_at = EXCLUDED.updated_at
			WHERE scheduled_tasks.builtin = true AND scheduled_tasks.deleted_at IS NULL`,
			task.TaskKey,
			task.JobKey,
			task.ModuleKey,
			task.Title,
			task.Description,
			task.CronExpression,
			task.Enabled,
			task.ConfigJSON,
			task.ConfigSource,
			task.CreatedAt.UTC(),
			task.UpdatedAt.UTC(),
		)
		if err != nil {
			return fmt.Errorf("seed builtin scheduled task %s: %w", task.TaskKey, err)
		}
	}
	return nil
}

// CreateTask persists a user-created scheduled task instance.
func (r *SQLTaskRepository) CreateTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	task.Builtin = false
	if task.ConfigJSON == "" {
		task.ConfigJSON = "{}"
	}
	if task.ConfigSource == "" {
		task.ConfigSource = taskConfigSourceUser
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
		job_key,
		module_key,
		title,
		description,
		cron_expression,
			enabled,
			builtin,
			task_type,
			config_json,
			config_source,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, false, 'job', $8, $9, $10, $11)
		RETURNING id, task_key, job_key, module_key, title, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
		task.TaskKey,
		task.JobKey,
		task.ModuleKey,
		task.Title,
		task.Description,
		task.CronExpression,
		task.Enabled,
		task.ConfigJSON,
		task.ConfigSource,
		task.CreatedAt.UTC(),
		task.UpdatedAt.UTC(),
	)
	return scanTaskDefinition(row)
}

// UpdateTask applies mutable field changes to a scheduled task.
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
			config_source = $6,
			updated_at = $7
		WHERE task_key = $8 AND deleted_at IS NULL
		RETURNING id, task_key, job_key, module_key, title, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
		next.Title,
		next.Description,
		next.CronExpression,
		next.Enabled,
		next.ConfigJSON,
		next.ConfigSource,
		next.UpdatedAt,
		key,
	)
	return scanTaskDefinition(row)
}

func validateTaskPatch(key string, existing TaskDefinition, patch TaskMutation) error {
	if patch.TaskKey != "" && patch.TaskKey != key {
		return ErrTaskImmutable
	}
	if patch.JobKey != "" && patch.JobKey != existing.JobKey {
		return ErrTaskImmutable
	}
	if existing.Builtin && (patch.Title != "" || patch.Description != "") {
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
		next.ConfigSource = taskConfigSourceUser
	}
	return next
}

// DeleteTask soft-deletes a user-created scheduled task.
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

// SetTaskEnabled updates the enabled state of a scheduled task.
func (r *SQLTaskRepository) SetTaskEnabled(ctx context.Context, key string, enabled bool) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	row := r.db.QueryRowContext(ctx, `UPDATE scheduled_tasks
	SET enabled = $1,
		updated_at = $2
	WHERE task_key = $3 AND deleted_at IS NULL
		RETURNING id, task_key, job_key, module_key, title, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
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

// ListTasks returns active persisted scheduled task instances.
func (r *SQLTaskRepository) ListTasks(ctx context.Context, query TaskListQuery) ([]TaskDefinition, int, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return nil, 0, err
	}
	total, err := r.countTasks(ctx)
	if err != nil {
		return nil, 0, err
	}
	statement := `SELECT id, task_key, job_key, module_key, title, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE deleted_at IS NULL
	ORDER BY builtin DESC, created_at ASC, id ASC`
	normalized := normalizeTaskListQuery(query)
	rows, err := r.db.QueryContext(ctx, statement+` LIMIT $1 OFFSET $2`, normalized.Limit, normalized.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list scheduled tasks: %w", err)
	}
	items, err := collectRows(rows, scanTaskDefinition, "iterate scheduled tasks")
	if err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func normalizeTaskListQuery(query TaskListQuery) TaskListQuery {
	if query.Limit <= 0 {
		query.Limit = defaultTaskListLimit
	} else if query.Limit > maxTaskListLimit {
		query.Limit = maxTaskListLimit
	}
	if query.Offset < 0 {
		query.Offset = 0
	}
	return query
}

func (r *SQLTaskRepository) countTasks(ctx context.Context) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*)
	FROM scheduled_tasks
	WHERE deleted_at IS NULL`)
	var total int
	if err := row.Scan(&total); err != nil {
		return 0, fmt.Errorf("count scheduled tasks: %w", err)
	}
	return total, nil
}

// GetTask returns one active persisted scheduled task by key.
func (r *SQLTaskRepository) GetTask(ctx context.Context, key string) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	if key == "" {
		return TaskDefinition{}, errors.New("scheduler task key is required")
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, module_key, title, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE task_key = $1 AND deleted_at IS NULL
	LIMIT 1`, key)
	task, err := scanTaskDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, err
}

func (r *SQLTaskRepository) ensureTaskAvailable() error {
	if r == nil || r.db == nil {
		return errors.New("scheduler task repository is unavailable")
	}
	return nil
}

// SQLRunRepository stores scheduler run history in SQL.
type SQLRunRepository struct {
	db *sql.DB
}

// NewSQLRunRepository creates the SQL-backed run-history repository.
func NewSQLRunRepository(db *sql.DB) (*SQLRunRepository, error) {
	if db == nil {
		return nil, errors.New("scheduler run repository requires a non-nil sql db")
	}
	return &SQLRunRepository{db: db}, nil
}

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
		task_name,
		task_name_key,
		owner,
		module,
		task_type,
		trigger_type,
		status,
		error,
		result_summary,
		result_json,
		error_message,
		started_at,
		finished_at,
		duration_ms,
		created_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, '', '{}', '', $11, NULL, NULL, $12)
	RETURNING id`,
		run.TaskKey,
		run.JobKey,
		run.TaskName,
		run.TaskNameKey,
		run.Owner,
		run.Module,
		schedulerRunTypeJob,
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
		error = $2,
		result_summary = $3,
		result_json = $4,
		error_message = $5,
		finished_at = $6,
		duration_ms = $7
	WHERE id = $8`,
		string(update.status),
		update.errorMessage,
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, task_key, job_key, task_name, task_name_key, owner, module, trigger_type, status, error, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
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
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, task_name, task_name_key, owner, module, trigger_type, status, error, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
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
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, task_name, task_name_key, owner, module, trigger_type, status, error, result_summary, result_json, error_message, started_at, finished_at, duration_ms, created_at
	FROM scheduler_task_runs
	WHERE id = $1`, sqlID)
	return scanTaskRun(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

type rowsScanner interface {
	rowScanner
	Close() error
	Err() error
	Next() bool
}

func collectRows[T any](rows rowsScanner, scan func(rowScanner) (T, error), iterateLabel string) ([]T, error) {
	defer func() {
		_ = rows.Close()
	}()

	items := make([]T, 0)
	for rows.Next() {
		item, scanErr := scan(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", iterateLabel, err)
	}
	return items, nil
}

func scanTaskRun(scanner rowScanner) (TaskRun, error) {
	var run TaskRun
	var id int64
	var triggerType string
	var status string
	var legacyError string
	var resultSummary string
	var resultJSON string
	var errorMessage string
	var finishedAt sql.NullTime
	var durationMS sql.NullInt64
	if err := scanner.Scan(
		&id,
		&run.TaskKey,
		&run.JobKey,
		&run.TaskName,
		&run.TaskNameKey,
		&run.Owner,
		&run.Module,
		&triggerType,
		&status,
		&legacyError,
		&resultSummary,
		&resultJSON,
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
	run.TriggerType = TriggerType(triggerType)
	run.Status = RunStatus(status)
	run.Result = resultSummary
	run.ResultJSON = defaultJSONObject(resultJSON)
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
		return 0, errors.New("scheduler id from database is invalid")
	}
	runID, err := strconv.ParseUint(strconv.FormatInt(id, 10), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("convert scheduler id from database: %w", err)
	}
	return runID, nil
}

func scanTaskDefinition(scanner rowScanner) (TaskDefinition, error) {
	var task TaskDefinition
	var id int64
	var deletedAt sql.NullTime
	if err := scanner.Scan(
		&id,
		&task.TaskKey,
		&task.JobKey,
		&task.ModuleKey,
		&task.Title,
		&task.Description,
		&task.CronExpression,
		&task.Enabled,
		&task.Builtin,
		&task.ConfigJSON,
		&task.ConfigSource,
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
	if task.ConfigSource == "" {
		if task.Builtin {
			task.ConfigSource = taskConfigSourceSystem
		} else {
			task.ConfigSource = taskConfigSourceUser
		}
	}
	if deletedAt.Valid {
		deleted := deletedAt.Time
		task.DeletedAt = &deleted
	}
	return task, nil
}

func scanJobDefinition(scanner rowScanner) (JobDefinition, error) {
	var definition JobDefinition
	var id int64
	var deletedAt sql.NullTime
	if err := scanner.Scan(
		&id,
		&definition.JobKey,
		&definition.ModuleKey,
		&definition.TitleKey,
		&definition.Title,
		&definition.DescriptionKey,
		&definition.Description,
		&definition.ConfigSchema,
		&definition.DefaultConfig,
		&definition.DefaultCron,
		&definition.Enabled,
		&definition.CreatedAt,
		&definition.UpdatedAt,
		&deletedAt,
	); err != nil {
		return JobDefinition{}, err
	}
	definitionID, err := taskRunIDFromSQL(id)
	if err != nil {
		return JobDefinition{}, err
	}
	definition.ID = definitionID
	if deletedAt.Valid {
		deleted := deletedAt.Time
		definition.DeletedAt = &deleted
	}
	return definition, nil
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
