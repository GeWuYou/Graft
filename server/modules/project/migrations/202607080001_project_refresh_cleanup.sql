DROP INDEX IF EXISTS compose_projects_refresh_status_updated;

ALTER TABLE compose_projects
  DROP CONSTRAINT IF EXISTS compose_projects_last_refresh_status_check,
  DROP COLUMN IF EXISTS last_refresh_status,
  DROP COLUMN IF EXISTS last_refresh_at,
  DROP COLUMN IF EXISTS last_refresh_error_code,
  DROP COLUMN IF EXISTS last_refresh_error_message,
  DROP COLUMN IF EXISTS last_refresh_config_hash;

ALTER TABLE compose_project_files
  DROP COLUMN IF EXISTS exists_on_last_refresh;
