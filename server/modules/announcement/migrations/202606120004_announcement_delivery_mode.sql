-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE announcements
  ADD COLUMN IF NOT EXISTS delivery_mode character varying NOT NULL DEFAULT 'silent';

COMMENT ON COLUMN announcements.delivery_mode IS '公告展示策略 typed contract，取值为 silent、popup，silent 仅进入公告中心，popup 会对未读用户弹窗提醒';
