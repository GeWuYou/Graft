-- 将角色模型升级为系统角色与自定义角色，并增加权限目录分组和风险元数据。
-- 版本编号已在默认迁移链中全局去重。
ALTER TABLE roles
  ADD COLUMN IF NOT EXISTS type VARCHAR(16) NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS builtin_key VARCHAR(64),
  ADD COLUMN IF NOT EXISTS editable BOOLEAN NOT NULL DEFAULT TRUE;

COMMENT ON COLUMN roles.type IS '角色类型，system 表示内置角色，custom 表示自定义角色';
COMMENT ON COLUMN roles.builtin_key IS '内置角色的稳定策略键，自定义角色为空';
COMMENT ON COLUMN roles.editable IS '角色核心定义和权限绑定是否可编辑';

UPDATE roles
SET type = CASE WHEN builtin THEN 'system' ELSE 'custom' END,
    builtin_key = CASE WHEN builtin AND name = 'admin' THEN 'admin' ELSE builtin_key END,
    editable = NOT builtin;

ALTER TABLE roles
  ADD CONSTRAINT roles_type_check CHECK (type IN ('system', 'custom')),
  ADD CONSTRAINT roles_system_contract_check CHECK (
    (type = 'system' AND builtin = TRUE AND builtin_key IS NOT NULL AND editable = FALSE)
    OR (type = 'custom' AND builtin = FALSE AND builtin_key IS NULL AND editable = TRUE)
  );

CREATE UNIQUE INDEX roles_builtin_key_system_unique
  ON roles (builtin_key)
  WHERE type = 'system' AND deleted_at = 0;

ALTER TABLE permissions
  ADD COLUMN IF NOT EXISTS resource VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS action VARCHAR(128) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS risk_level VARCHAR(16) NOT NULL DEFAULT 'read';

COMMENT ON COLUMN permissions.resource IS '权限所属资源领域，用于目录分组';
COMMENT ON COLUMN permissions.action IS '资源上的稳定动作';
COMMENT ON COLUMN permissions.risk_level IS '权限风险等级：read、write、destructive 或 security';

ALTER TABLE permissions
  ADD CONSTRAINT permissions_risk_level_check CHECK (risk_level IN ('read', 'write', 'destructive', 'security'));

ALTER TABLE role_permissions
  ADD COLUMN IF NOT EXISTS scope VARCHAR(16) NOT NULL DEFAULT 'all';

COMMENT ON COLUMN role_permissions.scope IS '权限绑定范围：all 或 owned';

ALTER TABLE role_permissions
  ADD CONSTRAINT role_permissions_scope_check CHECK (scope IN ('all', 'owned'));

-- 系统角色只由版本化策略迁移创建和同步，服务启动不得在这里外写入策略。
INSERT INTO roles (name, display, description, builtin, type, builtin_key, editable, created_at, created_by, updated_at, updated_by, disabled_at, deleted_at, deleted_by)
VALUES
  ('admin', '管理员', '完整平台管理权限', TRUE, 'system', 'admin', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('platform_operator', '平台运维', '基础设施与运行时运维', TRUE, 'system', 'platform_operator', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('application_operator', '应用运维', '应用发布和故障处理', TRUE, 'system', 'application_operator', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('developer', '开发者', '开发人员部署自己的应用', TRUE, 'system', 'developer', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('viewer', '只读用户', '平台只读访问', TRUE, 'system', 'viewer', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('monitor', '监控用户', '平台监控访问', TRUE, 'system', 'monitor', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('security_auditor', '安全审计员', '审计和安全证据只读访问', TRUE, 'system', 'security_auditor', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0),
  ('no_shell_operator', '受限运维', '不含终端与容器执行能力的运维访问', TRUE, 'system', 'no_shell_operator', FALSE, CURRENT_TIMESTAMP, 0, CURRENT_TIMESTAMP, 0, 0, 0, 0)
ON CONFLICT (name) DO UPDATE
SET builtin = EXCLUDED.builtin,
    type = EXCLUDED.type,
    builtin_key = EXCLUDED.builtin_key,
    editable = EXCLUDED.editable,
    updated_at = CURRENT_TIMESTAMP,
    updated_by = 0
WHERE roles.type = 'system';
