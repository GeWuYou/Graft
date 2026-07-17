ALTER TABLE application_template_versions
  DROP CONSTRAINT application_template_versions_status_check;

ALTER TABLE application_template_versions
  DROP CONSTRAINT application_template_versions_publish_state_check;

ALTER TABLE application_template_versions
  ADD COLUMN withdrawn_at timestamptz NULL,
  ADD COLUMN withdrawn_by bigint NULL;

ALTER TABLE application_template_versions
  ADD CONSTRAINT application_template_versions_status_check
  CHECK (status IN ('draft', 'published', 'withdrawn'));

ALTER TABLE application_template_versions
  ADD CONSTRAINT application_template_versions_publish_state_check
  CHECK ((status = 'draft' AND published_at IS NULL AND published_by IS NULL) OR (status IN ('published', 'withdrawn') AND published_at IS NOT NULL));

DROP TRIGGER application_template_versions_published_immutable ON application_template_versions;
DROP FUNCTION prevent_published_application_template_version_mutation();

CREATE FUNCTION prevent_immutable_application_template_version_mutation() RETURNS trigger AS $$
BEGIN
  IF TG_OP = 'DELETE' AND OLD.status IN ('published', 'withdrawn') THEN
    RAISE EXCEPTION 'published and withdrawn application template versions are immutable';
  END IF;
  IF TG_OP = 'UPDATE' AND OLD.status = 'withdrawn' THEN
    RAISE EXCEPTION 'withdrawn application template versions are immutable';
  END IF;
  IF TG_OP = 'UPDATE' AND OLD.status = 'published' AND (
    NEW.status <> 'withdrawn'
    OR NEW.template_version_id <> OLD.template_version_id
    OR NEW.template_id <> OLD.template_id
    OR NEW.version_number <> OLD.version_number
    OR NEW.definition_schema_version <> OLD.definition_schema_version
    OR NEW.definition_json IS DISTINCT FROM OLD.definition_json
    OR NEW.published_at IS DISTINCT FROM OLD.published_at
    OR NEW.published_by IS DISTINCT FROM OLD.published_by
    OR NEW.created_by IS DISTINCT FROM OLD.created_by
    OR NEW.created_at IS DISTINCT FROM OLD.created_at
    OR NEW.deleted_at <> OLD.deleted_at
  ) THEN
    RAISE EXCEPTION 'published application template versions may only be withdrawn';
  END IF;
  IF TG_OP = 'DELETE' THEN
    RETURN OLD;
  END IF;
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER application_template_versions_immutable
  BEFORE UPDATE OR DELETE ON application_template_versions
  FOR EACH ROW EXECUTE FUNCTION prevent_immutable_application_template_version_mutation();

UPDATE application_templates
SET deleted_at = EXTRACT(EPOCH FROM CURRENT_TIMESTAMP)::bigint,
    updated_at = CURRENT_TIMESTAMP
WHERE display_name = 'Compose Baseline'
  AND created_by IS NULL
  AND deleted_at = 0;

COMMENT ON COLUMN application_template_versions.status IS '版本状态，draft 可编辑，published 可创建应用，withdrawn 仅保留历史溯源';
COMMENT ON COLUMN application_template_versions.withdrawn_at IS '发布模板被撤回为历史快照的操作时间';
COMMENT ON COLUMN application_template_versions.withdrawn_by IS '撤回发布模板版本的操作者用户 ID';
