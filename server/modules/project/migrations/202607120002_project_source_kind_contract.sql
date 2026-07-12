UPDATE compose_projects
SET source_kind = CASE source_kind
  WHEN 'git' THEN 'managed'
  WHEN 'remote-host' THEN 'imported'
  ELSE source_kind
END
WHERE source_kind IN ('git', 'remote-host');

ALTER TABLE compose_projects
  DROP CONSTRAINT IF EXISTS compose_projects_source_kind_check;

ALTER TABLE compose_projects
  ADD CONSTRAINT compose_projects_source_kind_check CHECK (source_kind IN ('imported', 'managed', 'template'));

COMMENT ON COLUMN compose_projects.source_kind IS '项目来源 typed contract，取值为 imported、managed、template';
