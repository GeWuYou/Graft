ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS runtime_target_id bigint NULL;

CREATE INDEX IF NOT EXISTS compose_projects_runtime_target_live
  ON compose_projects (runtime_target_id, updated_at DESC, id DESC)
  WHERE deleted_at = 0;

COMMENT ON COLUMN compose_projects.runtime_target_id IS '关联 Runtime Target 主键；历史 local Compose 记录在运行目标发现后由 project Boot 幂等回填，过渡期允许为空';
COMMENT ON INDEX compose_projects_runtime_target_live IS '按运行目标筛选存活 Compose 应用的索引';
