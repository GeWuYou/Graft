INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT role_permissions.role_id, replacement.id, role_permissions.created_at
FROM role_permissions
JOIN permissions AS legacy ON legacy.id = role_permissions.permission_id
JOIN permissions AS replacement ON replacement.code = 'ops.project.creation-method.view'
WHERE legacy.code = 'ops.project.source.view'
ON CONFLICT (role_id, permission_id) DO NOTHING;

DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT id
  FROM permissions
  WHERE code = 'ops.project.source.view'
)
  AND EXISTS (
    SELECT 1
    FROM permissions
    WHERE code = 'ops.project.creation-method.view'
  );

DELETE FROM permissions
WHERE code = 'ops.project.source.view'
  AND EXISTS (
    SELECT 1
    FROM permissions
    WHERE code = 'ops.project.creation-method.view'
  );

UPDATE permissions
SET code = 'ops.project.creation-method.view',
    display_key = 'rbac.permissionCatalog.projectCreationMethodView.display',
    description_key = 'rbac.permissionCatalog.projectCreationMethodView.description',
    updated_at = CURRENT_TIMESTAMP
WHERE code = 'ops.project.source.view'
  AND NOT EXISTS (
    SELECT 1
    FROM permissions
    WHERE code = 'ops.project.creation-method.view'
  );

UPDATE permissions
SET display_key = 'rbac.permissionCatalog.projectCreationMethodView.display',
    description_key = 'rbac.permissionCatalog.projectCreationMethodView.description',
    updated_at = CURRENT_TIMESTAMP
WHERE code = 'ops.project.creation-method.view';
