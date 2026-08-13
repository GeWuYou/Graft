-- 前向修复 202608130001 的历史扫描谓词：唯一索引覆盖全部未软删除仓库。
DROP INDEX IF EXISTS uq_artifact_repositories_live_connection_display_name;

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

-- 同一连接内所有未软删除仓库的展示名称必须唯一，连接已软删除不改变仓库记录的唯一性约束。
CREATE UNIQUE INDEX uq_artifact_repositories_live_connection_display_name
ON artifact_repositories (connection_id, display_name)
WHERE deleted_at = 0;
