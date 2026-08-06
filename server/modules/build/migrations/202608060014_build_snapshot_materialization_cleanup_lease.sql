ALTER TABLE build_workspace_snapshots
  ADD COLUMN materialization_claimed_at TIMESTAMPTZ NULL;

ALTER TABLE build_workspace_snapshots
  DROP CONSTRAINT build_workspace_snapshots_state_check,
  ADD CONSTRAINT build_workspace_snapshots_state_check CHECK (materialization_state IN ('available', 'expired', 'purging', 'purged'));

CREATE INDEX idx_build_workspace_snapshots_cleanup_claim
  ON build_workspace_snapshots (retention_expires_at ASC, id ASC)
  WHERE materialization_owner = 'build' AND materialization_state IN ('available', 'expired', 'purging') AND retention_expires_at IS NOT NULL;

COMMENT ON COLUMN build_workspace_snapshots.materialization_claimed_at IS '过期物化清理租约的领取时间，用于恢复中断清理';
