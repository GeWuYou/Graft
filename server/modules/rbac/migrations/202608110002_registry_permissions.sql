WITH catalog(code, action, risk_level, risk_category) AS (
  VALUES
    ('registry.read', 'read', 'low', 'read'),
    ('registry.create', 'create', 'high', 'write'),
    ('registry.update', 'update', 'high', 'write'),
    ('registry.delete', 'delete', 'high', 'destructive'),
    ('registry.verify', 'verify', 'high', 'security'),
    ('registry.assignment.manage', 'assignment.manage', 'high', 'security')
)
INSERT INTO permissions (code, display, display_key, description, description_key, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code,
  CASE code
    WHEN 'registry.read' THEN 'rbac.permissionCatalog.registryRead.display'
    WHEN 'registry.create' THEN 'rbac.permissionCatalog.registryCreate.display'
    WHEN 'registry.update' THEN 'rbac.permissionCatalog.registryUpdate.display'
    WHEN 'registry.delete' THEN 'rbac.permissionCatalog.registryDelete.display'
    WHEN 'registry.verify' THEN 'rbac.permissionCatalog.registryVerify.display'
    ELSE 'rbac.permissionCatalog.registryAssignmentManage.display'
  END,
  code,
  CASE code
    WHEN 'registry.read' THEN 'rbac.permissionCatalog.registryRead.description'
    WHEN 'registry.create' THEN 'rbac.permissionCatalog.registryCreate.description'
    WHEN 'registry.update' THEN 'rbac.permissionCatalog.registryUpdate.description'
    WHEN 'registry.delete' THEN 'rbac.permissionCatalog.registryDelete.description'
    WHEN 'registry.verify' THEN 'rbac.permissionCatalog.registryVerify.description'
    ELSE 'rbac.permissionCatalog.registryAssignmentManage.description'
  END,
  'registry', 'registry', action, risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
FROM catalog
ON CONFLICT (code) DO UPDATE SET display_key = EXCLUDED.display_key, description_key = EXCLUDED.description_key, module = EXCLUDED.module, resource = EXCLUDED.resource, action = EXCLUDED.action, risk_level = EXCLUDED.risk_level, risk_category = EXCLUDED.risk_category, deleted_at = 0, deleted_by = 0, updated_at = CURRENT_TIMESTAMP, updated_by = 0;

INSERT INTO role_permissions (role_id, permission_id, created_at, scope)
SELECT roles.id, permissions.id, CURRENT_TIMESTAMP, 'all'
FROM roles JOIN permissions ON permissions.code IN ('registry.read', 'registry.create', 'registry.update', 'registry.delete', 'registry.verify', 'registry.assignment.manage')
WHERE roles.type = 'system' AND roles.builtin_key = 'admin' AND roles.deleted_at = 0 AND permissions.deleted_at = 0
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
