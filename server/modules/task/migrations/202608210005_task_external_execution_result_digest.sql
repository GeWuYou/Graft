ALTER TABLE task_external_execution_leases
  ADD COLUMN result_sha256 char(64) NULL,
  ADD CONSTRAINT task_external_execution_leases_result_digest
    CHECK (result_sha256 IS NULL OR result_sha256 ~ '^[0-9a-f]{64}$');

COMMENT ON COLUMN task_external_execution_leases.result_sha256 IS '当前围栏已接受的瞬时领域结果协议与载荷摘要，不保存结果原文';
