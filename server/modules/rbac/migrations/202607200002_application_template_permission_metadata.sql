-- 补齐应用模板权限的本地化 metadata，避免权限目录展示数据库中的英文 fallback。
WITH permission_metadata(code, display_key, description_key) AS (
  VALUES
    ('ops.application.template.manage', 'rbac.permissionCatalog.applicationTemplateManage.display', 'rbac.permissionCatalog.applicationTemplateManage.description'),
    ('ops.application.template.publish', 'rbac.permissionCatalog.applicationTemplatePublish.display', 'rbac.permissionCatalog.applicationTemplatePublish.description')
)
UPDATE permissions
SET display_key = permission_metadata.display_key,
    description_key = permission_metadata.description_key,
    updated_at = CURRENT_TIMESTAMP
FROM permission_metadata
WHERE permissions.code = permission_metadata.code;
