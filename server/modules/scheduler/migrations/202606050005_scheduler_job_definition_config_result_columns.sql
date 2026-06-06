ALTER TABLE "scheduler_job_definitions"
  ADD COLUMN IF NOT EXISTS "config_schema" jsonb NOT NULL DEFAULT '{}'::jsonb,
  ADD COLUMN IF NOT EXISTS "default_config" jsonb NOT NULL DEFAULT '{}'::jsonb;

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'scheduler_job_definitions'
      AND column_name = 'params_schema'
  ) THEN
    UPDATE "scheduler_job_definitions"
    SET "config_schema" = "params_schema"
    WHERE "config_schema" = '{}'::jsonb;
  END IF;

  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'scheduler_job_definitions'
      AND column_name = 'default_params'
  ) THEN
    UPDATE "scheduler_job_definitions"
    SET "default_config" = "default_params"
    WHERE "default_config" = '{}'::jsonb;
  END IF;
END $$;

ALTER TABLE "scheduler_job_definitions"
  DROP COLUMN IF EXISTS "params_schema",
  DROP COLUMN IF EXISTS "default_params";

ALTER TABLE "scheduler_task_runs"
  ADD COLUMN IF NOT EXISTS "result_json" jsonb NOT NULL DEFAULT '{}'::jsonb;

COMMENT ON COLUMN "scheduler_job_definitions"."config_schema" IS 'Scheduled Task 配置 JSON Schema';
COMMENT ON COLUMN "scheduler_job_definitions"."default_config" IS '创建 Scheduled Task 时的默认配置 JSON';
COMMENT ON COLUMN "scheduler_task_runs"."result_json" IS '本次执行的结构化 JobRunResult JSON';
