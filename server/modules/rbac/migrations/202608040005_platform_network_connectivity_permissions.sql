-- 将连通性目标管理与完整出口 IP 权限加入既有 RBAC 权限目录和 Admin 系统角色绑定。
-- 两项权限均涉及受 SSRF 保护的目标配置或敏感网络信息，仅 Admin 可使用。
WITH catalog(code, risk_level, risk_category, action) AS (
  VALUES
    ('platform-network.targets.manage', 'high', 'security', 'manage_targets'),
    ('platform-network.exit-ip.read', 'high', 'security', 'read_exit_ip')
)
INSERT INTO permissions (code, display, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code, 'platform-network', 'platform-network', action, risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
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
JOIN permissions ON permissions.code IN ('platform-network.targets.manage', 'platform-network.exit-ip.read')
WHERE roles.type = 'system'
  AND roles.builtin_key = 'admin'
  AND roles.deleted_at = 0
  AND permissions.deleted_at = 0
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
