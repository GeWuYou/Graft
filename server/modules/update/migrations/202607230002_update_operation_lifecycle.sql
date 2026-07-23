ALTER TABLE update_operations DROP CONSTRAINT update_operations_status_check;

ALTER TABLE update_operations ADD CONSTRAINT update_operations_status_check CHECK (status IN ('PLANNING', 'BACKING_UP', 'PULLING', 'MIGRATING', 'RECREATING', 'VERIFYING', 'SUCCESS', 'FAILED', 'RECOVERED', 'NEEDS_ATTENTION'));

COMMENT ON CONSTRAINT update_operations_status_check ON update_operations IS '更新编排生命周期与终态枚举约束';
