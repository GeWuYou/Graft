CREATE TABLE application_templates (
  template_id character varying PRIMARY KEY,
  display_name character varying NOT NULL,
  description text NOT NULL DEFAULT '',
  deployment_adapter_kind character varying NOT NULL,
  archived_at timestamptz NULL,
  created_by bigint NULL,
  updated_by bigint NULL,
  deleted_by bigint NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at bigint NOT NULL DEFAULT 0,
  CONSTRAINT application_templates_id_format_check CHECK (template_id ~ '^tpl_[0-9A-HJKMNP-TV-Z]{26}$'),
  CONSTRAINT application_templates_adapter_kind_check CHECK (deployment_adapter_kind IN ('compose'))
);

CREATE UNIQUE INDEX application_templates_display_name_live
  ON application_templates (display_name)
  WHERE deleted_at = 0;

CREATE INDEX application_templates_adapter_live
  ON application_templates (deployment_adapter_kind, updated_at DESC)
  WHERE deleted_at = 0 AND archived_at IS NULL;

CREATE TABLE application_template_versions (
  template_version_id character varying PRIMARY KEY,
  template_id character varying NOT NULL REFERENCES application_templates(template_id),
  version_number integer NOT NULL,
  status character varying NOT NULL,
  definition_schema_version integer NOT NULL,
  definition_json jsonb NOT NULL,
  published_at timestamptz NULL,
  published_by bigint NULL,
  created_by bigint NULL,
  updated_by bigint NULL,
  created_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at timestamptz NOT NULL DEFAULT CURRENT_TIMESTAMP,
  deleted_at bigint NOT NULL DEFAULT 0,
  CONSTRAINT application_template_versions_id_format_check CHECK (template_version_id ~ '^tplv_[0-9A-HJKMNP-TV-Z]{26}$'),
  CONSTRAINT application_template_versions_number_check CHECK (version_number > 0),
  CONSTRAINT application_template_versions_status_check CHECK (status IN ('draft', 'published')),
  CONSTRAINT application_template_versions_schema_check CHECK (definition_schema_version > 0),
  CONSTRAINT application_template_versions_publish_state_check CHECK ((status = 'draft' AND published_at IS NULL AND published_by IS NULL) OR (status = 'published' AND published_at IS NOT NULL))
);

CREATE UNIQUE INDEX application_template_versions_template_number_live
  ON application_template_versions (template_id, version_number)
  WHERE deleted_at = 0;

CREATE UNIQUE INDEX application_template_versions_one_draft_live
  ON application_template_versions (template_id)
  WHERE deleted_at = 0 AND status = 'draft';

CREATE INDEX application_template_versions_published_live
  ON application_template_versions (template_id, version_number DESC)
  WHERE deleted_at = 0 AND status = 'published';

CREATE FUNCTION prevent_published_application_template_version_mutation() RETURNS trigger AS $$
BEGIN
  IF OLD.status = 'published' THEN
    RAISE EXCEPTION 'published application template versions are immutable';
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER application_template_versions_published_immutable
  BEFORE UPDATE OR DELETE ON application_template_versions
  FOR EACH ROW EXECUTE FUNCTION prevent_published_application_template_version_mutation();

COMMENT ON TABLE application_templates IS '应用模板身份表，保存可复用创建蓝图的通用展示元数据';
COMMENT ON COLUMN application_templates.template_id IS '模板稳定公开标识，格式为 tpl_ 加 ULID，创建后不可变';
COMMENT ON COLUMN application_templates.display_name IS '模板展示名称，live 模板内唯一';
COMMENT ON COLUMN application_templates.description IS '模板面向操作者的说明，不保存密钥或变量值';
COMMENT ON COLUMN application_templates.deployment_adapter_kind IS '模板定义适配器类型，决定 definition_json 的解释与校验边界';
COMMENT ON COLUMN application_templates.archived_at IS '模板归档时间，归档后不得作为新应用创建来源';
COMMENT ON COLUMN application_templates.created_by IS '创建模板身份的操作者用户 ID';
COMMENT ON COLUMN application_templates.updated_by IS '最近更新模板元数据的操作者用户 ID';
COMMENT ON COLUMN application_templates.deleted_by IS '软删除模板的操作者用户 ID';
COMMENT ON COLUMN application_templates.created_at IS '模板身份创建时间';
COMMENT ON COLUMN application_templates.updated_at IS '模板身份最近更新时间';
COMMENT ON COLUMN application_templates.deleted_at IS '模板软删除时间戳，0 表示未删除';
COMMENT ON INDEX application_templates_display_name_live IS 'live 模板展示名称唯一索引';
COMMENT ON INDEX application_templates_adapter_live IS '按部署适配器查询未归档模板的索引';

COMMENT ON TABLE application_template_versions IS '应用模板版本表，草稿可编辑，发布版本为不可变定义快照';
COMMENT ON COLUMN application_template_versions.template_version_id IS '模板版本稳定公开标识，格式为 tplv_ 加 ULID，创建后不可变';
COMMENT ON COLUMN application_template_versions.template_id IS '所属应用模板稳定公开标识';
COMMENT ON COLUMN application_template_versions.version_number IS '同一模板内递增的业务版本号';
COMMENT ON COLUMN application_template_versions.status IS '版本状态，draft 可编辑，published 仅可读取和实例化';
COMMENT ON COLUMN application_template_versions.definition_schema_version IS '适配器定义快照的结构版本号';
COMMENT ON COLUMN application_template_versions.definition_json IS '由部署适配器拥有的类型化定义快照，不保存 Provider 专属列';
COMMENT ON COLUMN application_template_versions.published_at IS '版本发布为不可变快照的时间';
COMMENT ON COLUMN application_template_versions.published_by IS '发布模板版本的操作者用户 ID';
COMMENT ON COLUMN application_template_versions.created_by IS '创建模板版本的操作者用户 ID';
COMMENT ON COLUMN application_template_versions.updated_by IS '最近编辑草稿的操作者用户 ID';
COMMENT ON COLUMN application_template_versions.created_at IS '模板版本创建时间';
COMMENT ON COLUMN application_template_versions.updated_at IS '模板版本最近更新时间';
COMMENT ON COLUMN application_template_versions.deleted_at IS '模板版本软删除时间戳，0 表示未删除';
COMMENT ON INDEX application_template_versions_template_number_live IS '模板内 live 业务版本号唯一索引';
COMMENT ON INDEX application_template_versions_one_draft_live IS '每个模板最多一个可编辑草稿的唯一索引';
COMMENT ON INDEX application_template_versions_published_live IS '读取模板已发布版本目录的索引';
