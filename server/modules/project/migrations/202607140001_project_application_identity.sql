-- Public application identity is intentionally independent from the internal bigint key
-- used by project-owned child tables and foreign keys.
ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS application_id character varying,
  ADD COLUMN IF NOT EXISTS application_name character varying,
  ADD COLUMN IF NOT EXISTS workspace_path text,
  ADD COLUMN IF NOT EXISTS compose_project_name character varying,
  ADD COLUMN IF NOT EXISTS compose_project_name_source character varying;

-- Existing development rows retain their workspace and Compose identity while receiving a
-- deterministic, URL-safe public identifier. New writes use an application-generated ULID.
UPDATE compose_projects
SET application_id = 'app_' || lpad(upper(to_hex(id)), 26, '0'),
    workspace_path = working_directory,
    application_name = CASE
      WHEN source_kind <> 'imported' THEN NULLIF(
        regexp_replace(lower(regexp_replace(canonical_project_name, '[^a-z0-9]+', '-', 'g')), '(^-|-$)', '', 'g'),
        ''
      )
    END,
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
  ADD CONSTRAINT compose_projects_application_name_format_check
    CHECK (application_name IS NULL OR application_name ~ '^[a-z0-9][a-z0-9-]*$') NOT VALID,
  ADD CONSTRAINT compose_projects_compose_project_name_source_check
    CHECK (compose_project_name_source IN ('declared', 'generated', 'derived')) NOT VALID;

COMMENT ON COLUMN compose_projects.application_id IS '应用公开标识，格式为 app_ 加 ULID，供 HTTP 路径和外部引用使用';
COMMENT ON COLUMN compose_projects.application_name IS '受管应用的唯一安全名称，同时用于工作目录和默认 Compose 项目名称；外部导入应用为空';
COMMENT ON COLUMN compose_projects.workspace_path IS '应用工作区绝对路径，是应用文件的稳定载体';
COMMENT ON COLUMN compose_projects.compose_project_name IS 'Compose 顶层 name 对应的运行时项目名称';
COMMENT ON COLUMN compose_projects.compose_project_name_source IS 'Compose 名称来源，取值为 declared、generated、derived';
