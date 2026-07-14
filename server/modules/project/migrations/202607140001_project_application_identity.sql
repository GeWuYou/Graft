-- Public application identity is intentionally independent from the internal bigint key
-- used by project-owned child tables and foreign keys.
ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS application_id character varying,
  ADD COLUMN IF NOT EXISTS workspace_key character varying,
  ADD COLUMN IF NOT EXISTS workspace_path text,
  ADD COLUMN IF NOT EXISTS compose_project_name character varying,
  ADD COLUMN IF NOT EXISTS compose_project_name_source character varying;

-- Existing development rows retain their workspace and Compose identity while receiving a
-- deterministic, URL-safe public identifier. New writes use an application-generated ULID.
UPDATE compose_projects
SET application_id = 'app_' || lpad(upper(to_hex(id)), 26, '0'),
    workspace_path = working_directory,
    workspace_key = regexp_replace(lower(regexp_replace(canonical_project_name, '[^a-z0-9]+', '-', 'g')), '(^-|-$)', '', 'g'),
    compose_project_name = canonical_project_name,
    compose_project_name_source = CASE canonical_project_name_source
      WHEN 'override' THEN 'declared'
      ELSE 'derived'
    END
WHERE application_id IS NULL;

ALTER TABLE compose_projects
  ALTER COLUMN application_id SET NOT NULL,
  ALTER COLUMN workspace_path SET NOT NULL,
  ALTER COLUMN compose_project_name SET NOT NULL,
  ALTER COLUMN compose_project_name_source SET NOT NULL;

ALTER TABLE compose_projects
  ADD CONSTRAINT compose_projects_application_id_format_check
    CHECK (application_id ~ '^app_[0-9A-HJKMNP-TV-Z]{26}$') NOT VALID,
  ADD CONSTRAINT compose_projects_workspace_key_format_check
    CHECK (workspace_key IS NULL OR workspace_key ~ '^[a-z0-9][a-z0-9-]*$') NOT VALID,
  ADD CONSTRAINT compose_projects_compose_project_name_source_check
    CHECK (compose_project_name_source IN ('declared', 'generated', 'derived')) NOT VALID;

ALTER TABLE compose_projects
  VALIDATE CONSTRAINT compose_projects_application_id_format_check;
ALTER TABLE compose_projects
  VALIDATE CONSTRAINT compose_projects_workspace_key_format_check;
ALTER TABLE compose_projects
  VALIDATE CONSTRAINT compose_projects_compose_project_name_source_check;

DROP INDEX CONCURRENTLY IF EXISTS compose_projects_host_scope_canonical_project_name_live;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS compose_projects_application_id_live
  ON compose_projects (application_id)
  WHERE deleted_at = 0;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS compose_projects_runtime_compose_name_live
  ON compose_projects (runtime_target_id, compose_project_name)
  WHERE deleted_at = 0;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS compose_projects_workspace_path_live
  ON compose_projects (workspace_path)
  WHERE deleted_at = 0;
CREATE UNIQUE INDEX CONCURRENTLY IF NOT EXISTS compose_projects_workspace_key_live
  ON compose_projects (workspace_key)
  WHERE deleted_at = 0 AND workspace_key IS NOT NULL;

COMMENT ON COLUMN compose_projects.application_id IS '应用公开标识，格式为 app_ 加 ULID，供 HTTP 路径和外部引用使用';
COMMENT ON COLUMN compose_projects.workspace_key IS '受管应用工作区单层安全名称；外部导入工作区为空';
COMMENT ON COLUMN compose_projects.workspace_path IS '应用工作区绝对路径，是应用文件的稳定载体';
COMMENT ON COLUMN compose_projects.compose_project_name IS 'Compose 顶层 name 对应的运行时项目名称';
COMMENT ON COLUMN compose_projects.compose_project_name_source IS 'Compose 名称来源，取值为 declared、generated、derived';
COMMENT ON INDEX compose_projects_application_id_live IS '存活应用公开标识唯一索引';
COMMENT ON INDEX compose_projects_runtime_compose_name_live IS '同一运行目标内存活 Compose 运行时名称唯一索引';
COMMENT ON INDEX compose_projects_workspace_path_live IS '存活应用工作区路径唯一索引';
COMMENT ON INDEX compose_projects_workspace_key_live IS '受管应用工作区名称唯一索引';
