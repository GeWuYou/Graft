CREATE UNIQUE INDEX uq_runtime_target_builder_telemetry_agents_active_target
  ON runtime_target_builder_telemetry_agents (runtime_target_id)
  WHERE enabled = true;

COMMENT ON INDEX uq_runtime_target_builder_telemetry_agents_active_target IS '每个运行目标只允许一个可执行的已启用 Docker 构建代理范围';
