ALTER TABLE build_workspace_snapshots
  DROP CONSTRAINT uq_build_workspace_snapshots_digest,
  ADD COLUMN materialization_owner VARCHAR(64) NOT NULL DEFAULT 'build',
  ADD COLUMN materialization_state VARCHAR(32) NOT NULL DEFAULT 'available',
  ADD COLUMN retention_policy VARCHAR(64) NOT NULL DEFAULT 'task_lifetime',
  ADD COLUMN retention_expires_at TIMESTAMPTZ NULL;

ALTER TABLE build_workspace_snapshots
  ADD CONSTRAINT build_workspace_snapshots_state_check CHECK (materialization_state IN ('available', 'expired', 'purged'));

COMMENT ON COLUMN build_workspace_snapshots.materialization_owner IS '物化内容的生命周期责任域';
COMMENT ON COLUMN build_workspace_snapshots.materialization_state IS '物化内容当前可用状态';
COMMENT ON COLUMN build_workspace_snapshots.retention_policy IS '快照物化内容的保留策略';
COMMENT ON COLUMN build_workspace_snapshots.retention_expires_at IS '物化内容允许保留到的时间；快照身份不随物化清理消失';
