ALTER TABLE build_jobs
  ADD COLUMN workspace_root TEXT NOT NULL DEFAULT '';

COMMENT ON COLUMN build_jobs.workspace_root IS '请求期授权后冻结的构建工作区绝对根路径';
