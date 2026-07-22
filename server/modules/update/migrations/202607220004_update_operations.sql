CREATE TABLE update_operations (
  id BIGSERIAL PRIMARY KEY,
  operation_id VARCHAR(128) NOT NULL,
  source_version VARCHAR(128) NOT NULL,
  target_version VARCHAR(128) NOT NULL,
  task_id BIGINT NOT NULL,
  backup_id BIGINT NULL,
  requested_by BIGINT NULL,
  status VARCHAR(32) NOT NULL,
  receipt_integrity_sha256 CHAR(64) NULL,
  failure_code VARCHAR(128) NULL,
  recovery_completed BOOLEAN NOT NULL DEFAULT FALSE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  started_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  finished_at TIMESTAMPTZ NULL,
  CONSTRAINT uq_update_operations_operation UNIQUE (operation_id),
  CONSTRAINT uq_update_operations_task UNIQUE (task_id),
  CONSTRAINT update_operations_status_check CHECK (status IN ('PLANNING', 'BACKING_UP', 'INSTALLING', 'SUCCESS', 'FAILED', 'RECOVERED', 'NEEDS_ATTENTION'))
);

CREATE INDEX idx_update_operations_created_at ON update_operations (created_at DESC);
CREATE INDEX idx_update_operations_backup_id ON update_operations (backup_id) WHERE backup_id IS NOT NULL;

COMMENT ON TABLE update_operations IS '平台 Compose 更新编排事实与历史查询表';
COMMENT ON COLUMN update_operations.id IS '更新操作主键';
COMMENT ON COLUMN update_operations.operation_id IS '一次性 runner 操作稳定标识';
COMMENT ON COLUMN update_operations.source_version IS '更新前 Graft 版本';
COMMENT ON COLUMN update_operations.target_version IS '目标 release 版本';
COMMENT ON COLUMN update_operations.task_id IS '关联的 Task Runtime 主键';
COMMENT ON COLUMN update_operations.backup_id IS 'runner 结算后的 Backup 主键';
COMMENT ON COLUMN update_operations.requested_by IS '人工确认操作的请求用户标识';
COMMENT ON COLUMN update_operations.status IS '更新操作当前阶段或终态';
COMMENT ON COLUMN update_operations.receipt_integrity_sha256 IS '无秘密 runner receipt 的 SHA-256 摘要';
COMMENT ON COLUMN update_operations.failure_code IS '受限 runner 失败分类';
COMMENT ON COLUMN update_operations.recovery_completed IS '迁移前配置和镜像恢复证据标志';
COMMENT ON COLUMN update_operations.created_at IS '操作创建时间';
COMMENT ON COLUMN update_operations.started_at IS 'runner 启动准备时间';
COMMENT ON COLUMN update_operations.finished_at IS 'receipt 结算完成时间';
