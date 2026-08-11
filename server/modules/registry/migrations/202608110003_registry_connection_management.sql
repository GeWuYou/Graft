ALTER TABLE registry_connections
  ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN insecure BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN description VARCHAR(500) NOT NULL DEFAULT '',
  ADD COLUMN auth_mode VARCHAR(32) NOT NULL DEFAULT 'credential_ref',
  ADD COLUMN verification_status VARCHAR(32) NOT NULL DEFAULT 'unknown',
  ADD COLUMN last_verified_at TIMESTAMPTZ NULL,
  ADD COLUMN last_verification_error_code VARCHAR(128) NOT NULL DEFAULT '';

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
