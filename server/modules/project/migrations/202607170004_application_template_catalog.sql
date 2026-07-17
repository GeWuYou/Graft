ALTER TABLE application_templates
  ADD COLUMN category character varying;

UPDATE application_templates
SET category = 'other'
WHERE category IS NULL;

ALTER TABLE application_templates
  ALTER COLUMN category SET DEFAULT 'other',
  ALTER COLUMN category SET NOT NULL,
  ADD CONSTRAINT application_templates_category_check CHECK (category IN ('database', 'cache', 'mq', 'proxy', 'storage', 'monitoring', 'logging', 'cicd', 'ai', 'other'));

CREATE INDEX application_templates_catalog_live
  ON application_templates (deployment_adapter_kind, category, updated_at DESC, template_id DESC)
  WHERE deleted_at = 0 AND archived_at IS NULL;

COMMENT ON COLUMN application_templates.category IS '模板目录受控分类，用于创建者发现与筛选';
COMMENT ON INDEX application_templates_catalog_live IS '按部署适配器、分类和最近更新时间读取可用模板目录的索引';
