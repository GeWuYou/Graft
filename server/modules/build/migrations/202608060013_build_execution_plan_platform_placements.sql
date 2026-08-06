ALTER TABLE build_execution_plans
  ADD COLUMN builder_placements_json JSONB NOT NULL DEFAULT '[]'::jsonb;

COMMENT ON COLUMN build_execution_plans.builder_placements_json IS '按目标平台冻结的 Builder Instance、运行目标与调度策略分配证据';
