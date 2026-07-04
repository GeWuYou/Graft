ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS lifecycle_strategy_kind character varying NOT NULL DEFAULT 'standard',
  ADD COLUMN IF NOT EXISTS lifecycle_review_status character varying NOT NULL DEFAULT 'review_required',
  ADD COLUMN IF NOT EXISTS lifecycle_config_json jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER TABLE compose_projects
  DROP CONSTRAINT IF EXISTS compose_projects_source_kind_check;

ALTER TABLE compose_projects
  ADD CONSTRAINT compose_projects_source_kind_check CHECK (source_kind IN ('imported', 'managed', 'git', 'template', 'remote-host'));

ALTER TABLE compose_projects
  DROP CONSTRAINT IF EXISTS compose_projects_lifecycle_strategy_kind_check;

ALTER TABLE compose_projects
  ADD CONSTRAINT compose_projects_lifecycle_strategy_kind_check CHECK (lifecycle_strategy_kind IN ('standard'));

ALTER TABLE compose_projects
  DROP CONSTRAINT IF EXISTS compose_projects_lifecycle_review_status_check;

ALTER TABLE compose_projects
  ADD CONSTRAINT compose_projects_lifecycle_review_status_check CHECK (lifecycle_review_status IN ('review_required', 'confirmed'));

COMMENT ON COLUMN compose_projects.lifecycle_strategy_kind IS '项目生命周期执行策略类型 typed contract，当前固定为 standard';
COMMENT ON COLUMN compose_projects.lifecycle_review_status IS '项目生命周期配置确认状态 typed contract，取值为 review_required、confirmed';
COMMENT ON COLUMN compose_projects.lifecycle_config_json IS '项目生命周期配置 JSON，存储 standard compose 执行策略的可编辑选项';
