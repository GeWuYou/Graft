ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS workspace_annotations_json jsonb NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN compose_projects.workspace_annotations_json IS '项目工作台文件与目录注释 JSON，键为相对路径，值为用户维护的提示说明';
