-- atlas:txmode none

ALTER TABLE update_operations ADD COLUMN request_id VARCHAR(128) NULL;

COMMENT ON COLUMN update_operations.request_id IS '发起更新确认请求的稳定关联标识';

CREATE UNIQUE INDEX CONCURRENTLY uq_update_failure_diagnostics_operation_id
ON update_failure_diagnostics (operation_id)
WHERE operation_id IS NOT NULL;

-- 一个更新操作只保留一条受控终态诊断，便于历史列表和详情稳定定位。
