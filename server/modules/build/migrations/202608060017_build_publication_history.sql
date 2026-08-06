-- Promotion 必须保留同一目的地引用在不同时间指向不同摘要的历史事实。
ALTER TABLE build_publications
  DROP CONSTRAINT uq_build_publications_destination_reference;

CREATE INDEX idx_build_publications_destination_reference
  ON build_publications (connection_ref, repository_ref, mutable_reference, created_at DESC);

COMMENT ON INDEX idx_build_publications_destination_reference IS '按目的地引用查询不可变发布历史';
