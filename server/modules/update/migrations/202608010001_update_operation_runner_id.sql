ALTER TABLE update_operations
ADD COLUMN runner_id VARCHAR(128) NULL;

COMMENT ON COLUMN update_operations.runner_id IS '执行本次更新的 runner 稳定标识，终态历史由状态卷回执确认';
