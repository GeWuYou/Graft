CREATE TABLE "app_logs" (
  "id" bigserial PRIMARY KEY,
  "occurred_at" timestamptz NOT NULL,
  "severity" varchar(16) NOT NULL,
  "component" varchar(191) NOT NULL,
  "operation" varchar(191) NULL,
  "request_id" varchar(64) NULL,
  "trace_id" varchar(64) NULL,
  "route" text NULL,
  "method" varchar(16) NULL,
  "error" text NULL,
  "message" text NOT NULL,
  "fields" jsonb NOT NULL DEFAULT '{}'::jsonb
);

CREATE INDEX "idx_app_logs_occurred_at_id" ON "app_logs" ("occurred_at" DESC, "id" DESC);
CREATE INDEX "idx_app_logs_severity_occurred_at" ON "app_logs" ("severity", "occurred_at" DESC);
CREATE INDEX "idx_app_logs_component_occurred_at" ON "app_logs" ("component", "occurred_at" DESC);
CREATE INDEX "idx_app_logs_request_id" ON "app_logs" ("request_id");
CREATE INDEX "idx_app_logs_trace_id" ON "app_logs" ("trace_id");
CREATE INDEX "idx_app_logs_keyword_search"
ON "app_logs"
USING GIN (
  to_tsvector(
    'simple',
    "component" || ' ' || COALESCE("operation", '') || ' ' || "message" || ' ' || COALESCE("error", '')
  )
);

COMMENT ON TABLE "app_logs" IS '应用运行日志表';
COMMENT ON COLUMN "app_logs"."id" IS '主键 ID';
COMMENT ON COLUMN "app_logs"."occurred_at" IS '日志发生时间';
COMMENT ON COLUMN "app_logs"."severity" IS '日志级别：debug、info、warn、error';
COMMENT ON COLUMN "app_logs"."component" IS '应用日志组件标识';
COMMENT ON COLUMN "app_logs"."operation" IS '稳定操作名称';
COMMENT ON COLUMN "app_logs"."request_id" IS '请求关联 ID，为空表示非 HTTP 场景或未提供';
COMMENT ON COLUMN "app_logs"."trace_id" IS '链路关联 ID，MVP 阶段通常等同或派生自 request_id';
COMMENT ON COLUMN "app_logs"."route" IS 'HTTP 路由模板，为空表示非 HTTP 场景';
COMMENT ON COLUMN "app_logs"."method" IS 'HTTP 请求方法，为空表示非 HTTP 场景';
COMMENT ON COLUMN "app_logs"."error" IS '错误摘要文本';
COMMENT ON COLUMN "app_logs"."message" IS '日志消息正文';
COMMENT ON COLUMN "app_logs"."fields" IS '边界内结构化扩展字段，不包含访问、审计或安全归属字段';
