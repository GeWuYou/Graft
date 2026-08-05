-- atlas:txmode none

CREATE TABLE task_submissions (
  id varchar(64) NOT NULL,
  task_type varchar(191) NOT NULL,
  owner_type varchar(64) NOT NULL,
  owner_id varchar(191) NOT NULL,
  requested_by bigint NULL,
  idempotency_key_hash char(64) NULL,
  submission_fingerprint char(64) NULL,
  state varchar(16) NOT NULL,
  submission_version bigint NOT NULL,
  lease_ttl_ms bigint NOT NULL,
  lease_renewable boolean NOT NULL,
  lease_token_hash char(64) NOT NULL,
  lease_expires_at timestamptz NOT NULL,
  absolute_deadline_at timestamptz NOT NULL,
  prerequisite_kind varchar(191) NOT NULL,
  prerequisite_ref varchar(191) NULL,
  task_id bigint NULL,
  terminal_reason varchar(128) NULL,
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW(),
  activated_at timestamptz NULL,
  terminal_at timestamptz NULL,
  PRIMARY KEY (id),
  CONSTRAINT task_submissions_type_not_blank CHECK (btrim(task_type) <> ''),
  CONSTRAINT task_submissions_owner_type_not_blank CHECK (btrim(owner_type) <> ''),
  CONSTRAINT task_submissions_owner_id_not_blank CHECK (btrim(owner_id) <> ''),
  CONSTRAINT task_submissions_state_check CHECK (state IN ('reserved', 'activated', 'discarded', 'expired')),
  CONSTRAINT task_submissions_version_positive CHECK (submission_version > 0),
  CONSTRAINT task_submissions_lease_ttl_positive CHECK (lease_ttl_ms > 0),
  CONSTRAINT task_submissions_lease_token_hash_sha256_check CHECK (lease_token_hash ~ '^[0-9a-f]{64}$'),
  CONSTRAINT task_submissions_lease_before_deadline CHECK (lease_expires_at < absolute_deadline_at),
  CONSTRAINT task_submissions_prerequisite_kind_not_blank CHECK (btrim(prerequisite_kind) <> ''),
  CONSTRAINT task_submissions_task_id_fkey FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE RESTRICT,
  CONSTRAINT task_submissions_idempotency_key_hash_sha256_check CHECK (idempotency_key_hash IS NULL OR idempotency_key_hash ~ '^[0-9a-f]{64}$'),
  CONSTRAINT task_submissions_fingerprint_sha256_check CHECK (submission_fingerprint IS NULL OR submission_fingerprint ~ '^[0-9a-f]{64}$')
);

CREATE UNIQUE INDEX uq_task_submissions_reserved_owner ON task_submissions (owner_type, owner_id) WHERE state = 'reserved';
CREATE UNIQUE INDEX uq_task_submissions_idempotency ON task_submissions (task_type, owner_type, owner_id, COALESCE(requested_by, 0), idempotency_key_hash) WHERE idempotency_key_hash IS NOT NULL;
CREATE UNIQUE INDEX uq_task_submissions_task_id ON task_submissions (task_id) WHERE task_id IS NOT NULL;
CREATE INDEX idx_task_submissions_expiry ON task_submissions (lease_expires_at ASC, id ASC) WHERE state = 'reserved';

ALTER TABLE tasks DROP CONSTRAINT tasks_status_check;
ALTER TABLE tasks ADD CONSTRAINT tasks_status_check CHECK (status IN ('pending', 'ready', 'scheduled', 'running', 'success', 'failed', 'cancelled', 'needs_attention')) NOT VALID;
ALTER TABLE tasks VALIDATE CONSTRAINT tasks_status_check;
DROP INDEX uq_tasks_active_owner;
CREATE UNIQUE INDEX CONCURRENTLY uq_tasks_active_owner ON tasks (owner_type, owner_id) WHERE status IN ('pending', 'ready', 'scheduled', 'running', 'needs_attention');

COMMENT ON TABLE task_submissions IS '任务物化前的提交与前置条件租约事实表';
COMMENT ON COLUMN task_submissions.id IS '任务提交稳定标识';
COMMENT ON COLUMN task_submissions.task_type IS '提交完成后将创建的业务任务类型';
COMMENT ON COLUMN task_submissions.owner_type IS '任务所属业务资源类型';
COMMENT ON COLUMN task_submissions.owner_id IS '任务所属业务资源稳定标识';
COMMENT ON COLUMN task_submissions.requested_by IS '发起提交的用户标识，系统提交为空';
COMMENT ON COLUMN task_submissions.idempotency_key_hash IS '调用方幂等键的 SHA-256 十六进制摘要';
COMMENT ON COLUMN task_submissions.submission_fingerprint IS '冻结提交内容的规范化 SHA-256 摘要';
COMMENT ON COLUMN task_submissions.state IS '提交状态，reserved、activated、discarded 或 expired';
COMMENT ON COLUMN task_submissions.submission_version IS '所有提交状态和租约变更使用的 fencing 版本';
COMMENT ON COLUMN task_submissions.lease_ttl_ms IS '创建时冻结的单次租约时长，单位毫秒';
COMMENT ON COLUMN task_submissions.lease_renewable IS '是否允许调用方在绝对截止时间前续租';
COMMENT ON COLUMN task_submissions.lease_token_hash IS '提交 handle 租约令牌的 SHA-256 摘要';
COMMENT ON COLUMN task_submissions.lease_expires_at IS '当前租约失效时间，以数据库时间判断';
COMMENT ON COLUMN task_submissions.absolute_deadline_at IS '提交不可续租超过的绝对截止时间';
COMMENT ON COLUMN task_submissions.prerequisite_kind IS '物化前置条件的稳定类型标识';
COMMENT ON COLUMN task_submissions.prerequisite_ref IS '调用模块持久化前置条件后的稳定引用';
COMMENT ON COLUMN task_submissions.task_id IS '原子物化后关联的任务执行主键';
COMMENT ON COLUMN task_submissions.terminal_reason IS '丢弃或过期的稳定终结原因';
COMMENT ON COLUMN task_submissions.created_at IS '提交事实创建时间';
COMMENT ON COLUMN task_submissions.updated_at IS '提交事实最近更新时间';
COMMENT ON COLUMN task_submissions.activated_at IS '提交原子物化为任务的时间';
COMMENT ON COLUMN task_submissions.terminal_at IS '提交进入丢弃或过期终态的时间';
COMMENT ON INDEX uq_task_submissions_reserved_owner IS '同一资源只允许一条持有租约的任务提交';

COMMENT ON COLUMN tasks.activation_required IS '迁移期历史兼容字段，新提交生命周期不使用该字段';
