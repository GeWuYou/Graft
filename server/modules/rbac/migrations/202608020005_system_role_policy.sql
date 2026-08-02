-- 将历史权限风险类别迁移为独立的严重级别与操作类别，并同步批准的系统角色策略。
-- 本迁移只面向既有 RBAC 表，始终向前兼容，绝不修改自定义角色绑定。
ALTER TABLE permissions
  ADD COLUMN IF NOT EXISTS risk_category VARCHAR(16) NOT NULL DEFAULT 'read';

ALTER TABLE permissions
  DROP CONSTRAINT IF EXISTS permissions_risk_level_check;

UPDATE permissions
SET risk_category = CASE risk_level
      WHEN 'read' THEN 'read'
      WHEN 'write' THEN 'write'
      WHEN 'destructive' THEN 'destructive'
      WHEN 'security' THEN 'security'
      ELSE 'read'
    END,
    risk_level = CASE risk_level
      WHEN 'destructive' THEN 'high'
      WHEN 'security' THEN 'high'
      WHEN 'write' THEN 'medium'
      ELSE 'low'
    END;

ALTER TABLE permissions
  ADD CONSTRAINT permissions_risk_level_check CHECK (risk_level IN ('low', 'medium', 'high', 'critical')),
  ADD CONSTRAINT permissions_risk_category_check CHECK (risk_category IN ('read', 'write', 'destructive', 'security'));

COMMENT ON COLUMN permissions.risk_level IS '权限严重级别：low、medium、high 或 critical';
COMMENT ON COLUMN permissions.risk_category IS '权限操作类别：read、write、destructive 或 security';

-- 目录以稳定权限编码为唯一键 UPSERT，保留历史行 ID 和所有现有角色引用。
WITH catalog(code, risk_level, risk_category) AS (
  VALUES
    ('announcement.read','low','read'),('announcement.create','medium','write'),('announcement.update','medium','write'),('announcement.publish','medium','write'),('announcement.delete','medium','destructive'),
    ('audit.read','low','read'),('audit.manage','high','security'),
    ('container.view','low','read'),('container.detail','low','read'),('container.events','low','read'),('container.logs','low','read'),('container.environment','high','security'),('container.shell','critical','security'),('container.start','medium','write'),('container.stop','medium','write'),('container.restart','medium','write'),('container.remove','high','destructive'),('container.volume.remove','high','destructive'),('container.image.pull','medium','write'),('container.image.tag','medium','write'),('container.image.untag','high','destructive'),('container.image.remove','high','destructive'),('container.network.create','medium','write'),('container.network.remove','high','destructive'),
    ('access_log.read','low','read'),('app_log.read','low','read'),('app_log.delete','high','destructive'),('modules.runtime.read','low','read'),('monitor.server-status.read','low','read'),('security.overview.read','low','read'),
    ('notification.view','low','read'),('notification.read','medium','write'),('notification.manage','medium','write'),('platform-backup.read','low','read'),('platform-backup.create','high','write'),('platform-update.read','low','read'),('platform-update.check','medium','write'),('platform-update.manage','high','security'),
    ('application.view','low','read'),('application.create','medium','write'),('application.deploy','medium','write'),('application.lifecycle','medium','write'),('application.import','medium','write'),('application.refresh','medium','write'),('application.discovery.view','medium','read'),('application.destroy','high','destructive'),('application.creation-method.view','low','read'),('application.template.manage','medium','write'),('application.template.publish','medium','write'),
    ('role.read','low','read'),('role.create','high','security'),('role.update','high','security'),('role.status.update','high','security'),('role.delete','high','security'),('role.permission.assign','critical','security'),('permission.read','low','read'),('user.role.read','low','read'),('user.role.assign','critical','security'),
    ('runtime_target.view','low','read'),('runtime_target.manage','high','write'),('runtime_target.assignment.manage','high','security'),('runtime_target.refresh','medium','write'),
    ('scheduled-task.read','low','read'),('scheduled-task.create','medium','write'),('scheduled-task.update','medium','write'),('scheduled-task.delete','medium','destructive'),('scheduled-task.run','medium','write'),('scheduled-task.enable','medium','write'),
    ('system-config.read','low','read'),('system-config.write','critical','security'),('user.read','low','read'),('user.create','medium','write'),('user.update','medium','write'),('user.disable','high','security'),('user.session.read','low','read'),('user.session.revoke','high','security')
)
INSERT INTO permissions (code, display, module, resource, action, risk_level, risk_category, created_at, created_by, updated_at, updated_by, deleted_at, deleted_by)
SELECT code, code, 'policy-migration', split_part(code, '.', 1), regexp_replace(code, '^[^.]+\\.', ''), risk_level, risk_category, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0
FROM catalog
ON CONFLICT (code) DO UPDATE SET
  resource = EXCLUDED.resource, action = EXCLUDED.action, risk_level = EXCLUDED.risk_level, risk_category = EXCLUDED.risk_category,
  updated_at = CURRENT_TIMESTAMP, updated_by = 0;

-- 仅 type='system' 的角色绑定会被替换；同名自定义角色和全部自定义绑定保持不动。
DELETE FROM role_permissions rp
USING roles r
WHERE rp.role_id = r.id AND r.type = 'system';

WITH grants(role_key, code, scope) AS (
  VALUES
    ('platform_operator','container.view','all'),('platform_operator','container.detail','all'),('platform_operator','container.events','all'),('platform_operator','container.logs','all'),('platform_operator','container.start','all'),('platform_operator','container.stop','all'),('platform_operator','container.restart','all'),('platform_operator','container.image.pull','all'),
    ('no_shell_operator','container.view','all'),('no_shell_operator','container.detail','all'),('no_shell_operator','container.events','all'),('no_shell_operator','container.logs','all'),('no_shell_operator','container.start','all'),('no_shell_operator','container.stop','all'),('no_shell_operator','container.restart','all'),
    ('application_operator','application.view','all'),('application_operator','application.deploy','all'),('application_operator','application.lifecycle','all'),('application_operator','application.import','all'),('application_operator','application.refresh','all'),('application_operator','application.discovery.view','all'),
    ('developer','application.view','owned'),('developer','application.create','owned'),('developer','application.deploy','owned'),
    ('viewer','announcement.read','all'),('viewer','monitor.server-status.read','all'),('monitor','monitor.server-status.read','all'),('monitor','access_log.read','all'),('monitor','app_log.read','all'),('security_auditor','audit.read','all'),('security_auditor','security.overview.read','all')
), admin_grants AS (
  SELECT r.id AS role_id, p.id AS permission_id, 'all'::VARCHAR AS scope FROM roles r CROSS JOIN permissions p WHERE r.type = 'system' AND r.builtin_key = 'admin' AND r.deleted_at = 0 AND p.deleted_at = 0
), named_grants AS (
  SELECT r.id AS role_id, p.id AS permission_id, g.scope FROM grants g JOIN roles r ON r.type = 'system' AND r.builtin_key = g.role_key AND r.deleted_at = 0 JOIN permissions p ON p.code = g.code AND p.deleted_at = 0
)
INSERT INTO role_permissions (role_id, permission_id, created_at, scope)
SELECT role_id, permission_id, CURRENT_TIMESTAMP, scope FROM admin_grants
UNION ALL SELECT role_id, permission_id, CURRENT_TIMESTAMP, scope FROM named_grants
ON CONFLICT (role_id, permission_id) DO UPDATE SET scope = EXCLUDED.scope;
