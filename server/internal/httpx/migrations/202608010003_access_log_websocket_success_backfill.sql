-- 修复历史成功 WebSocket 握手因 Gin 状态码保持为 200 而被记录为普通 HTTP 的访问日志事实。
UPDATE "access_logs"
SET "connection_type" = 'websocket'
WHERE "connection_type" = 'http'
  AND "status_code" = 200
  AND (
    "route" IN ('/ws', '/api/ops/containers/:id/shell/ws')
    OR ("route" IS NULL AND "path" = '/ws')
  );
