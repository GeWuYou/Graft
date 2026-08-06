-- atlas:txmode none

ALTER TABLE build_jobs
  ADD COLUMN submission_id VARCHAR(64) NULL;

ALTER TABLE build_jobs
  ALTER COLUMN task_id DROP NOT NULL;

ALTER TABLE build_jobs
  ADD CONSTRAINT build_jobs_submission_id_fkey FOREIGN KEY (submission_id) REFERENCES task_submissions (id) ON DELETE RESTRICT NOT VALID;

ALTER TABLE build_jobs
  ADD CONSTRAINT build_jobs_binding_required CHECK (task_id IS NOT NULL OR submission_id IS NOT NULL) NOT VALID;

CREATE UNIQUE INDEX CONCURRENTLY uq_build_jobs_submission_id ON build_jobs (submission_id) WHERE submission_id IS NOT NULL;

COMMENT ON COLUMN build_jobs.submission_id IS '原子物化前关联的任务提交稳定标识';
