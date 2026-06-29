package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

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
			SELECT cron_expression, enabled, config_json, config_source
			FROM scheduled_tasks
			WHERE task_key = $1 AND builtin = true AND deleted_at = 0
		)
			INSERT INTO scheduled_tasks (
				task_key,
				job_key,
				title_key,
				title,
				description_key,
				description,
				cron_expression,
				enabled,
				builtin,
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
				$6,
				COALESCE((SELECT cron_expression FROM existing), $7),
				COALESCE((SELECT enabled FROM existing), $8),
				true,
				COALESCE((SELECT config_json FROM existing), $9),
				COALESCE((SELECT config_source FROM existing), $10),
				$11,
				$12
			)
			ON CONFLICT (task_key) WHERE deleted_at = 0 DO UPDATE
			SET job_key = EXCLUDED.job_key,
				title_key = EXCLUDED.title_key,
				title = EXCLUDED.title,
				description_key = EXCLUDED.description_key,
				description = EXCLUDED.description,
				builtin = true,
				config_json = EXCLUDED.config_json,
				config_source = EXCLUDED.config_source,
				updated_at = EXCLUDED.updated_at
			WHERE scheduled_tasks.builtin = true AND scheduled_tasks.deleted_at = 0`,
			task.TaskKey,
			task.JobKey,
			task.TitleKey,
			task.Title,
			task.DescriptionKey,
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
		title_key,
		title,
		description_key,
		description,
		cron_expression,
		enabled,
		builtin,
		config_json,
		config_source,
		created_at,
		updated_at
	) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $10, $11, $12)
	RETURNING id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
		task.TaskKey,
		task.JobKey,
		task.TitleKey,
		task.Title,
		task.DescriptionKey,
		task.Description,
		task.CronExpression,
		task.Enabled,
		task.ConfigJSON,
		task.ConfigSource,
		task.CreatedAt.UTC(),
		task.UpdatedAt.UTC(),
	)
	taskDefinition, err := scanTaskDefinition(row)
	if err != nil {
		return TaskDefinition{}, mapScheduledTaskWriteError(err)
	}
	return taskDefinition, nil
}

// ReplaceTask restores one active scheduled task definition after a failed runtime refresh.
func (r *SQLTaskRepository) ReplaceTask(ctx context.Context, task TaskDefinition) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	if err := validateDefinition(task); err != nil {
		return TaskDefinition{}, err
	}
	row := r.db.QueryRowContext(ctx, `UPDATE scheduled_tasks
		SET job_key = $1,
			title_key = $2,
			title = $3,
			description_key = $4,
			description = $5,
			cron_expression = $6,
			enabled = $7,
			builtin = $8,
			config_json = $9,
			config_source = $10,
			created_at = $11,
			updated_at = $12
		WHERE task_key = $13 AND deleted_at = 0
		RETURNING id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
		task.JobKey,
		task.TitleKey,
		task.Title,
		task.DescriptionKey,
		task.Description,
		task.CronExpression,
		task.Enabled,
		task.Builtin,
		task.ConfigJSON,
		task.ConfigSource,
		task.CreatedAt.UTC(),
		task.UpdatedAt.UTC(),
		task.TaskKey,
	)
	taskDefinition, err := scanTaskDefinition(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskDefinition{}, ErrTaskNotFound
		}
		return TaskDefinition{}, mapScheduledTaskWriteError(err)
	}
	return taskDefinition, nil
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
		WHERE task_key = $8 AND deleted_at = 0
		RETURNING id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
		next.Title,
		next.Description,
		next.CronExpression,
		next.Enabled,
		next.ConfigJSON,
		next.ConfigSource,
		next.UpdatedAt,
		key,
	)
	taskDefinition, err := scanTaskDefinition(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return TaskDefinition{}, ErrTaskNotFound
		}
		return TaskDefinition{}, mapScheduledTaskWriteError(err)
	}
	return taskDefinition, nil
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
	deletedAt := time.Now().UTC()
	result, err := r.db.ExecContext(ctx, `UPDATE scheduled_tasks
	SET deleted_at = $1,
		updated_at = $2
	WHERE task_key = $3 AND deleted_at = 0`, deletedAt.Unix(), deletedAt, key)
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
	WHERE task_key = $3 AND deleted_at = 0
		RETURNING id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at`,
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
	statement := `SELECT id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE deleted_at = 0
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

func (r *SQLTaskRepository) countTasks(ctx context.Context) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COUNT(*)
	FROM scheduled_tasks
	WHERE deleted_at = 0`)
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
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE task_key = $1 AND deleted_at = 0
	LIMIT 1`, key)
	task, err := scanTaskDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, err
}

// GetTaskByTitle returns one active persisted scheduled task by display title.
func (r *SQLTaskRepository) GetTaskByTitle(ctx context.Context, title string) (TaskDefinition, error) {
	if err := r.ensureTaskAvailable(); err != nil {
		return TaskDefinition{}, err
	}
	normalized := strings.TrimSpace(title)
	if normalized == "" {
		return TaskDefinition{}, ErrTaskNotFound
	}
	row := r.db.QueryRowContext(ctx, `SELECT id, task_key, job_key, title_key, title, description_key, description, cron_expression, enabled, builtin, config_json, config_source, created_at, updated_at, deleted_at
	FROM scheduled_tasks
	WHERE title = $1 AND deleted_at = 0
	LIMIT 1`, normalized)
	task, err := scanTaskDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return TaskDefinition{}, ErrTaskNotFound
	}
	return task, err
}
