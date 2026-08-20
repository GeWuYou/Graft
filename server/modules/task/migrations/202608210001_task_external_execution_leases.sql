CREATE TABLE task_external_execution_leases (
  id varchar(64) NOT NULL,
  task_id bigint NOT NULL,
  stage_id bigint NOT NULL,
  attempt integer NOT NULL,
  executor_type varchar(191) NOT NULL,
  runtime_target_id bigint NOT NULL,
  provider_id varchar(64) NOT NULL,
  capability varchar(64) NOT NULL,
  receipt_protocol varchar(128) NOT NULL,
  operation_id varchar(256) NOT NULL,
  payload_sha256 char(64) NOT NULL,
  fence_token_hash char(64) NOT NULL,
  state varchar(16) NOT NULL,
  lease_ttl_ms bigint NOT NULL,
  lease_expires_at timestamptz NOT NULL,
  absolute_deadline_at timestamptz NOT NULL,
  cancel_observed_at timestamptz NULL,
  settled_at timestamptz NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  PRIMARY KEY (id),
  CONSTRAINT task_external_execution_leases_task_id_fkey FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE RESTRICT,
  CONSTRAINT task_external_execution_leases_task_stage_fkey FOREIGN KEY (task_id, stage_id) REFERENCES task_stages (task_id, id) ON DELETE RESTRICT,
  CONSTRAINT uq_task_external_execution_leases_stage_attempt UNIQUE (task_id, stage_id, attempt),
  CONSTRAINT uq_task_external_execution_leases_task_operation_attempt UNIQUE (task_id, operation_id, attempt),
  CONSTRAINT task_external_execution_leases_attempt_positive CHECK (attempt > 0),
  CONSTRAINT task_external_execution_leases_executor_not_blank CHECK (btrim(executor_type) <> ''),
  CONSTRAINT task_external_execution_leases_target_positive CHECK (runtime_target_id > 0),
  CONSTRAINT task_external_execution_leases_provider_not_blank CHECK (btrim(provider_id) <> ''),
  CONSTRAINT task_external_execution_leases_capability_not_blank CHECK (btrim(capability) <> ''),
  CONSTRAINT task_external_execution_leases_protocol_not_blank CHECK (btrim(receipt_protocol) <> ''),
  CONSTRAINT task_external_execution_leases_operation_not_blank CHECK (btrim(operation_id) <> ''),
  CONSTRAINT task_external_execution_leases_payload_digest CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
  CONSTRAINT task_external_execution_leases_fence_digest CHECK (fence_token_hash ~ '^[0-9a-f]{64}$'),
  CONSTRAINT task_external_execution_leases_state_check CHECK (state IN ('claimed', 'settled', 'expired')),
  CONSTRAINT task_external_execution_leases_ttl_positive CHECK (lease_ttl_ms > 0),
  CONSTRAINT task_external_execution_leases_expiry_order CHECK (lease_expires_at <= absolute_deadline_at),
  CONSTRAINT task_external_execution_leases_settlement_check CHECK ((state = 'settled' AND settled_at IS NOT NULL) OR (state <> 'settled' AND settled_at IS NULL))
);

CREATE INDEX idx_task_external_execution_leases_expiry
  ON task_external_execution_leases (lease_expires_at ASC, id ASC)
  WHERE state = 'claimed';
CREATE INDEX idx_task_external_execution_leases_binding
  ON task_external_execution_leases (runtime_target_id, provider_id, capability, created_at ASC)
  WHERE state = 'claimed';

ALTER TABLE task_external_receipts
  ADD COLUMN lease_id varchar(64) NULL,
  ADD COLUMN attempt integer NOT NULL DEFAULT 1,
  ADD COLUMN settled_stage_status varchar(16) GENERATED ALWAYS AS (
    CASE outcome WHEN 'success' THEN 'success' WHEN 'failed' THEN 'failed' ELSE 'unknown' END
  ) STORED,
  ADD CONSTRAINT task_external_receipts_lease_id_fkey FOREIGN KEY (lease_id) REFERENCES task_external_execution_leases (id) ON DELETE RESTRICT,
  ADD CONSTRAINT task_external_receipts_attempt_positive CHECK (attempt > 0),
  ADD CONSTRAINT task_external_receipts_stage_status_check CHECK (settled_stage_status IN ('success', 'failed', 'unknown'));

