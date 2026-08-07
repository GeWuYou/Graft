ALTER TABLE build_builder_reservations
  ADD COLUMN leg_id VARCHAR(64) NOT NULL DEFAULT 'single';

DROP INDEX uq_build_builder_reservations_plan_task_attempt;

CREATE UNIQUE INDEX uq_build_builder_reservations_plan_task_attempt_leg
  ON build_builder_reservations (plan_id, task_id, attempt, leg_id);

COMMENT ON COLUMN build_builder_reservations.leg_id IS '协调构建中占用容量的稳定平台 leg 标识，单平台使用 single';
