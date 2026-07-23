CREATE TABLE "event_outbox" (
  "event_id" character varying NOT NULL,
  "event_type" character varying NOT NULL,
  "version" integer NOT NULL,
  "source" character varying NOT NULL,
  "payload" jsonb NOT NULL,
  "metadata" jsonb NOT NULL DEFAULT '{}'::jsonb,
  "occurred_at" timestamptz NOT NULL,
  "created_at" timestamptz NOT NULL,
  "correlation_id" character varying NULL,
  "causation_id" character varying NULL,
  "idempotency_key" character varying NULL,
  PRIMARY KEY ("event_id")
);

CREATE TABLE "event_deliveries" (
  "event_id" character varying NOT NULL,
  "consumer_id" character varying NOT NULL,
  "status" character varying NOT NULL,
  "attempt_count" integer NOT NULL DEFAULT 0,
  "available_at" timestamptz NOT NULL,
  "lease_owner" character varying NULL,
  "lease_expires_at" timestamptz NULL,
  "last_error" text NOT NULL DEFAULT '',
  "delivered_at" timestamptz NULL,
  "created_at" timestamptz NOT NULL,
  "updated_at" timestamptz NOT NULL,
  PRIMARY KEY ("event_id", "consumer_id"),
  CONSTRAINT "event_deliveries_event_id_fkey"
    FOREIGN KEY ("event_id") REFERENCES "event_outbox" ("event_id") ON DELETE CASCADE,
  CONSTRAINT "event_deliveries_status_check"
    CHECK ("status" IN ('pending', 'processing', 'delivered'))
);

-- pending 索引服务于按可用时间 claim 的正常投递路径。
CREATE INDEX "event_deliveries_pending_claim"
  ON "event_deliveries" ("available_at", "created_at")
  WHERE "status" = 'pending';
-- processing 索引让进程崩溃后的过期租约能被其他实例快速恢复。
CREATE INDEX "event_deliveries_expired_lease"
  ON "event_deliveries" ("lease_expires_at")
  WHERE "status" = 'processing';

COMMENT ON TABLE "event_outbox" IS '通用异步事件的持久化事实与重启恢复来源';
COMMENT ON COLUMN "event_outbox"."event_id" IS '事件全局唯一标识，同时作为幂等写入主键';
COMMENT ON COLUMN "event_outbox"."event_type" IS '稳定事件类型，用于定位已注册消费者';
COMMENT ON COLUMN "event_outbox"."version" IS '事件载荷版本，由事件所有者定义兼容语义';
COMMENT ON COLUMN "event_outbox"."source" IS '发布事件的稳定模块或核心能力标识';
COMMENT ON COLUMN "event_outbox"."payload" IS '事件业务载荷的 JSON 快照，由事件类型所有者解释';
COMMENT ON COLUMN "event_outbox"."metadata" IS '可选事件上下文 JSON 快照，不承载消费者投递状态';
COMMENT ON COLUMN "event_outbox"."occurred_at" IS '业务事件实际发生时间';
COMMENT ON COLUMN "event_outbox"."created_at" IS '事件写入 Outbox 的时间';
COMMENT ON COLUMN "event_outbox"."correlation_id" IS '请求或链路关联标识，用于跨系统追踪';
COMMENT ON COLUMN "event_outbox"."causation_id" IS '触发当前事件的上游事件标识';
COMMENT ON COLUMN "event_outbox"."idempotency_key" IS '发布方提供的业务幂等键，供事件消费者诊断';

COMMENT ON TABLE "event_deliveries" IS '异步事件按消费者独立维护的可靠投递状态';
COMMENT ON COLUMN "event_deliveries"."event_id" IS '关联的 Outbox 事件标识';
COMMENT ON COLUMN "event_deliveries"."consumer_id" IS '事件处理器稳定标识，作为消费者幂等边界';
COMMENT ON COLUMN "event_deliveries"."status" IS '投递状态，取值为 pending、processing 或 delivered';
COMMENT ON COLUMN "event_deliveries"."attempt_count" IS '该消费者已被 claim 并执行的尝试次数';
COMMENT ON COLUMN "event_deliveries"."available_at" IS 'pending 投递最早允许再次 claim 的时间';
COMMENT ON COLUMN "event_deliveries"."lease_owner" IS '当前处理实例生成的短期租约所有者标识';
COMMENT ON COLUMN "event_deliveries"."lease_expires_at" IS 'processing 租约到期时间，到期后可由其他实例恢复';
COMMENT ON COLUMN "event_deliveries"."last_error" IS '最近一次消费者执行失败的错误摘要';
COMMENT ON COLUMN "event_deliveries"."delivered_at" IS '消费者成功完成处理的时间';
COMMENT ON COLUMN "event_deliveries"."created_at" IS '消费者投递记录创建时间';
COMMENT ON COLUMN "event_deliveries"."updated_at" IS '消费者投递状态最近更新时间';