ALTER TABLE task_stages
  ADD COLUMN external_execution boolean NOT NULL DEFAULT false;

ALTER TABLE task_external_receipts DROP CONSTRAINT uq_task_external_receipts_task_operation;
ALTER TABLE task_external_receipts
  ADD CONSTRAINT uq_task_external_receipts_task_operation_attempt UNIQUE (task_id, operation_id, attempt);

ALTER TABLE task_external_receipts DROP CONSTRAINT task_external_receipts_settled_status_check;
ALTER TABLE task_external_receipts
  ADD CONSTRAINT task_external_receipts_settled_status_check CHECK (settled_task_status IN ('running', 'success', 'failed', 'cancelled', 'needs_attention'));

CREATE UNIQUE INDEX uq_task_external_receipts_lease
  ON task_external_receipts (lease_id)
  WHERE lease_id IS NOT NULL;

COMMENT ON TABLE task_external_execution_leases IS '任务运行时拥有的外部阶段执行租约与围栏事实表';
COMMENT ON COLUMN task_stages.external_execution IS '阶段是否只能由运行时代理通过围栏租约领取';
COMMENT ON COLUMN task_external_execution_leases.id IS '外部执行租约的随机稳定标识';
COMMENT ON COLUMN task_external_execution_leases.task_id IS '租约所属任务执行主键';
COMMENT ON COLUMN task_external_execution_leases.stage_id IS '租约所属阶段主键';
COMMENT ON COLUMN task_external_execution_leases.attempt IS '租约绑定的阶段执行尝试次数';
COMMENT ON COLUMN task_external_execution_leases.executor_type IS '冻结计划声明的阶段执行器类型';
COMMENT ON COLUMN task_external_execution_leases.runtime_target_id IS '经认证代理绑定的运行目标标识';
COMMENT ON COLUMN task_external_execution_leases.provider_id IS '执行该阶段的提供者稳定标识';
COMMENT ON COLUMN task_external_execution_leases.capability IS '代理领取阶段所需的受控能力标识';
COMMENT ON COLUMN task_external_execution_leases.receipt_protocol IS '外部执行回执协议及版本标识';
COMMENT ON COLUMN task_external_execution_leases.operation_id IS '冻结任务计划绑定的外部操作标识';
COMMENT ON COLUMN task_external_execution_leases.payload_sha256 IS '冻结阶段载荷的规范化摘要';
COMMENT ON COLUMN task_external_execution_leases.fence_token_hash IS '仅用于围栏校验的租约令牌摘要';
COMMENT ON COLUMN task_external_execution_leases.state IS '租约状态，已领取、已结算或已过期';
COMMENT ON COLUMN task_external_execution_leases.lease_ttl_ms IS '单次租约续期时长，单位毫秒';
COMMENT ON COLUMN task_external_execution_leases.lease_expires_at IS '当前租约失效时间';
COMMENT ON COLUMN task_external_execution_leases.absolute_deadline_at IS '该阶段尝试不可续租超过的绝对截止时间';
COMMENT ON COLUMN task_external_execution_leases.cancel_observed_at IS '代理首次确认观察到取消请求的时间';
COMMENT ON COLUMN task_external_execution_leases.settled_at IS '完全匹配的回执完成租约结算的时间';
COMMENT ON COLUMN task_external_execution_leases.created_at IS '租约事实创建时间';
COMMENT ON COLUMN task_external_execution_leases.updated_at IS '租约事实最近更新时间';
COMMENT ON COLUMN task_external_receipts.lease_id IS '新外部执行回执绑定的租约标识，历史控制器回执为空';
COMMENT ON COLUMN task_external_receipts.attempt IS '回执绑定的阶段执行尝试次数';
COMMENT ON COLUMN task_external_receipts.settled_stage_status IS '由受限回执结果确定的阶段结算状态';
COMMENT ON COLUMN task_external_receipts.settled_task_status IS '回执写入后任务运行时持有的任务状态';
