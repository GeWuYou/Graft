UPDATE tasks
SET owner_type = 'application',
    task_type = regexp_replace(task_type, '^project\.compose\.', 'application.compose.'),
    input_json = CASE
      WHEN input_json ? 'project_id' THEN (input_json - 'project_id') || jsonb_build_object('application_id', input_json -> 'project_id')
      ELSE input_json
    END,
    metadata_json = CASE
      WHEN metadata_json ? 'project_id' THEN (metadata_json - 'project_id') || jsonb_build_object('application_id', metadata_json -> 'project_id')
      ELSE metadata_json
    END,
    plan_json = replace(
      replace(plan_json::text, '"project_id":', '"application_id":'),
      '"working_directory":',
      '"workspace_path":'
    )::jsonb,
    updated_at = CURRENT_TIMESTAMP
WHERE owner_type = 'compose_project'
   OR task_type LIKE 'project.compose.%';

UPDATE task_stages
SET executor_type = regexp_replace(executor_type, '^project\.compose\.', 'application.compose.'),
    input_json = CASE
      WHEN input_json ? 'working_directory' THEN (input_json - 'working_directory') || jsonb_build_object('workspace_path', input_json -> 'working_directory')
      ELSE input_json
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE executor_type LIKE 'project.compose.%';

COMMENT ON COLUMN tasks.task_type IS '业务模块定义的稳定任务类型，Compose 应用任务使用 application.compose 前缀';
COMMENT ON COLUMN tasks.owner_type IS '任务所属业务资源类型，Compose 应用使用 application';
COMMENT ON COLUMN tasks.owner_id IS '任务所属业务资源公开稳定标识，应用资源使用 app_ 加 ULID';
COMMENT ON COLUMN tasks.input_json IS '任务提交时冻结的业务输入 JSON，应用引用使用 application_id';
COMMENT ON COLUMN tasks.metadata_json IS '可展示且已脱敏的任务元数据 JSON，应用引用使用 application_id';
COMMENT ON COLUMN tasks.plan_json IS '提交时冻结的串行任务计划 JSON，Compose 应用阶段输入使用 workspace_path';
COMMENT ON COLUMN task_stages.executor_type IS '业务模块注册的阶段执行器类型，Compose 应用执行器使用 application.compose 前缀';
COMMENT ON COLUMN task_stages.input_json IS '冻结的阶段执行器输入 JSON，Compose 应用工作区使用 workspace_path';
