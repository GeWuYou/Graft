ALTER TABLE tasks
  ADD COLUMN idempotency_key_hash char(64) NULL,
  ADD COLUMN submission_fingerprint char(64) NULL,
  ADD CONSTRAINT tasks_idempotency_key_hash_sha256_check
    CHECK (idempotency_key_hash IS NULL OR idempotency_key_hash ~ '^[0-9a-f]{64}$'),
  ADD CONSTRAINT tasks_submission_fingerprint_sha256_check
    CHECK (submission_fingerprint IS NULL OR submission_fingerprint ~ '^[0-9a-f]{64}$');

CREATE UNIQUE INDEX uq_tasks_idempotency_submission
  ON tasks (task_type, owner_type, owner_id, COALESCE(created_by, 0), idempotency_key_hash)
  WHERE idempotency_key_hash IS NOT NULL;

COMMENT ON COLUMN tasks.idempotency_key_hash IS '调用方幂等提交键的 SHA-256 十六进制摘要，不保存原始键';
COMMENT ON COLUMN tasks.submission_fingerprint IS '冻结提交内容的规范化 SHA-256 十六进制摘要，用于识别键复用冲突';
