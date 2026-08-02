ALTER TABLE update_operations
ADD COLUMN recovery_claim_id VARCHAR(128) NULL,
ADD COLUMN recovery_claimed_at TIMESTAMPTZ NULL;

COMMENT ON COLUMN update_operations.recovery_claim_id IS '恢复启动协调认领标识，不表示 runner 生命周期阶段';
COMMENT ON COLUMN update_operations.recovery_claimed_at IS '恢复启动协调认领的服务端记录时间';
