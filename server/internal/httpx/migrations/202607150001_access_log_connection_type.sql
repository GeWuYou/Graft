ALTER TABLE "access_logs"
  ADD COLUMN "connection_type" character varying NOT NULL DEFAULT 'http'
  CHECK ("connection_type" IN ('http', 'websocket'));

COMMENT ON COLUMN "access_logs"."connection_type" IS '访问连接类型，http 表示普通请求响应，websocket 表示成功升级后的长连接';

UPDATE "access_logs"
SET "connection_type" = 'websocket'
WHERE "route" IN ('/ws', '/api/ops/containers/:id/shell/ws')
   OR ("route" IS NULL AND "path" = '/ws');
