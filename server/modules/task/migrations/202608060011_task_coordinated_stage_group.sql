ALTER TABLE task_stages
  ADD COLUMN coordination_group varchar(128) NOT NULL DEFAULT '',
  ADD COLUMN leg_id varchar(128) NOT NULL DEFAULT '';

COMMENT ON COLUMN task_stages.coordination_group IS '同一协调任务内允许并行领取的 leg 分组标识，空值表示串行阶段';
COMMENT ON COLUMN task_stages.leg_id IS '协调任务内平台 leg 的稳定标识，普通阶段为空';

ALTER TABLE task_stages
  ADD CONSTRAINT task_stages_coordination_group_length CHECK (length(coordination_group) <= 128),
  ADD CONSTRAINT task_stages_leg_id_length CHECK (length(leg_id) <= 128);

CREATE UNIQUE INDEX uq_task_stages_task_leg_id
  ON task_stages (task_id, leg_id)
  WHERE leg_id <> '';
