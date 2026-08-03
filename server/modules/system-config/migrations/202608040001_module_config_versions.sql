-- 为模块管理配置增加单调递增版本，以单条条件写入实现跨实例的乐观并发控制。
ALTER TABLE system_config_values
    ALTER COLUMN override_value DROP NOT NULL,
    ADD COLUMN version BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN system_config_values.override_value IS '用户覆盖 JSON；为空表示已重置且保留版本墓碑，模块默认值不会复制到此表';
COMMENT ON COLUMN system_config_values.version IS '模块配置乐观并发版本；零表示从未写入覆盖值，后续每次保存或重置均单调递增';
