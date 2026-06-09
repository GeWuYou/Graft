-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

-- system_config_values records which user wrote each override when request context provides one.

ALTER TABLE system_config_values
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMPTZ NULL,
    ADD COLUMN IF NOT EXISTS created_by BIGINT NULL,
    ADD COLUMN IF NOT EXISTS updated_by BIGINT NULL;

UPDATE system_config_values
SET created_at = updated_at
WHERE created_at IS NULL;

ALTER TABLE system_config_values
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN created_at SET NOT NULL;

COMMENT ON TABLE system_config_values IS '用户提供的系统配置覆盖值表';
COMMENT ON COLUMN system_config_values.key IS '模块注册的稳定配置定义键';
COMMENT ON COLUMN system_config_values.override_value IS '用户覆盖 JSON；模块默认值不会复制到此表';
COMMENT ON COLUMN system_config_values.created_at IS '覆盖值首次写入时间';
COMMENT ON COLUMN system_config_values.created_by IS '首次写入覆盖值的用户 ID；为空表示请求上下文未提供用户';
COMMENT ON COLUMN system_config_values.updated_at IS '最近一次覆盖值写入时间';
COMMENT ON COLUMN system_config_values.updated_by IS '最近一次写入覆盖值的用户 ID；为空表示请求上下文未提供用户';
