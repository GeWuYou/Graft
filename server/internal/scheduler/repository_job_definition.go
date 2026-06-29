package scheduler

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

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
			category,
			title_key,
			title,
			short_title_key,
			short_title,
			description_key,
			description,
			config_schema,
			default_config,
			default_cron,
			default_enabled,
			enabled,
			created_at,
			updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (job_key) WHERE deleted_at = 0 DO UPDATE
		SET module_key = EXCLUDED.module_key,
			category = EXCLUDED.category,
			title_key = EXCLUDED.title_key,
			title = EXCLUDED.title,
			short_title_key = EXCLUDED.short_title_key,
			short_title = EXCLUDED.short_title,
			description_key = EXCLUDED.description_key,
			description = EXCLUDED.description,
			config_schema = EXCLUDED.config_schema,
			default_config = EXCLUDED.default_config,
			default_cron = EXCLUDED.default_cron,
			default_enabled = EXCLUDED.default_enabled,
			enabled = EXCLUDED.enabled,
			updated_at = EXCLUDED.updated_at
		WHERE scheduler_job_definitions.deleted_at = 0`,
			definition.JobKey,
			definition.ModuleKey,
			string(definition.Category),
			definition.TitleKey,
			definition.Title,
			definition.ShortTitleKey,
			definition.ShortTitle,
			definition.DescriptionKey,
			definition.Description,
			definition.ConfigSchema,
			definition.DefaultConfig,
			definition.DefaultCron,
			definition.DefaultEnabled,
			definition.Enabled,
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
	rows, err := r.db.QueryContext(ctx, `SELECT id, job_key, module_key, category, title_key, title, short_title_key, short_title, description_key, description, config_schema, default_config, default_cron, default_enabled, enabled, created_at, updated_at, deleted_at
	FROM scheduler_job_definitions
	WHERE deleted_at = 0
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
	row := r.db.QueryRowContext(ctx, `SELECT id, job_key, module_key, category, title_key, title, short_title_key, short_title, description_key, description, config_schema, default_config, default_cron, default_enabled, enabled, created_at, updated_at, deleted_at
	FROM scheduler_job_definitions
	WHERE job_key = $1 AND deleted_at = 0
	LIMIT 1`, key)
	item, err := scanJobDefinition(row)
	if errors.Is(err, sql.ErrNoRows) {
		return JobDefinition{}, ErrJobDefinitionNotFound
	}
	return item, err
}
