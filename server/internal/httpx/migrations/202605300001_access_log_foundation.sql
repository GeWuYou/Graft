CREATE TABLE "access_logs" (
  "id" bigserial PRIMARY KEY,
  "request_id" varchar(64) NOT NULL,
  "method" varchar(16) NOT NULL,
  "path" text NOT NULL,
  "route" text NULL,
  "status_code" integer NOT NULL,
  "duration_ms" bigint NOT NULL,
  "client_ip" varchar(64) NULL,
  "user_agent" text NULL,
  "user_id" bigint NULL,
  "username" varchar(191) NULL,
  "request_size" bigint NULL,
  "response_size" bigint NULL,
  "occurred_at" timestamptz NOT NULL
);

CREATE INDEX "idx_access_logs_occurred_at_id" ON "access_logs" ("occurred_at" DESC, "id" DESC);
CREATE INDEX "idx_access_logs_request_id" ON "access_logs" ("request_id");
CREATE INDEX "idx_access_logs_route_occurred_at" ON "access_logs" ("route", "occurred_at" DESC);
CREATE INDEX "idx_access_logs_user_id_occurred_at" ON "access_logs" ("user_id", "occurred_at" DESC);
