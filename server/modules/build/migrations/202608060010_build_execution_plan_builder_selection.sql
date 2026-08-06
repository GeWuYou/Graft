ALTER TABLE build_execution_plans
  ADD COLUMN builder_pool_id VARCHAR(64) NULL,
  ADD COLUMN builder_instance_id VARCHAR(64) NULL;

COMMENT ON COLUMN build_execution_plans.builder_pool_id IS '计划冻结时使用的 Builder Pool 稳定标识，空值表示直接选择 Runtime Target';
COMMENT ON COLUMN build_execution_plans.builder_instance_id IS '计划冻结时实际选中的 Builder Instance 稳定标识，空值表示直接选择 Runtime Target';

CREATE INDEX idx_build_execution_plans_builder_pool_created
  ON build_execution_plans (builder_pool_id, created_at DESC)
  WHERE builder_pool_id IS NOT NULL;
