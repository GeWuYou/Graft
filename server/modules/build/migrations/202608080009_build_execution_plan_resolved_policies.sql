-- 验证前序容量约束时不阻塞 reservation 写入；约束在创建后已持续保护新数据。
ALTER TABLE build_builder_reservations
  VALIDATE CONSTRAINT build_builder_reservations_capacity_units_positive,
  VALIDATE CONSTRAINT build_builder_reservations_slot_budget_positive,
  VALIDATE CONSTRAINT build_builder_reservations_capacity_within_slot_budget;

ALTER TABLE build_execution_plans
  ADD COLUMN cache_policy VARCHAR(64) NOT NULL DEFAULT 'disabled',
  ADD COLUMN security_policy VARCHAR(64) NOT NULL DEFAULT 'default';

COMMENT ON COLUMN build_execution_plans.cache_policy IS '执行计划冻结的构建缓存策略';
COMMENT ON COLUMN build_execution_plans.security_policy IS '执行计划冻结的构建安全策略';
