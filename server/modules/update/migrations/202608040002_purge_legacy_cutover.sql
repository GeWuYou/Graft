-- atlas:txmode none

-- 仅删除 beta schema-v1 cutover 显式标记的 Update 事实；其它失败记录仍需保留供审计调查。
DELETE FROM update_failure_diagnostics
WHERE operation_id IN (
  SELECT operation_id
  FROM update_operations
  WHERE failure_code = 'PLATFORM_UPDATE_LEGACY_CUTOVER'
);

DELETE FROM update_operations
WHERE failure_code = 'PLATFORM_UPDATE_LEGACY_CUTOVER';
