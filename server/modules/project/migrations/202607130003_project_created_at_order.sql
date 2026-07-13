-- atlas:txmode none
-- 项目列表默认按创建时间倒序分页；id 作为同一创建时间下的稳定次序。
CREATE INDEX CONCURRENTLY IF NOT EXISTS compose_projects_created_at_live
  ON compose_projects (created_at DESC, id DESC)
  WHERE deleted_at = 0;

COMMENT ON INDEX compose_projects_created_at_live IS '按创建时间倒序查询存活 Compose 项目并以项目主键稳定排序';
