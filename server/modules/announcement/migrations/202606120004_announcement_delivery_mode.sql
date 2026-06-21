ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS delivery_mode character varying NOT NULL DEFAULT 'silent';

ALTER TABLE announcements
  ADD CONSTRAINT announcements_delivery_mode_check CHECK (delivery_mode IN ('silent', 'popup'));

COMMENT ON COLUMN announcements.delivery_mode IS '公告展示策略 typed contract，取值为 silent、popup，silent 仅进入公告中心，popup 会对未读用户弹窗提醒';
