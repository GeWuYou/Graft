ALTER TABLE update_discovery_cache ADD COLUMN catalog_json JSONB NULL;

COMMENT ON COLUMN update_discovery_cache.catalog_json IS '最近成功检查的完整已验证发行版本目录，不含发布清单正文；历史空值沿用最近候选版本';
