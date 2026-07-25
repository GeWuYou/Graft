-- atlas:txmode none
-- 用 durable delivery 标识约束审计事实写入，避免并发重试产生重复记录。
CREATE UNIQUE INDEX CONCURRENTLY "audit_logs_event_id_unique"
  ON "audit_logs" (("metadata" ->> 'eventId'))
  WHERE "metadata" ->> 'eventId' IS NOT NULL;
