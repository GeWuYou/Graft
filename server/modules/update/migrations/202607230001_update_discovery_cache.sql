CREATE TABLE update_discovery_cache (
  cache_key VARCHAR(64) PRIMARY KEY,
  latest_release_json JSONB NULL,
  last_successful_at TIMESTAMPTZ NULL,
  last_attempt_at TIMESTAMPTZ NULL,
  check_error TEXT NULL,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT update_discovery_cache_key_check CHECK (cache_key = 'release_catalog')
);

COMMENT ON TABLE update_discovery_cache IS '平台更新已验证发布目录最近成功快照表';
COMMENT ON COLUMN update_discovery_cache.cache_key IS '固定的发布目录缓存键';
COMMENT ON COLUMN update_discovery_cache.latest_release_json IS '最近成功检查选出的已验证候选发行版投影，不含清单正文';
COMMENT ON COLUMN update_discovery_cache.last_successful_at IS '最近一次成功获取并验证发布目录的时间';
COMMENT ON COLUMN update_discovery_cache.last_attempt_at IS '最近一次检查发布目录的尝试时间';
COMMENT ON COLUMN update_discovery_cache.check_error IS '最近一次检查失败的无秘密摘要';
COMMENT ON COLUMN update_discovery_cache.updated_at IS '缓存快照最后写入时间';
