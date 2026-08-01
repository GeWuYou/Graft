-- 保存视图默认状态属于用户与消费列表页面的组合范围，软删除记录不参与默认视图唯一性。
ALTER TABLE saved_views ADD COLUMN is_default BOOLEAN NOT NULL DEFAULT FALSE;

-- 同一用户在同一列表页面至多保留一个有效默认视图，删除记录可保留历史状态。
CREATE UNIQUE INDEX uq_saved_views_owner_surface_default_live
  ON saved_views (owner_user_id, surface_key)
  WHERE deleted_at = 0 AND is_default = TRUE;

COMMENT ON COLUMN saved_views.is_default IS '是否为用户在该列表页面自动应用的默认保存视图，true 表示默认';
