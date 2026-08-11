ALTER TABLE registry_connections
  ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN insecure BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN description VARCHAR(500) NOT NULL DEFAULT '',
  ADD COLUMN auth_mode VARCHAR(32) NOT NULL DEFAULT 'credential_ref',
  ADD COLUMN verification_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
  ADD COLUMN last_verified_at TIMESTAMPTZ NULL,
  ADD COLUMN last_verification_error_code VARCHAR(128) NOT NULL DEFAULT '';

UPDATE registry_connections
SET auth_mode = CASE WHEN credential_ref IS NULL OR credential_ref = '' THEN 'anonymous' ELSE 'credential_ref' END,
    insecure = LOWER(endpoint) LIKE 'http://%',
    verification_status = CASE WHEN verification_status = 'succeeded' THEN 'verified' ELSE verification_status END
WHERE deleted_at = 0;

DO $$
BEGIN
  IF EXISTS (
    WITH ranked_connections AS (
      SELECT id,
             ROW_NUMBER() OVER (
               PARTITION BY provider, endpoint
               ORDER BY system_managed DESC, updated_at DESC, id DESC
             ) AS position
      FROM registry_connections
      WHERE deleted_at = 0
    )
    SELECT 1
    FROM ranked_connections duplicate
    JOIN artifact_repositories repository ON repository.connection_id = duplicate.id
    WHERE duplicate.position > 1 AND repository.deleted_at = 0
  ) THEN
    RAISE EXCEPTION 'registry connection duplicates referenced by active artifact repositories require manual reconciliation';
  END IF;
END $$;

WITH ranked_connections AS (
  SELECT id,
         ROW_NUMBER() OVER (
           PARTITION BY provider, endpoint
           ORDER BY system_managed DESC, updated_at DESC, id DESC
         ) AS position
  FROM registry_connections
  WHERE deleted_at = 0
)
UPDATE registry_connections
SET deleted_at = EXTRACT(EPOCH FROM NOW())::BIGINT,
    deleted_by = 0,
    updated_at = NOW(),
    updated_by = 0
WHERE id IN (SELECT id FROM ranked_connections WHERE position > 1);

CREATE UNIQUE INDEX uq_registry_connections_live_provider_endpoint
ON registry_connections (provider, endpoint)
WHERE deleted_at = 0;

COMMENT ON COLUMN registry_connections.enabled IS '是否允许新的构建或运行时操作使用该镜像仓库连接';
COMMENT ON COLUMN registry_connections.insecure IS '是否以明确的 HTTP 协议访问连接，false 时必须使用 HTTPS';
COMMENT ON COLUMN registry_connections.description IS '镜像仓库连接的管理说明';
COMMENT ON COLUMN registry_connections.auth_mode IS '连接认证模式，anonymous 表示无认证，credential_ref 表示使用不透明凭据引用';
COMMENT ON COLUMN registry_connections.verification_status IS '最近一次连接验证的稳定状态';
COMMENT ON COLUMN registry_connections.last_verified_at IS '最近一次完成镜像仓库连接验证的时间';
COMMENT ON COLUMN registry_connections.last_verification_error_code IS '最近一次连接验证失败的脱敏稳定错误码';
