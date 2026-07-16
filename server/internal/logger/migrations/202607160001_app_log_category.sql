ALTER TABLE "app_logs"
ADD COLUMN "category" varchar(64) NOT NULL DEFAULT 'application';

UPDATE "app_logs"
SET "category" = 'application'
WHERE "category" IS NULL OR "category" = '';

-- 类别筛选按发生时间倒序分页，并以主键保持相同时间记录的稳定顺序。
CREATE INDEX "idx_app_logs_category_occurred_at_id"
ON "app_logs" ("category", "occurred_at" DESC, "id" DESC);

COMMENT ON COLUMN "app_logs"."category" IS '日志类别，取自 logger 注册的运行时诊断类别';
