UPDATE "access_logs"
SET "trace_id" = NULL
WHERE "trace_id" = '';

ALTER TABLE "access_logs"
  ALTER COLUMN "trace_id" DROP NOT NULL,
  ALTER COLUMN "trace_id" DROP DEFAULT;

DROP INDEX IF EXISTS "idx_access_logs_trace_id";

COMMENT ON TABLE "access_logs" IS 'HTTP 访问日志表';
COMMENT ON COLUMN "access_logs"."id" IS '主键 ID';
COMMENT ON COLUMN "access_logs"."request_id" IS '请求 ID，当前访问日志关联查询主键';
COMMENT ON COLUMN "access_logs"."trace_id" IS '链路追踪 ID 预留字段，为空表示当前请求未记录独立 trace';
COMMENT ON COLUMN "access_logs"."method" IS 'HTTP 请求方法';
COMMENT ON COLUMN "access_logs"."path" IS '规范化后的请求路径';
COMMENT ON COLUMN "access_logs"."route" IS '命中的路由模板';
COMMENT ON COLUMN "access_logs"."status_code" IS 'HTTP 响应状态码';
COMMENT ON COLUMN "access_logs"."duration_ms" IS '请求处理耗时，单位毫秒';
COMMENT ON COLUMN "access_logs"."client_ip" IS '请求来源 IP 地址';
COMMENT ON COLUMN "access_logs"."user_agent" IS '请求客户端标识';
COMMENT ON COLUMN "access_logs"."user_id" IS '认证用户 ID，为空表示匿名请求';
COMMENT ON COLUMN "access_logs"."username" IS '认证用户名，为空表示匿名请求';
COMMENT ON COLUMN "access_logs"."request_size" IS '请求体大小，单位字节';
COMMENT ON COLUMN "access_logs"."response_size" IS '响应体大小，单位字节';
COMMENT ON COLUMN "access_logs"."occurred_at" IS '请求完成时间';
