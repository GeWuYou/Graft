ALTER TABLE build_jobs RENAME COLUMN application_id TO application_record_id;
ALTER TABLE build_jobs ADD COLUMN application_id VARCHAR(30);

-- 历史作业只保存内部应用记录主键；公开标识必须由 Project 应用注册表回填，不能临时生成。
UPDATE build_jobs AS job
SET application_id = application.application_id
FROM applications AS application
WHERE application.application_record_id = job.application_record_id;

ALTER TABLE build_jobs ALTER COLUMN application_id SET NOT NULL;

ALTER INDEX idx_build_jobs_application_created RENAME TO idx_build_jobs_application_record_created;
CREATE INDEX idx_build_jobs_public_application_created ON build_jobs (application_id, created_at DESC, id DESC);

COMMENT ON COLUMN build_jobs.application_record_id IS '来源应用的私有记录主键快照';
COMMENT ON COLUMN build_jobs.application_id IS '来源应用的公开应用标识快照';
