ALTER TABLE compose_projects RENAME TO applications;
ALTER TABLE compose_project_files RENAME TO application_files;
ALTER TABLE compose_project_snapshots RENAME TO application_snapshots;

ALTER TABLE applications RENAME COLUMN id TO application_record_id;
ALTER TABLE applications RENAME COLUMN source_kind TO source_type;
ALTER TABLE application_files RENAME COLUMN project_id TO application_record_id;
ALTER TABLE application_snapshots RENAME COLUMN project_id TO application_record_id;

ALTER TABLE applications
  DROP CONSTRAINT compose_projects_compose_project_name_source_check;

UPDATE applications
SET compose_project_name_source = CASE compose_project_name_source
  WHEN 'declared' THEN 'override'
  WHEN 'generated' THEN 'computed'
  WHEN 'derived' THEN 'computed'
  ELSE compose_project_name_source
END;

ALTER TABLE applications
  ADD COLUMN application_type character varying NOT NULL DEFAULT 'compose',
  DROP COLUMN canonical_project_name,
  DROP COLUMN canonical_project_name_source,
  DROP COLUMN host_scope,
  DROP COLUMN working_directory;

ALTER TABLE applications
  ADD CONSTRAINT applications_application_type_check CHECK (application_type IN ('compose')),
  ADD CONSTRAINT applications_compose_project_name_source_check CHECK (compose_project_name_source IN ('computed', 'override'));

ALTER TABLE applications RENAME CONSTRAINT compose_projects_pkey TO applications_pkey;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_application_id_format_check TO applications_application_id_format_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_application_name_format_check TO applications_application_name_format_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_source_kind_check TO applications_source_type_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_ownership_mode_check TO applications_ownership_mode_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_drift_status_check TO applications_drift_status_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_lifecycle_strategy_kind_check TO applications_lifecycle_strategy_kind_check;
ALTER TABLE applications RENAME CONSTRAINT compose_projects_lifecycle_review_status_check TO applications_lifecycle_review_status_check;

ALTER TABLE application_files RENAME CONSTRAINT compose_project_files_pkey TO application_files_pkey;
ALTER TABLE application_files RENAME CONSTRAINT compose_project_files_project_id_fkey TO application_files_application_record_id_fkey;
ALTER TABLE application_files RENAME CONSTRAINT compose_project_files_kind_check TO application_files_kind_check;
ALTER TABLE application_files RENAME CONSTRAINT compose_project_files_role_check TO application_files_role_check;
ALTER TABLE application_snapshots RENAME CONSTRAINT compose_project_snapshots_pkey TO application_snapshots_pkey;
ALTER TABLE application_snapshots RENAME CONSTRAINT compose_project_snapshots_project_id_fkey TO application_snapshots_application_record_id_fkey;

ALTER INDEX compose_projects_application_id_live RENAME TO applications_application_id_live;
ALTER INDEX compose_projects_runtime_compose_name_live RENAME TO applications_runtime_compose_name_live;
ALTER INDEX compose_projects_workspace_path_live RENAME TO applications_workspace_path_live;
ALTER INDEX compose_projects_application_name_live RENAME TO applications_application_name_live;
ALTER INDEX compose_projects_drift_status_updated RENAME TO applications_drift_status_updated;
ALTER INDEX compose_projects_runtime_target_live RENAME TO applications_runtime_target_live;
ALTER INDEX compose_projects_created_at_live RENAME TO applications_created_at_live;
ALTER INDEX compose_project_files_project_order RENAME TO application_files_record_order;
ALTER INDEX compose_project_files_project_absolute_path RENAME TO application_files_record_absolute_path;
ALTER INDEX compose_project_files_project_kind_role RENAME TO application_files_record_kind_role;

