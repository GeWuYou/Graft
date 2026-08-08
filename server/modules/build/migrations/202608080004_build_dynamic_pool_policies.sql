ALTER TABLE build_builder_pools
  DROP CONSTRAINT build_builder_pools_policy_check,
  ADD CONSTRAINT build_builder_pools_policy_check CHECK (scheduling_policy IN ('manual', 'round_robin', 'random', 'least_load', 'capacity', 'affinity', 'labels', 'region'));

COMMENT ON COLUMN build_builder_pools.scheduling_policy IS 'Pool 策略记录；新写入允许 manual、round_robin、random、least_load、capacity 或 affinity，region 与 labels 仅保留历史读取';
COMMENT ON COLUMN build_builder_pools.selector_json IS 'Pool 静态资格选择条件；affinity 策略使用 affinity_key 与受信遥测中的亲和标识匹配';
