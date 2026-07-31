ALTER TABLE update_operations
ADD COLUMN updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW();

COMMENT ON COLUMN update_operations.updated_at IS '更新操作阶段或终态最近一次持久化时间';
