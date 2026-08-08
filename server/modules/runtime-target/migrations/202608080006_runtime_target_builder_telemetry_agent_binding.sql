ALTER TABLE runtime_target_builder_telemetry_agents
  ADD COLUMN provider_id VARCHAR(128),
  ADD COLUMN builder_scope VARCHAR(255),
  ADD COLUMN capability_profile VARCHAR(255),
  ADD COLUMN capability_version VARCHAR(128),
  ADD COLUMN last_sequence BIGINT NOT NULL DEFAULT 0;

UPDATE runtime_target_builder_telemetry_agents
SET enabled = false
WHERE provider_id IS NULL
   OR builder_scope IS NULL
   OR capability_profile IS NULL
   OR capability_version IS NULL;

ALTER TABLE runtime_target_builder_telemetry_agents
  ADD CONSTRAINT runtime_target_builder_telemetry_agents_sequence_check CHECK (last_sequence >= 0),
  ADD CONSTRAINT runtime_target_builder_telemetry_agents_binding_check CHECK (
    (NOT enabled) OR (
      provider_id = 'docker'
      AND builder_scope IS NOT NULL AND btrim(builder_scope) <> ''
      AND capability_profile IS NOT NULL AND btrim(capability_profile) <> ''
      AND capability_version IS NOT NULL AND btrim(capability_version) <> ''
    )
  );

COMMENT ON COLUMN runtime_target_builder_telemetry_agents.provider_id IS '构建代理绑定的唯一可动态准入运行时提供方标识';
COMMENT ON COLUMN runtime_target_builder_telemetry_agents.builder_scope IS '构建代理受控执行账本对应的构建器范围标识';
COMMENT ON COLUMN runtime_target_builder_telemetry_agents.capability_profile IS '构建代理报告必须匹配的能力配置档案标识';
COMMENT ON COLUMN runtime_target_builder_telemetry_agents.capability_version IS '构建代理报告必须匹配的能力配置档案版本';
COMMENT ON COLUMN runtime_target_builder_telemetry_agents.last_sequence IS '控制平面已接受的该构建代理最大单调报告序列';
