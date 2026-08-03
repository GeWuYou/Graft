-- 将平台出站网络策略权限加入既有 RBAC 权限目录与 Admin 系统角色绑定。
-- 网络策略会影响平台主动 HTTP(S) 访问路径，仅 Admin 可管理或执行诊断。
WITH catalog(code, risk_level, risk_category) AS (
  VALUES
    ('platform-network.read', 'low', 'read'),
    ('platform-network.write', 'high', 'security'),
    ('platform-network.diagnose', 'medium', 'write')
)
INSERT INTO permissions (code, display, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code, 'platform-network', 'platform-network', regexp_replace(code, '^platform-network\\.', ''), risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
FROM catalog
ON CONFLICT (code) DO UPDATE SET
  module = EXCLUDED.module,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action,
  risk_level = EXCLUDED.risk_level,
  risk_category = EXCLUDED.risk_category,
  updated_at = CURRENT_TIMESTAMP,
  updated_by = 0;

INSERT INTO role_permissions (role_id, permission_id, created_at, scope)
SELECT roles.id, permissions.id, CURRENT_TIMESTAMP, 'all'
FROM roles
JOIN permissions ON permissions.code IN ('platform-network.read', 'platform-network.write', 'platform-network.diagnose')
WHERE roles.type = 'system'
  AND roles.builtin_key = 'admin'
  AND roles.deleted_at = 0
  AND permissions.deleted_at = 0
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
