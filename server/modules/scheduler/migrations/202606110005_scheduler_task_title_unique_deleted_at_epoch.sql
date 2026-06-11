-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'scheduled_tasks'
      AND column_name = 'deleted_at'
      AND data_type <> 'bigint'
  ) THEN
    ALTER TABLE "scheduled_tasks"
      ALTER COLUMN "deleted_at" DROP DEFAULT,
      ALTER COLUMN "deleted_at" TYPE bigint
        USING CASE
          WHEN "deleted_at" IS NULL THEN 0
          ELSE EXTRACT(EPOCH FROM "deleted_at")::bigint
        END,
      ALTER COLUMN "deleted_at" SET DEFAULT 0,
      ALTER COLUMN "deleted_at" SET NOT NULL;
  ELSE
    ALTER TABLE "scheduled_tasks"
      ALTER COLUMN "deleted_at" SET DEFAULT 0,
      ALTER COLUMN "deleted_at" SET NOT NULL;
  END IF;
END $$;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM "scheduled_tasks"
    WHERE "deleted_at" = 0
    GROUP BY "title"
    HAVING COUNT(*) > 1
  ) THEN
    RAISE EXCEPTION 'scheduled_tasks contains duplicate active title values';
  END IF;
END $$;

DROP INDEX IF EXISTS "scheduled_tasks_deleted_at";
CREATE INDEX IF NOT EXISTS "scheduled_tasks_deleted_at" ON "scheduled_tasks" ("deleted_at");
CREATE UNIQUE INDEX IF NOT EXISTS "scheduled_tasks_title_active_key" ON "scheduled_tasks" ("title") WHERE "deleted_at" = 0;

COMMENT ON COLUMN "scheduled_tasks"."deleted_at" IS '软删除时间戳，0 表示未删除，非 0 表示删除发生时的 Unix 秒';
COMMENT ON INDEX "scheduled_tasks_title_active_key" IS '活跃定时任务标题唯一索引，软删除后允许复用标题';
