-- 将平台能力健康快照权限加入 RBAC 目录，并仅授予 Admin 系统角色。
-- 该权限是 core 只读诊断能力；自定义角色及其绑定保持不变。
WITH catalog(code, display_key, description_key, action, risk_level, risk_category) AS (
  VALUES
    ('platform.capabilities.read', 'rbac.permissionCatalog.platformCapabilitiesRead.display', 'rbac.permissionCatalog.platformCapabilitiesRead.description', 'read', 'low', 'read')
)
INSERT INTO permissions (code, display, display_key, description, description_key, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code, display_key, code, description_key, 'core', 'platform-capabilities', action, risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
FROM catalog
ON CONFLICT (code) DO UPDATE SET
  display_key = EXCLUDED.display_key,
  description_key = EXCLUDED.description_key,
  module = EXCLUDED.module,
  resource = EXCLUDED.resource,
  action = EXCLUDED.action,
  risk_level = EXCLUDED.risk_level,
  risk_category = EXCLUDED.risk_category,
  deleted_at = 0,
  deleted_by = 0,
  updated_at = CURRENT_TIMESTAMP,
  updated_by = 0;

INSERT INTO role_permissions (role_id, permission_id, created_at, scope)
SELECT roles.id, permissions.id, CURRENT_TIMESTAMP, 'all'
FROM roles
JOIN permissions ON permissions.code = 'platform.capabilities.read'
WHERE roles.type = 'system'
  AND roles.builtin_key = 'admin'
  AND roles.deleted_at = 0
  AND permissions.deleted_at = 0
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
