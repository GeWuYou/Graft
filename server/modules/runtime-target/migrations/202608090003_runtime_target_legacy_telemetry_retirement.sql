DROP INDEX IF EXISTS idx_runtime_target_agent_generations_active_lookup;

ALTER TABLE runtime_target_builder_telemetry_agents
  ADD CONSTRAINT runtime_target_builder_telemetry_agents_legacy_disabled_check
  CHECK (enabled = false);
