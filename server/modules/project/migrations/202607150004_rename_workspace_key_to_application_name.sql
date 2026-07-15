-- Upgrade local databases that applied the pre-release application identity migration
-- before the public field was renamed from workspace_key to application_name.
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'compose_projects' AND column_name = 'workspace_key'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = 'public' AND table_name = 'compose_projects' AND column_name = 'application_name'
  ) THEN
    ALTER TABLE compose_projects RENAME COLUMN workspace_key TO application_name;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'compose_projects'::regclass AND conname = 'compose_projects_workspace_key_format_check'
  ) AND NOT EXISTS (
    SELECT 1
    FROM pg_constraint
    WHERE conrelid = 'compose_projects'::regclass AND conname = 'compose_projects_application_name_format_check'
  ) THEN
    ALTER TABLE compose_projects RENAME CONSTRAINT compose_projects_workspace_key_format_check TO compose_projects_application_name_format_check;
  END IF;

  IF to_regclass('public.compose_projects_workspace_key_live') IS NOT NULL
    AND to_regclass('public.compose_projects_application_name_live') IS NULL THEN
    ALTER INDEX compose_projects_workspace_key_live RENAME TO compose_projects_application_name_live;
  END IF;
END $$;

COMMENT ON COLUMN compose_projects.application_name IS '受管应用的唯一安全名称，同时用于工作目录和默认 Compose 项目名称；外部导入应用为空';
COMMENT ON INDEX compose_projects_application_name_live IS '受管应用名唯一索引';
