CREATE TABLE runtime_target_assignment_revisions (
  runtime_target_id bigint PRIMARY KEY REFERENCES runtime_targets(id),
  revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0),
  created_at timestamptz NOT NULL DEFAULT NOW(),
  updated_at timestamptz NOT NULL DEFAULT NOW()
);
COMMENT ON TABLE runtime_target_assignment_revisions IS '运行目标用户授权集合的乐观并发版本事实表';
COMMENT ON COLUMN runtime_target_assignment_revisions.runtime_target_id IS '运行目标记录主键';
COMMENT ON COLUMN runtime_target_assignment_revisions.revision IS '授权集合单调递增版本号';
COMMENT ON COLUMN runtime_target_assignment_revisions.created_at IS '版本记录创建时间';
COMMENT ON COLUMN runtime_target_assignment_revisions.updated_at IS '版本记录最近更新时间';
