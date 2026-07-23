ALTER TABLE task_events DROP CONSTRAINT task_events_type_check;

ALTER TABLE task_events ADD CONSTRAINT task_events_type_check CHECK (event_type IN (
  'created', 'cancel_requested', 'cancelled', 'retry_requested', 'retry_scheduled',
  'recovery_required', 'recovery_resolved', 'external_receipt_settled'
)) NOT VALID;

ALTER TABLE task_events VALIDATE CONSTRAINT task_events_type_check;

ALTER TABLE task_stages ADD CONSTRAINT task_stages_task_id_id_key UNIQUE (task_id, id);

CREATE TABLE task_external_receipts (
  id BIGSERIAL PRIMARY KEY,
  task_id BIGINT NOT NULL,
  stage_id BIGINT NOT NULL,
  executor_type TEXT NOT NULL,
  receipt_protocol TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  outcome TEXT NOT NULL,
  failure_code TEXT NULL,
  integrity_sha256 CHAR(64) NOT NULL,
  settled_task_status TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT task_external_receipts_task_id_fkey FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE RESTRICT,
  CONSTRAINT task_external_receipts_task_stage_fkey FOREIGN KEY (task_id, stage_id) REFERENCES task_stages (task_id, id) ON DELETE RESTRICT,
  CONSTRAINT uq_task_external_receipts_task_operation UNIQUE (task_id, operation_id),
  CONSTRAINT task_external_receipts_protocol_check CHECK (btrim(receipt_protocol) <> ''),
  CONSTRAINT task_external_receipts_operation_check CHECK (btrim(operation_id) <> ''),
  CONSTRAINT task_external_receipts_outcome_check CHECK (outcome IN ('success', 'failed', 'needs_attention')),
  CONSTRAINT task_external_receipts_failure_code_check CHECK ((outcome = 'success' AND failure_code IS NULL) OR (outcome IN ('failed', 'needs_attention') AND failure_code IS NOT NULL AND btrim(failure_code) <> '')),
  CONSTRAINT task_external_receipts_integrity_check CHECK (integrity_sha256 ~ '^[0-9a-f]{64}$'),
  CONSTRAINT task_external_receipts_settled_status_check CHECK (settled_task_status IN ('success', 'failed', 'needs_attention'))
);

CREATE INDEX idx_task_external_receipts_stage_id ON task_external_receipts (stage_id);

COMMENT ON TABLE task_external_receipts IS '任务运行时保存的外部执行结算回执事实表';
COMMENT ON COLUMN task_external_receipts.id IS '外部执行回执主键';
COMMENT ON COLUMN task_external_receipts.task_id IS '关联的任务执行主键';
COMMENT ON COLUMN task_external_receipts.stage_id IS '关联的最终阶段主键';
COMMENT ON COLUMN task_external_receipts.executor_type IS '预期外部阶段执行器类型';
COMMENT ON COLUMN task_external_receipts.receipt_protocol IS '回执协议与版本标识';
COMMENT ON COLUMN task_external_receipts.operation_id IS '冻结任务计划绑定的外部操作标识';
COMMENT ON COLUMN task_external_receipts.outcome IS '外部执行声明的受限结算结果';
COMMENT ON COLUMN task_external_receipts.failure_code IS '失败或人工处置结果的稳定错误码';
COMMENT ON COLUMN task_external_receipts.integrity_sha256 IS '回执规范化内容的十六进制摘要';
COMMENT ON COLUMN task_external_receipts.settled_task_status IS '任务运行时写入的最终任务状态';
COMMENT ON COLUMN task_external_receipts.created_at IS '回执事实首次结算时间';
