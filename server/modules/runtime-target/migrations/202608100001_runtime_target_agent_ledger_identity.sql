-- mTLS Agent generation 已取代历史 Ed25519 telemetry 身份；执行账本不能继续依赖已退役的表。
ALTER TABLE runtime_target_builder_execution_ledgers
  DROP CONSTRAINT IF EXISTS fk_runtime_target_builder_execution_ledgers_agent;
