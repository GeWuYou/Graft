-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "notification_events"
  ADD COLUMN IF NOT EXISTS "resource_type_key" character varying NOT NULL DEFAULT '';

COMMENT ON COLUMN "notification_events"."resource_type_key" IS '通知资源类型展示国际化键';
