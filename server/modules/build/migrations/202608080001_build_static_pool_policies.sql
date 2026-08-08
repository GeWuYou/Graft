ALTER TABLE build_builder_pools
  DROP CONSTRAINT build_builder_pools_policy_check,
  ADD CONSTRAINT build_builder_pools_policy_check CHECK (scheduling_policy IN ('manual', 'round_robin', 'random', 'least_load', 'labels', 'affinity', 'region'));

COMMENT ON COLUMN build_builder_pools.scheduling_policy IS 'Pool 策略记录；新写入仅允许 manual、round_robin 或 random，历史动态策略仅可读取';
COMMENT ON COLUMN build_builder_pools.selector_json IS 'Pool 静态资格选择条件，包含实例标签或 manual 策略的 instance_id';
