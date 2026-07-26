ALTER TABLE backups ADD COLUMN task_id BIGINT NULL;

UPDATE backups
SET task_id = handoff.task_id
FROM backup_runner_handoffs AS handoff
WHERE handoff.backup_id = backups.id
	AND handoff.status = 'COMPLETED'
	AND handoff.backup_id IS NOT NULL
  AND backups.task_id IS NULL;

-- 一项 Task 只能形成一条不可变的备份事实，重复记录必须读取既有事实。
ALTER TABLE backups
  ADD CONSTRAINT backups_task_id_fkey FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE RESTRICT,
  ADD CONSTRAINT uq_backups_task_id UNIQUE (task_id);

COMMENT ON COLUMN backups.task_id IS '创建该备份事实的 Task Runtime 任务主键，历史未关联记录为空';
