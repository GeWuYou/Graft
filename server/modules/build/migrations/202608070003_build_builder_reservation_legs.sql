-- atlas:txmode none

ALTER TABLE build_builder_reservations
  ADD COLUMN leg_id VARCHAR(64) NOT NULL DEFAULT 'single';

CREATE UNIQUE INDEX CONCURRENTLY uq_build_builder_reservations_plan_task_attempt_leg
  ON build_builder_reservations (plan_id, task_id, attempt, leg_id);

DROP INDEX CONCURRENTLY uq_build_builder_reservations_plan_task_attempt;

COMMENT ON TABLE build_builder_reservations IS 'Build 对 Builder Instance 的容量租约与 fencing 事实，按平台 leg 隔离';
COMMENT ON COLUMN build_builder_reservations.leg_id IS '协调构建中占用容量的稳定平台 leg 标识，单平台使用 single';
