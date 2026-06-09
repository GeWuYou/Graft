-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "permissions"
  ADD COLUMN IF NOT EXISTS "display_key" character varying NULL,
  ADD COLUMN IF NOT EXISTS "description_key" character varying NULL;

COMMENT ON COLUMN "permissions"."display_key" IS '权限点显示名称本地化 key';
COMMENT ON COLUMN "permissions"."description_key" IS '权限点描述本地化 key';
