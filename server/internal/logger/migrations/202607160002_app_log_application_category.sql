ALTER TABLE "app_logs"
ALTER COLUMN "category" SET DEFAULT 'application';

COMMENT ON COLUMN "app_logs"."category" IS '日志类别，取自 logger 注册的运行时诊断类别；默认值为 application 通用应用日志';
