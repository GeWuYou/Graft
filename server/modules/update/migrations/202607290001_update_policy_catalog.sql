ALTER TABLE update_discovery_cache ADD COLUMN catalog_json JSONB NULL;

COMMENT ON COLUMN update_discovery_cache.catalog_json IS '已验证 release catalog 投影，供固定版本策略选择';

ALTER TABLE update_operations ADD COLUMN update_policy VARCHAR(16) NULL;

ALTER TABLE update_operations ADD CONSTRAINT update_operations_policy_check
CHECK (update_policy IS NULL OR update_policy IN ('stable', 'beta', 'fixed', 'manual'));

COMMENT ON COLUMN update_operations.update_policy IS '本次更新执行时由部署 .env 权威确定的策略';
