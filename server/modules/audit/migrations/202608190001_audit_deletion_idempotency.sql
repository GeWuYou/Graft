-- atlas:txmode none
-- 在创建唯一索引前阻断历史重复删除凭证，保留所有受保护审计事实供操作员核验。
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM "audit_logs"
    WHERE "action" = 'audit.logs.batch_deleted'
      AND "metadata" ->> 'auditDeletionIdempotencyKey' IS NOT NULL
    GROUP BY "metadata" ->> 'auditDeletionIdempotencyKey'
    HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'duplicate audit deletion idempotency receipts require manual reconciliation';
  END IF;
END
$$;

-- 该部分表达式唯一索引让并发同键删除在数据库边界等待并重放，且不影响普通审计记录。
CREATE UNIQUE INDEX CONCURRENTLY "audit_logs_deletion_idempotency_key_unique"
  ON "audit_logs" (("metadata" ->> 'auditDeletionIdempotencyKey'))
  WHERE "action" = 'audit.logs.batch_deleted'
    AND "metadata" ->> 'auditDeletionIdempotencyKey' IS NOT NULL;
