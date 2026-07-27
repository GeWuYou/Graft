CREATE TABLE update_failure_diagnostics (
  id BIGSERIAL PRIMARY KEY,
  request_id VARCHAR(128) NOT NULL,
  operation_id VARCHAR(128) NULL,
  task_id BIGINT NULL,
  requested_by BIGINT NOT NULL,
  target_version VARCHAR(128) NOT NULL,
  failure_code VARCHAR(128) NOT NULL,
  failure_stage VARCHAR(64) NOT NULL,
  summary TEXT NOT NULL,
  detail TEXT NOT NULL,
  occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT uq_update_failure_diagnostics_request UNIQUE (request_id)
);

COMMENT ON TABLE update_failure_diagnostics IS '平台更新启动失败的受控诊断记录表';
COMMENT ON COLUMN update_failure_diagnostics.id IS '更新失败诊断记录主键';
COMMENT ON COLUMN update_failure_diagnostics.request_id IS '关联 HTTP 请求的稳定标识';
COMMENT ON COLUMN update_failure_diagnostics.operation_id IS '已生成时关联的一次性更新操作标识';
COMMENT ON COLUMN update_failure_diagnostics.task_id IS '已生成时关联的任务运行时主键';
COMMENT ON COLUMN update_failure_diagnostics.requested_by IS '发起更新确认的用户标识';
COMMENT ON COLUMN update_failure_diagnostics.target_version IS '本次请求的目标发行版本';
COMMENT ON COLUMN update_failure_diagnostics.failure_code IS '面向调用方的稳定更新失败码';
COMMENT ON COLUMN update_failure_diagnostics.failure_stage IS '服务端判定的更新启动失败阶段';
COMMENT ON COLUMN update_failure_diagnostics.summary IS '用于快速识别失败语义的受控摘要';
COMMENT ON COLUMN update_failure_diagnostics.detail IS '已过滤敏感信息的完整错误链诊断文本';
COMMENT ON COLUMN update_failure_diagnostics.occurred_at IS '更新启动失败发生时间';
