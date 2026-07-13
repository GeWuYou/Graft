CREATE UNIQUE INDEX uq_runtime_targets_live_display_name_normalized
ON runtime_targets (lower(btrim(display_name)))
WHERE deleted_at = 0;
COMMENT ON INDEX uq_runtime_targets_live_display_name_normalized IS '存活运行目标展示名称去空白后不区分大小写唯一索引';
