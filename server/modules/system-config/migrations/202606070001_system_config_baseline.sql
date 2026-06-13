-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

-- system_config_values stores user overrides only.
-- ConfigDefinition metadata and defaults remain module-registered runtime authority.

CREATE TABLE system_config_values (
    key TEXT PRIMARY KEY,
    override_value JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by BIGINT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by BIGINT NULL
);

COMMENT ON TABLE system_config_values IS '用户提供的系统配置覆盖值表';
COMMENT ON COLUMN system_config_values.key IS '模块注册的稳定配置定义键';
COMMENT ON COLUMN system_config_values.override_value IS '用户覆盖 JSON；模块默认值不会复制到此表';
COMMENT ON COLUMN system_config_values.created_at IS '覆盖值首次写入时间';
COMMENT ON COLUMN system_config_values.created_by IS '首次写入覆盖值的用户 ID；为空表示请求上下文未提供用户';
COMMENT ON COLUMN system_config_values.updated_at IS '最近一次覆盖值写入时间';
COMMENT ON COLUMN system_config_values.updated_by IS '最近一次写入覆盖值的用户 ID；为空表示请求上下文未提供用户';
