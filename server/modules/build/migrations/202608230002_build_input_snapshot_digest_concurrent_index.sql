-- atlas:txmode none
CREATE UNIQUE INDEX CONCURRENTLY uq_build_workspace_snapshots_content_digest_rebuild
  ON build_workspace_snapshots (content_digest);
DROP INDEX CONCURRENTLY IF EXISTS uq_build_workspace_snapshots_content_digest;
ALTER INDEX uq_build_workspace_snapshots_content_digest_rebuild
  RENAME TO uq_build_workspace_snapshots_content_digest;
COMMENT ON INDEX uq_build_workspace_snapshots_content_digest IS 'Build 输入快照内容摘要去重索引；快照身份保留为不可变历史证据';
