CREATE TABLE backup_runner_handoffs (
  id BIGSERIAL PRIMARY KEY,
  operation_id VARCHAR(128) NOT NULL,
  task_id BIGINT NOT NULL,
  purpose VARCHAR(64) NOT NULL,
  retain_until TIMESTAMPTZ NOT NULL,
  created_by BIGINT NULL,
  artifact_root TEXT NOT NULL,
  config_snapshot_ref TEXT NOT NULL,
  database_dump_ref TEXT NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'PLANNED',
  backup_id BIGINT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  completed_at TIMESTAMPTZ NULL,
  CONSTRAINT uq_backup_runner_handoffs_operation UNIQUE (operation_id),
  CONSTRAINT uq_backup_runner_handoffs_task UNIQUE (task_id),
  CONSTRAINT backup_runner_handoffs_status_check CHECK (status IN ('PLANNED', 'COMPLETED')),
  CONSTRAINT backup_runner_handoffs_backup_id_fkey FOREIGN KEY (backup_id) REFERENCES backups (id) ON DELETE RESTRICT
);

CREATE INDEX idx_backup_runner_handoffs_backup_id ON backup_runner_handoffs (backup_id) WHERE backup_id IS NOT NULL;

COMMENT ON TABLE backup_runner_handoffs IS '一次性备份 runner 的冻结范围与结算证据表';
COMMENT ON COLUMN backup_runner_handoffs.id IS '备份 runner 交接主键';
COMMENT ON COLUMN backup_runner_handoffs.operation_id IS '更新操作绑定的一次性稳定标识';
COMMENT ON COLUMN backup_runner_handoffs.task_id IS 'Task Runtime 任务执行主键';
COMMENT ON COLUMN backup_runner_handoffs.purpose IS '创建备份的冻结用途标识';
COMMENT ON COLUMN backup_runner_handoffs.retain_until IS '创建备份后使用的冻结保留截止时间';
COMMENT ON COLUMN backup_runner_handoffs.created_by IS '发起备份交接的管理员用户标识';
COMMENT ON COLUMN backup_runner_handoffs.artifact_root IS 'runner 可写入且 server 可复核的受控工件根目录';
COMMENT ON COLUMN backup_runner_handoffs.config_snapshot_ref IS '冻结的配置快照文件引用，不保存文件正文';
COMMENT ON COLUMN backup_runner_handoffs.database_dump_ref IS '冻结的数据库导出文件引用，不保存文件正文';
COMMENT ON COLUMN backup_runner_handoffs.status IS '交接状态，取值为 PLANNED 或 COMPLETED';
COMMENT ON COLUMN backup_runner_handoffs.backup_id IS '完成交接后创建的备份事实主键';
COMMENT ON COLUMN backup_runner_handoffs.created_at IS '交接计划持久化时间';
COMMENT ON COLUMN backup_runner_handoffs.completed_at IS '交接完成并写入备份事实的时间';
