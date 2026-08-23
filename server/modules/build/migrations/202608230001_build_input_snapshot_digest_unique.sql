CREATE UNIQUE INDEX IF NOT EXISTS uq_build_workspace_snapshots_content_digest
  ON build_workspace_snapshots (content_digest);

COMMENT ON INDEX uq_build_workspace_snapshots_content_digest IS 'Build 输入快照内容摘要去重索引；快照身份保留为不可变历史证据';
