ALTER TABLE compose_project_files
  ADD COLUMN IF NOT EXISTS last_observed_content text NOT NULL DEFAULT '';

COMMENT ON COLUMN compose_project_files.last_observed_content IS '最近一次刷新时观测到的原始文件内容基线';

UPDATE compose_project_files
SET exists_on_last_refresh = false,
    last_observed_hash = '',
    last_observed_content = '';

DELETE FROM compose_project_snapshots;

UPDATE compose_projects
SET last_refresh_status = 'never',
    last_refresh_at = NULL,
    last_refresh_error_code = '',
    last_refresh_error_message = '',
    last_refresh_config_hash = '',
    last_observed_config_hash = '',
    last_drift_checked_at = NULL,
    drift_status = 'unknown',
    updated_at = NOW()
WHERE deleted_at = 0;
