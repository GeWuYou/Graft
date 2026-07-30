ALTER TABLE update_operations
ADD COLUMN update_mode VARCHAR(32) NOT NULL DEFAULT 'unknown';

ALTER TABLE update_operations
ALTER COLUMN update_mode DROP DEFAULT;

ALTER TABLE update_operations
ADD CONSTRAINT update_operations_update_mode_check
CHECK (update_mode IN ('stable_tracking', 'beta_tracking', 'pinned_stable', 'pinned_beta', 'unknown')) NOT VALID;

ALTER TABLE update_operations
VALIDATE CONSTRAINT update_operations_update_mode_check;

COMMENT ON COLUMN update_operations.update_mode IS '创建更新操作时冻结的部署升级意图；历史记录无法推导时标记为未知';
