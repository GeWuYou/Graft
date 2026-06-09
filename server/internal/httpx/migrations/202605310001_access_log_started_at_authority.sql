-- Copyright (c) 2025-2026 GeWuYou
-- SPDX-License-Identifier: Apache-2.0

ALTER TABLE "access_logs"
ADD COLUMN "started_at" timestamptz NULL;

UPDATE "access_logs"
SET "started_at" = "occurred_at" - make_interval(secs => GREATEST("duration_ms", 0) / 1000.0)
WHERE "started_at" IS NULL;

ALTER TABLE "access_logs"
ALTER COLUMN "started_at" SET NOT NULL;

CREATE INDEX "idx_access_logs_started_at_id" ON "access_logs" ("started_at" DESC, "id" DESC);
CREATE INDEX "idx_access_logs_route_started_at" ON "access_logs" ("route", "started_at" DESC);
CREATE INDEX "idx_access_logs_user_id_started_at" ON "access_logs" ("user_id", "started_at" DESC);

COMMENT ON COLUMN "access_logs"."started_at" IS '请求开始时间';
