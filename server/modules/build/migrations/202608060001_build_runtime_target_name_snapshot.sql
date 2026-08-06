ALTER TABLE build_jobs
  ADD COLUMN runtime_target_name VARCHAR(255) NOT NULL DEFAULT '';

COMMENT ON COLUMN build_jobs.runtime_target_name IS '提交构建时冻结的运行目标显示名快照';
