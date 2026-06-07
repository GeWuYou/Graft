-- system_config_values stores administrator overrides only.
-- ConfigDefinition metadata and defaults remain module-registered runtime authority.

CREATE TABLE IF NOT EXISTS system_config_values (
    key TEXT PRIMARY KEY,
    override_value JSONB NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE system_config_values IS '管理员提供的系统配置覆盖值表';
COMMENT ON COLUMN system_config_values.key IS '模块注册的稳定配置定义键';
COMMENT ON COLUMN system_config_values.override_value IS '管理员覆盖 JSON；模块默认值不会复制到此表';
COMMENT ON COLUMN system_config_values.updated_at IS '最近一次覆盖值写入时间';
