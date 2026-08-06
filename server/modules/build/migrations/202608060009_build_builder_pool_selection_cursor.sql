ALTER TABLE build_builder_pools
  ADD COLUMN selection_cursor BIGINT NOT NULL DEFAULT 0;

COMMENT ON COLUMN build_builder_pools.selection_cursor IS 'Round Robin 选择游标，仅用于事务化分配下一个可执行成员';
