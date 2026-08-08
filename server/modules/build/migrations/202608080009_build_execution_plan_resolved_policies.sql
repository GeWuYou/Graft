ALTER TABLE build_execution_plans
  ADD COLUMN cache_policy VARCHAR(64) NOT NULL DEFAULT 'disabled',
  ADD COLUMN security_policy VARCHAR(64) NOT NULL DEFAULT 'default';

COMMENT ON COLUMN build_execution_plans.cache_policy IS '执行计划冻结的构建缓存策略';
COMMENT ON COLUMN build_execution_plans.security_policy IS '执行计划冻结的构建安全策略';
