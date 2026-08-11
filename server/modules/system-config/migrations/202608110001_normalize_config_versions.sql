-- 将版本迁移前已有的覆盖行升级为首个有效版本，避免首次保存错误地走创建分支。
UPDATE system_config_values
SET version = 1
WHERE version = 0;

ALTER TABLE system_config_values
    ALTER COLUMN version SET DEFAULT 1;

COMMENT ON TABLE system_config_values IS '系统配置覆盖值及其乐观并发版本记录';
COMMENT ON COLUMN system_config_values.version IS '系统配置覆盖记录的乐观并发版本；首个持久化版本为一，后续每次保存或重置单调递增';
