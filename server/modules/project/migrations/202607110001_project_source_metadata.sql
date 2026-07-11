ALTER TABLE compose_projects
  ADD COLUMN IF NOT EXISTS source_metadata_json jsonb NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN compose_projects.source_metadata_json IS '项目来源专属元数据，仅保存无密钥的受控来源标识与工作区派生信息';
