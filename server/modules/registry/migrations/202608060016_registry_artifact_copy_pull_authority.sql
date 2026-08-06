ALTER TABLE artifact_repositories
  ADD COLUMN allow_pull BOOLEAN NOT NULL DEFAULT true;

COMMENT ON COLUMN artifact_repositories.allow_pull IS '是否允许授权用户将该仓库作为摘要寻址产物复制来源，true 表示允许读取，false 表示禁止作为复制源';
