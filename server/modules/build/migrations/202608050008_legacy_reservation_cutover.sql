-- Legacy cutover is intentionally evidence based: the activation flag alone never creates a ready Task.
-- Rows present at this migration are treated as the legacy grace window having elapsed.

ALTER TABLE build_jobs VALIDATE CONSTRAINT build_jobs_submission_id_fkey;
ALTER TABLE build_jobs VALIDATE CONSTRAINT build_jobs_binding_required;

INSERT INTO task_submissions (
  id, task_type, owner_type, owner_id, requested_by, state, submission_version,
  lease_ttl_ms, lease_renewable, lease_token_hash, lease_expires_at,
  absolute_deadline_at, prerequisite_kind, prerequisite_ref, task_id,
  terminal_reason, created_at, updated_at, activated_at, terminal_at
)
SELECT
  'legacy-task-' || task.id::text,
  task.task_type,
  task.owner_type,
  task.owner_id,
  task.created_by,
  CASE WHEN job.task_id IS NULL THEN 'expired' ELSE 'activated' END,
  1,
  600000,
  FALSE,
  repeat('0', 64),
  NOW(),
  NOW() + INTERVAL '1 second',
  CASE WHEN job.task_id IS NULL THEN 'legacy.activation_required' ELSE 'build.snapshot.v1' END,
  job.build_id,
  CASE WHEN job.task_id IS NULL THEN NULL ELSE task.id END,
  CASE WHEN job.task_id IS NULL THEN 'legacy_snapshot_missing' ELSE NULL END,
  task.created_at,
  NOW(),
  CASE WHEN job.task_id IS NULL THEN NULL ELSE NOW() END,
  CASE WHEN job.task_id IS NULL THEN NOW() ELSE NULL END
FROM tasks AS task
LEFT JOIN build_jobs AS job ON job.task_id = task.id
WHERE task.activation_required = TRUE
ON CONFLICT (id) DO NOTHING;

UPDATE tasks AS task
SET activation_required = FALSE,
    status = 'ready',
    updated_at = NOW()
FROM build_jobs AS job
WHERE task.id = job.task_id
  AND task.activation_required = TRUE
  AND task.status IN ('pending', 'ready', 'scheduled', 'needs_attention');

UPDATE tasks AS task
SET activation_required = FALSE,
    status = 'cancelled',
    finished_at = COALESCE(task.finished_at, NOW()),
    updated_at = NOW()
WHERE task.activation_required = TRUE
  AND task.status IN ('pending', 'ready', 'scheduled', 'needs_attention')
  AND NOT EXISTS (SELECT 1 FROM build_jobs AS job WHERE job.task_id = task.id);

COMMENT ON COLUMN tasks.activation_required IS '迁移期历史兼容字段；legacy cutover 后新 Task 不使用该字段';
