-- 为不可逆 App Log 删除命令保存持久化幂等回执，避免重试再次删除记录。
CREATE TABLE app_log_deletion_receipts (
  idempotency_key text PRIMARY KEY,
  deleted_ids jsonb NOT NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP
);

COMMENT ON TABLE app_log_deletion_receipts IS 'App Log 不可逆删除命令的持久化幂等回执。';
COMMENT ON COLUMN app_log_deletion_receipts.idempotency_key IS '调用方提供的稳定幂等键。';
COMMENT ON COLUMN app_log_deletion_receipts.deleted_ids IS '首次执行锁定的规范化 App Log ID 集合。';
COMMENT ON COLUMN app_log_deletion_receipts.created_at IS '回执首次写入时间。';
