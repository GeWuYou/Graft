WITH permission_mapping(old_code, new_code, display_key, description_key) AS (
  VALUES
    ('ops.project.view', 'ops.application.view', 'rbac.permissionCatalog.applicationView.display', 'rbac.permissionCatalog.applicationView.description'),
    ('ops.project.import', 'ops.application.import', 'rbac.permissionCatalog.applicationImport.display', 'rbac.permissionCatalog.applicationImport.description'),
    ('ops.project.refresh', 'ops.application.refresh', 'rbac.permissionCatalog.applicationRefresh.display', 'rbac.permissionCatalog.applicationRefresh.description'),
    ('ops.project.lifecycle', 'ops.application.lifecycle', 'rbac.permissionCatalog.applicationLifecycle.display', 'rbac.permissionCatalog.applicationLifecycle.description'),
    ('ops.project.destroy', 'ops.application.destroy', 'rbac.permissionCatalog.applicationDestroy.display', 'rbac.permissionCatalog.applicationDestroy.description'),
    ('ops.project.create', 'ops.application.create', 'rbac.permissionCatalog.applicationCreate.display', 'rbac.permissionCatalog.applicationCreate.description'),
    ('ops.project.creation-method.view', 'ops.application.creation-method.view', 'rbac.permissionCatalog.applicationCreationMethodView.display', 'rbac.permissionCatalog.applicationCreationMethodView.description'),
    ('ops.project.discovery.view', 'ops.application.discovery.view', 'rbac.permissionCatalog.applicationDiscoveryView.display', 'rbac.permissionCatalog.applicationDiscoveryView.description'),
    ('ops.project.deploy', 'ops.application.deploy', 'rbac.permissionCatalog.applicationDeploy.display', 'rbac.permissionCatalog.applicationDeploy.description')
)
INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT role_permissions.role_id, replacement.id, role_permissions.created_at
FROM permission_mapping
JOIN permissions AS legacy ON legacy.code = permission_mapping.old_code
JOIN permissions AS replacement ON replacement.code = permission_mapping.new_code
JOIN role_permissions ON role_permissions.permission_id = legacy.id
ON CONFLICT (role_id, permission_id) DO NOTHING;

WITH permission_mapping(old_code, new_code) AS (
  VALUES
    ('ops.project.view', 'ops.application.view'),
    ('ops.project.import', 'ops.application.import'),
    ('ops.project.refresh', 'ops.application.refresh'),
    ('ops.project.lifecycle', 'ops.application.lifecycle'),
    ('ops.project.destroy', 'ops.application.destroy'),
    ('ops.project.create', 'ops.application.create'),
    ('ops.project.creation-method.view', 'ops.application.creation-method.view'),
    ('ops.project.discovery.view', 'ops.application.discovery.view'),
    ('ops.project.deploy', 'ops.application.deploy')
)
DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT legacy.id
  FROM permission_mapping
  JOIN permissions AS legacy ON legacy.code = permission_mapping.old_code
  JOIN permissions AS replacement ON replacement.code = permission_mapping.new_code
);

WITH permission_mapping(old_code, new_code) AS (
  VALUES
    ('ops.project.view', 'ops.application.view'),
    ('ops.project.import', 'ops.application.import'),
    ('ops.project.refresh', 'ops.application.refresh'),
    ('ops.project.lifecycle', 'ops.application.lifecycle'),
    ('ops.project.destroy', 'ops.application.destroy'),
    ('ops.project.create', 'ops.application.create'),
    ('ops.project.creation-method.view', 'ops.application.creation-method.view'),
    ('ops.project.discovery.view', 'ops.application.discovery.view'),
    ('ops.project.deploy', 'ops.application.deploy')
)
DELETE FROM permissions
USING permission_mapping
WHERE permissions.code = permission_mapping.old_code
  AND EXISTS (SELECT 1 FROM permissions AS replacement WHERE replacement.code = permission_mapping.new_code);

WITH permission_mapping(old_code, new_code, display_key, description_key) AS (
  VALUES
    ('ops.project.view', 'ops.application.view', 'rbac.permissionCatalog.applicationView.display', 'rbac.permissionCatalog.applicationView.description'),
    ('ops.project.import', 'ops.application.import', 'rbac.permissionCatalog.applicationImport.display', 'rbac.permissionCatalog.applicationImport.description'),
    ('ops.project.refresh', 'ops.application.refresh', 'rbac.permissionCatalog.applicationRefresh.display', 'rbac.permissionCatalog.applicationRefresh.description'),
    ('ops.project.lifecycle', 'ops.application.lifecycle', 'rbac.permissionCatalog.applicationLifecycle.display', 'rbac.permissionCatalog.applicationLifecycle.description'),
    ('ops.project.destroy', 'ops.application.destroy', 'rbac.permissionCatalog.applicationDestroy.display', 'rbac.permissionCatalog.applicationDestroy.description'),
    ('ops.project.create', 'ops.application.create', 'rbac.permissionCatalog.applicationCreate.display', 'rbac.permissionCatalog.applicationCreate.description'),
    ('ops.project.creation-method.view', 'ops.application.creation-method.view', 'rbac.permissionCatalog.applicationCreationMethodView.display', 'rbac.permissionCatalog.applicationCreationMethodView.description'),
    ('ops.project.discovery.view', 'ops.application.discovery.view', 'rbac.permissionCatalog.applicationDiscoveryView.display', 'rbac.permissionCatalog.applicationDiscoveryView.description'),
    ('ops.project.deploy', 'ops.application.deploy', 'rbac.permissionCatalog.applicationDeploy.display', 'rbac.permissionCatalog.applicationDeploy.description')
)
UPDATE permissions
SET code = permission_mapping.new_code,
    display_key = permission_mapping.display_key,
    description_key = permission_mapping.description_key,
    updated_at = CURRENT_TIMESTAMP
FROM permission_mapping
WHERE permissions.code = permission_mapping.old_code;

COMMENT ON COLUMN permissions.code IS '权限点编码，Application 管理权限使用 ops.application 前缀';
COMMENT ON COLUMN permissions.display_key IS '权限点显示名称本地化 key，Application 权限使用 application 命名空间';
COMMENT ON COLUMN permissions.description_key IS '权限点描述本地化 key，Application 权限使用 application 命名空间';
