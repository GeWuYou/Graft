-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "scheduler_job_definitions"
  ADD COLUMN IF NOT EXISTS "category" character varying NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS "short_title_key" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "short_title" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "default_enabled" boolean NOT NULL DEFAULT false;

UPDATE "scheduler_job_definitions"
SET "short_title" = CASE WHEN "short_title" = '' THEN "title" ELSE "short_title" END,
    "default_enabled" = CASE WHEN "default_enabled" = false THEN "enabled" ELSE "default_enabled" END
WHERE "deleted_at" IS NULL OR "deleted_at" = 0;

COMMENT ON COLUMN "scheduler_job_definitions"."category" IS 'Job Definition 稳定分类，取值来自 scheduler job category typed contract';
COMMENT ON COLUMN "scheduler_job_definitions"."short_title_key" IS 'Job Definition 短标题国际化键';
COMMENT ON COLUMN "scheduler_job_definitions"."short_title" IS 'Job Definition 默认短标题';
COMMENT ON COLUMN "scheduler_job_definitions"."default_enabled" IS '内置 Scheduled Task 初次种子任务是否默认启用';

DROP INDEX IF EXISTS "scheduler_job_definitions_job_key_key";
DROP INDEX IF EXISTS "scheduler_job_definitions_job_key_live_key";

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_schema = current_schema()
      AND table_name = 'scheduler_job_definitions'
      AND column_name = 'deleted_at'
      AND data_type <> 'bigint'
  ) THEN
    ALTER TABLE "scheduler_job_definitions"
      ALTER COLUMN "deleted_at" DROP DEFAULT,
      ALTER COLUMN "deleted_at" TYPE bigint
        USING CASE
          WHEN "deleted_at" IS NULL THEN 0
          ELSE EXTRACT(EPOCH FROM "deleted_at")::bigint
        END,
      ALTER COLUMN "deleted_at" SET DEFAULT 0,
      ALTER COLUMN "deleted_at" SET NOT NULL;
  ELSE
    ALTER TABLE "scheduler_job_definitions"
      ALTER COLUMN "deleted_at" SET DEFAULT 0,
      ALTER COLUMN "deleted_at" SET NOT NULL;
  END IF;
END $$;

CREATE UNIQUE INDEX IF NOT EXISTS "scheduler_job_definitions_job_key_live_key" ON "scheduler_job_definitions" ("job_key") WHERE "deleted_at" = 0;
CREATE INDEX IF NOT EXISTS "scheduler_job_definitions_category" ON "scheduler_job_definitions" ("category");
CREATE INDEX IF NOT EXISTS "scheduler_job_definitions_deleted_at" ON "scheduler_job_definitions" ("deleted_at");

COMMENT ON COLUMN "scheduler_job_definitions"."deleted_at" IS '软删除时间戳，0 表示未删除，非 0 表示删除发生时的 Unix 秒';
COMMENT ON INDEX "scheduler_job_definitions_job_key_live_key" IS '活跃 Job Definition job_key 唯一索引';

ALTER TABLE "scheduled_tasks"
  ADD COLUMN IF NOT EXISTS "title_key" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "description_key" character varying NOT NULL DEFAULT '';

COMMENT ON COLUMN "scheduled_tasks"."title_key" IS '任务实例标题国际化键';
COMMENT ON COLUMN "scheduled_tasks"."description_key" IS '任务实例说明国际化键';

DROP INDEX IF EXISTS "scheduled_tasks_task_key_key";
DROP INDEX IF EXISTS "scheduled_tasks_task_key_live_key";
DROP INDEX IF EXISTS "scheduled_tasks_title_active_key";
DROP INDEX IF EXISTS "scheduled_tasks_title_live_key";

CREATE UNIQUE INDEX IF NOT EXISTS "scheduled_tasks_task_key_live_key" ON "scheduled_tasks" ("task_key") WHERE "deleted_at" = 0;
CREATE UNIQUE INDEX IF NOT EXISTS "scheduled_tasks_title_live_key" ON "scheduled_tasks" ("title") WHERE "deleted_at" = 0;

COMMENT ON INDEX "scheduled_tasks_task_key_live_key" IS '活跃任务实例 task_key 唯一索引';
COMMENT ON INDEX "scheduled_tasks_title_live_key" IS '活跃任务实例标题唯一索引，软删除后允许复用标题';

ALTER TABLE "scheduler_task_runs"
  ADD COLUMN IF NOT EXISTS "task_title" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "task_title_key" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "job_title" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "job_title_key" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "job_short_title" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "job_short_title_key" character varying NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS "job_category" character varying NOT NULL DEFAULT 'custom',
  ADD COLUMN IF NOT EXISTS "module_key" character varying NOT NULL DEFAULT '';

UPDATE "scheduler_task_runs"
SET "task_title" = CASE WHEN "task_title" = '' THEN "task_name" ELSE "task_title" END,
    "task_title_key" = CASE WHEN "task_title_key" = '' THEN "task_name_key" ELSE "task_title_key" END,
    "job_title" = CASE WHEN "job_title" = '' THEN "task_name" ELSE "job_title" END,
    "job_title_key" = CASE WHEN "job_title_key" = '' THEN "task_name_key" ELSE "job_title_key" END,
    "job_short_title" = CASE WHEN "job_short_title" = '' THEN "task_name" ELSE "job_short_title" END,
    "job_short_title_key" = CASE WHEN "job_short_title_key" = '' THEN "task_name_key" ELSE "job_short_title_key" END,
    "module_key" = CASE
      WHEN "module_key" <> '' THEN "module_key"
      WHEN "module" <> '' THEN "module"
      WHEN "owner" <> '' THEN "owner"
      ELSE "module_key"
    END;

COMMENT ON COLUMN "scheduler_task_runs"."task_title" IS '执行时任务实例标题快照';
COMMENT ON COLUMN "scheduler_task_runs"."task_title_key" IS '执行时任务实例标题国际化键快照';
COMMENT ON COLUMN "scheduler_task_runs"."job_title" IS '执行时 Job Definition 完整标题快照';
COMMENT ON COLUMN "scheduler_task_runs"."job_title_key" IS '执行时 Job Definition 标题国际化键快照';
COMMENT ON COLUMN "scheduler_task_runs"."job_short_title" IS '执行时 Job Definition 短标题快照';
COMMENT ON COLUMN "scheduler_task_runs"."job_short_title_key" IS '执行时 Job Definition 短标题国际化键快照';
COMMENT ON COLUMN "scheduler_task_runs"."job_category" IS '执行时 Job Definition 稳定分类快照';
COMMENT ON COLUMN "scheduler_task_runs"."module_key" IS '执行时 Job Definition 所属模块标识快照';

ALTER TABLE "scheduler_task_runs"
  DROP COLUMN IF EXISTS "task_name",
  DROP COLUMN IF EXISTS "task_name_key",
  DROP COLUMN IF EXISTS "owner",
  DROP COLUMN IF EXISTS "module",
  DROP COLUMN IF EXISTS "task_type",
  DROP COLUMN IF EXISTS "error";
