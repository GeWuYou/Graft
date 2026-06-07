ALTER TABLE "scheduled_tasks"
  ADD COLUMN IF NOT EXISTS "config_source" TEXT NOT NULL DEFAULT 'system';

UPDATE "scheduled_tasks"
SET "config_source" = CASE
  WHEN "builtin" = TRUE THEN 'system'
  ELSE 'user'
END
WHERE "config_source" NOT IN ('system', 'user');

COMMENT ON COLUMN "scheduled_tasks"."config_source" IS '任务配置来源：system 表示使用 Job Definition 默认配置，user 表示用户显式覆盖配置';
