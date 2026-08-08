ALTER TABLE build_builder_reservations
  ADD COLUMN capacity_units INTEGER NOT NULL DEFAULT 1,
  ADD COLUMN slot_budget INTEGER NOT NULL DEFAULT 1,
  ADD CONSTRAINT build_builder_reservations_capacity_units_positive CHECK (capacity_units > 0),
  ADD CONSTRAINT build_builder_reservations_slot_budget_positive CHECK (slot_budget > 0),
  ADD CONSTRAINT build_builder_reservations_capacity_within_slot_budget CHECK (capacity_units <= slot_budget);

DROP INDEX uq_build_builder_reservations_live_instance;

CREATE INDEX idx_build_builder_reservations_live_instance
  ON build_builder_reservations (builder_instance_id)
  WHERE state IN ('reserved', 'accepted', 'running');

COMMENT ON COLUMN build_builder_reservations.capacity_units IS '该租约占用的 Builder capacity unit 数；当前每个执行 leg 占用一个单位';
COMMENT ON COLUMN build_builder_reservations.slot_budget IS '创建 Placement 时由 Build 冻结的 Builder 可用 slot 上限；不得从 Runtime Target 默认推导';
