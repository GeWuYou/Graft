ALTER TABLE tasks
  ADD COLUMN activation_required boolean NOT NULL DEFAULT FALSE;

COMMENT ON COLUMN tasks.activation_required IS 'Task 是否等待消费模块完成快照持久化后才允许 worker 领取';
