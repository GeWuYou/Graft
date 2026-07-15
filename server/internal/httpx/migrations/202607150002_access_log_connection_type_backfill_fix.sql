UPDATE "access_logs"
SET "connection_type" = 'http'
WHERE "connection_type" = 'websocket'
  AND "status_code" <> 101
  AND (
    "route" IN ('/ws', '/api/ops/containers/:id/shell/ws')
    OR ("route" IS NULL AND "path" = '/ws')
  );
