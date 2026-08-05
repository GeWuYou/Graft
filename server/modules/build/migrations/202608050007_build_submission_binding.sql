ALTER TABLE build_jobs
  ADD COLUMN submission_id VARCHAR(64) NULL;

ALTER TABLE build_jobs
  ALTER COLUMN task_id DROP NOT NULL;

ALTER TABLE build_jobs
  ADD CONSTRAINT build_jobs_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES task_submissions (id) ON DELETE RESTRICT;

CREATE UNIQUE INDEX uq_build_jobs_submission_id ON build_jobs (submission_id) WHERE submission_id IS NOT NULL;

COMMENT ON COLUMN build_jobs.submission_id IS '原子物化前关联的任务提交稳定标识';