COMMENT ON TABLE applications IS '应用注册主表，当前由 Compose Application 实现拥有';
COMMENT ON COLUMN applications.application_record_id IS '应用内部数值主键，仅供模块持久化和子表外键使用';
COMMENT ON COLUMN applications.application_id IS '应用公开标识，格式为 app_ 加 ULID，供 HTTP 路径和跨模块引用使用';
COMMENT ON COLUMN applications.application_name IS '受管应用的唯一安全名称，外部导入应用允许为空';
COMMENT ON COLUMN applications.application_type IS '应用部署类型，当前固定为 compose';
COMMENT ON COLUMN applications.runtime_target_id IS '关联 Runtime Target 主键，运行目标是运行环境与能力的权威来源';
COMMENT ON COLUMN applications.display_name IS '应用展示名称，不影响 Compose 运行时身份';
COMMENT ON COLUMN applications.compose_project_name IS 'Compose 顶层 name 对应的运行时项目名称';
COMMENT ON COLUMN applications.compose_project_name_source IS 'Compose 项目名称来源，取值为 computed、override';
COMMENT ON COLUMN applications.source_type IS '应用来源类型，取值为 imported、managed、template';
COMMENT ON COLUMN applications.workspace_path IS '应用工作区绝对路径，是应用文件内容的稳定载体';
COMMENT ON COLUMN applications.ownership_mode IS '应用工作区所有权模式，决定注销与销毁保护语义';
COMMENT ON COLUMN applications.source_metadata_json IS '应用来源专属元数据，仅保存无密钥的受控来源标识与派生信息';
COMMENT ON COLUMN applications.lifecycle_strategy_kind IS '应用生命周期执行策略类型，当前固定为 standard';
COMMENT ON COLUMN applications.lifecycle_review_status IS '应用生命周期配置确认状态，取值为 review_required、confirmed';
COMMENT ON COLUMN applications.lifecycle_config_json IS '应用生命周期配置 JSON，保存 Compose 执行策略的可编辑选项';
COMMENT ON COLUMN applications.last_observed_config_hash IS '最近一次观测到的应用配置哈希';
COMMENT ON COLUMN applications.workspace_annotations_json IS '应用工作区文件与目录注释 JSON，键为相对路径';
COMMENT ON COLUMN applications.last_drift_checked_at IS '最近一次执行应用配置漂移检查的时间';
COMMENT ON COLUMN applications.drift_status IS '应用配置漂移状态，取值为 unknown、clean、changed、missing';
COMMENT ON COLUMN applications.created_by IS '创建应用注册记录的操作者用户 ID';
COMMENT ON COLUMN applications.updated_by IS '最近更新应用注册记录的操作者用户 ID';
COMMENT ON COLUMN applications.deleted_by IS '软删除应用注册记录的操作者用户 ID';
COMMENT ON COLUMN applications.created_at IS '应用注册记录创建时间';
COMMENT ON COLUMN applications.updated_at IS '应用注册记录最近更新时间';
COMMENT ON COLUMN applications.deleted_at IS '应用注册记录软删除时间戳，0 表示未删除';

COMMENT ON TABLE application_files IS '应用拥有的 Compose 与环境文件清单表';
COMMENT ON COLUMN application_files.id IS '应用文件记录内部主键';
COMMENT ON COLUMN application_files.application_record_id IS '关联 applications.application_record_id 的内部数值外键';
COMMENT ON COLUMN application_files.kind IS '应用文件类型，取值为 compose、env';
COMMENT ON COLUMN application_files.role IS '应用文件角色，取值为 primary、override、env';
COMMENT ON COLUMN application_files.absolute_path IS '应用文件绝对路径';
COMMENT ON COLUMN application_files.display_path IS '应用工作区内用于展示的相对路径';
COMMENT ON COLUMN application_files.order_index IS 'Compose 或环境文件的有序合并顺序';
COMMENT ON COLUMN application_files.last_observed_hash IS '最近一次观测到的单文件内容哈希';
COMMENT ON COLUMN application_files.created_at IS '应用文件记录创建时间';
COMMENT ON COLUMN application_files.updated_at IS '应用文件记录最近更新时间';

COMMENT ON TABLE application_snapshots IS '应用最近一次成功解析的 Compose 快照表';
COMMENT ON COLUMN application_snapshots.application_record_id IS '关联 applications.application_record_id 的内部数值外键，同时作为快照主键';
COMMENT ON COLUMN application_snapshots.normalized_compose_json IS '最近一次成功解析得到的规范化 Compose JSON 快照';
COMMENT ON COLUMN application_snapshots.config_hash IS '与成功解析快照对应的配置哈希';
COMMENT ON COLUMN application_snapshots.declared_service_count IS '成功解析快照声明的服务数量';
COMMENT ON COLUMN application_snapshots.declared_services_digest IS '成功解析快照声明服务集合摘要';
COMMENT ON COLUMN application_snapshots.refreshed_at IS '成功解析快照生成时间';
