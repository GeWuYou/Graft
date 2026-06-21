ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS published_at timestamptz NULL;

COMMENT ON COLUMN announcements.published_at IS '最近一次发布或重新发布公告的实际动作时间，仅用于审计和后台展示，不参与用户端可见性判断';
