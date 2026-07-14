-- atlas:txmode none
-- Validate new constraints after their addition transaction commits, then build
-- replacement indexes without blocking writes to the live project registry.
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

COMMENT ON INDEX compose_projects_application_id_live IS '存活应用公开标识唯一索引';
COMMENT ON INDEX compose_projects_runtime_compose_name_live IS '同一运行目标内存活 Compose 运行时名称唯一索引';
COMMENT ON INDEX compose_projects_workspace_path_live IS '存活应用工作区路径唯一索引';
COMMENT ON INDEX compose_projects_workspace_key_live IS '受管应用工作区名称唯一索引';
