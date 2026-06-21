ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS published_by bigint NULL,
  ADD COLUMN IF NOT EXISTS archived_at timestamptz NULL;

COMMENT ON COLUMN announcements.published_by IS '最近一次发布或重新发布公告的管理员用户 ID，历史数据不回填';
COMMENT ON COLUMN announcements.archived_at IS '公告当前归档状态开始时间，仅当状态为 archived 时有业务意义，重新发布时清空';
