-- 将权限编码收敛到模块或功能前缀，并合并可能已经存在的目标权限绑定。
CREATE TEMP TABLE permission_code_migration AS
WITH mapped AS (
  SELECT
    permissions.id AS old_id,
    permissions.code AS old_code,
    CASE permissions.code
      WHEN 'ops.project.source.view' THEN 'application.creation-method.view'
      WHEN 'ops.project.view' THEN 'application.view'
      WHEN 'ops.project.import' THEN 'application.import'
      WHEN 'ops.project.refresh' THEN 'application.refresh'
      WHEN 'ops.project.lifecycle' THEN 'application.lifecycle'
      WHEN 'ops.project.destroy' THEN 'application.destroy'
      WHEN 'ops.project.create' THEN 'application.create'
      WHEN 'ops.project.creation-method.view' THEN 'application.creation-method.view'
      WHEN 'ops.project.discovery.view' THEN 'application.discovery.view'
      WHEN 'ops.project.deploy' THEN 'application.deploy'
      ELSE substring(permissions.code FROM 5)
    END AS new_code
  FROM permissions
  WHERE permissions.code LIKE 'ops.%'
)
SELECT
  mapped.old_id,
  mapped.old_code,
  mapped.new_code,
  COALESCE(target.id, MIN(mapped.old_id) OVER (PARTITION BY mapped.new_code)) AS canonical_id
FROM mapped
LEFT JOIN permissions AS target ON target.code = mapped.new_code;

COMMENT ON TABLE permission_code_migration IS '权限编码迁移过程中的旧编码与规范编码映射';
COMMENT ON COLUMN permission_code_migration.old_id IS '待迁移旧权限记录 ID';
COMMENT ON COLUMN permission_code_migration.old_code IS '待迁移旧权限编码';
COMMENT ON COLUMN permission_code_migration.new_code IS '迁移后的规范权限编码';
COMMENT ON COLUMN permission_code_migration.canonical_id IS '保留并承载角色绑定的规范权限记录 ID';

INSERT INTO role_permissions (role_id, permission_id, created_at)
SELECT bindings.role_id, mappings.canonical_id, bindings.created_at
FROM permission_code_migration AS mappings
JOIN role_permissions AS bindings ON bindings.permission_id = mappings.old_id
WHERE mappings.old_id <> mappings.canonical_id
ON CONFLICT (role_id, permission_id) DO NOTHING;

DELETE FROM role_permissions
WHERE permission_id IN (
  SELECT old_id
  FROM permission_code_migration
  WHERE old_id <> canonical_id
);

DELETE FROM permissions
WHERE id IN (
  SELECT old_id
  FROM permission_code_migration
  WHERE old_id <> canonical_id
);

UPDATE permissions
SET code = mappings.new_code,
    updated_at = CURRENT_TIMESTAMP
FROM permission_code_migration AS mappings
WHERE permissions.id = mappings.canonical_id
  AND permissions.code <> mappings.new_code;

COMMENT ON COLUMN permissions.code IS '权限点编码，使用模块或功能前缀标识权限归属';
