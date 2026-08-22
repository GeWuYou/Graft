UPDATE build_jobs
SET workspace_root = ''
WHERE workspace_root <> '';

DELETE FROM build_job_args;

COMMENT ON COLUMN build_jobs.workspace_root IS '历史构建工作区路径脱敏占位；新执行必须在有效任务围栏后瞬时解析且不得写入数据库';
COMMENT ON TABLE build_job_args IS '历史构建参数表；运行期参数不得作为敏感材料持久化，新构建不再写入记录';
COMMENT ON COLUMN build_job_args.value IS '历史参数值字段；迁移后保持无记录且禁止新执行写入敏感材料';
