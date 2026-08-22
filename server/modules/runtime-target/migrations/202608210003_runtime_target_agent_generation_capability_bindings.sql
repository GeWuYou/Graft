-- 能力准入绑定到证书世代，禁止通过稳定身份行原地扩权旧证书。
ALTER TABLE runtime_target_agent_capability_bindings
  ADD COLUMN generation_id BIGINT REFERENCES runtime_target_agent_generations(id) ON DELETE RESTRICT;

UPDATE runtime_target_agent_capability_bindings b
SET generation_id = selected.id,
    capability_version = identities.capability_version
FROM runtime_target_agent_identities identities,
LATERAL (
  SELECT g.id
  FROM runtime_target_agent_generations g
  WHERE g.identity_id = identities.id
  ORDER BY CASE WHEN g.status = 'active' THEN 0 ELSE 1 END, g.generation DESC
  LIMIT 1
) selected
WHERE b.identity_id = identities.id;

ALTER TABLE runtime_target_agent_capability_bindings
  ALTER COLUMN generation_id SET NOT NULL,
  DROP CONSTRAINT uq_runtime_target_agent_capability_bindings_identity,
  ADD CONSTRAINT uq_runtime_target_agent_capability_bindings_generation UNIQUE (generation_id);

COMMENT ON COLUMN runtime_target_agent_capability_bindings.generation_id IS '能力绑定所属的不可复用 Agent 信任世代主键';
