DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM artifact_repositories repository
    WHERE repository.deleted_at = 0
    GROUP BY repository.connection_id, repository.display_name
    HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'duplicate live artifact repository display names require manual reconciliation';
  END IF;
END $$;

-- 同一连接内的展示名称必须唯一，避免详情页授权操作产生歧义；软删除记录不参与约束。
CREATE UNIQUE INDEX uq_artifact_repositories_live_connection_display_name
ON artifact_repositories (connection_id, display_name)
WHERE deleted_at = 0;
