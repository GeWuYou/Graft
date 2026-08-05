-- 将构建中心权限及其风险元数据加入既有 RBAC 目录，并只授予 Admin 系统角色。
-- 构建会触发 Docker 镜像操作，创建、取消和重试均保持管理员受限。
WITH catalog(code, action, risk_level, risk_category) AS (
  VALUES
    ('build.read', 'read', 'low', 'read'),
    ('build.create', 'create', 'high', 'write'),
    ('build.cancel', 'cancel', 'high', 'destructive'),
    ('build.retry', 'retry', 'high', 'write')
)
INSERT INTO permissions (code, display, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code, 'build', 'build', action, risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
FROM catalog
ON CONFLICT (code) DO UPDATE SET
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
JOIN permissions ON permissions.code IN ('build.read', 'build.create', 'build.cancel', 'build.retry')
WHERE roles.type = 'system'
  AND roles.builtin_key = 'admin'
  AND roles.deleted_at = 0
  AND permissions.deleted_at = 0
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
