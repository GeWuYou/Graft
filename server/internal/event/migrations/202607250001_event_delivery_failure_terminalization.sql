ALTER TABLE "event_deliveries"
  DROP CONSTRAINT "event_deliveries_status_check";

ALTER TABLE "event_deliveries"
  ADD COLUMN "failed_at" timestamptz NULL,
  ADD CONSTRAINT "event_deliveries_status_check"
    CHECK ("status" IN ('pending', 'processing', 'delivered', 'failed')) NOT VALID;

ALTER TABLE "event_deliveries"
  VALIDATE CONSTRAINT "event_deliveries_status_check";

COMMENT ON COLUMN "event_deliveries"."status" IS '投递状态，取值为 pending、processing、delivered 或 failed；failed 表示达到重试上限或消费者缺失后的终态';
COMMENT ON COLUMN "event_deliveries"."failed_at" IS '投递进入 failed 终态的时间，空值表示尚未终态失败';
