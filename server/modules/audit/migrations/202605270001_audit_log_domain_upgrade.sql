ALTER TABLE "audit_logs" RENAME COLUMN "operator_id" TO "actor_user_id";
ALTER TABLE "audit_logs" RENAME COLUMN "operator_name" TO "actor_display_name";
ALTER TABLE "audit_logs" RENAME COLUMN "error_message" TO "message";
ALTER TABLE "audit_logs" DROP COLUMN "request_method";
ALTER TABLE "audit_logs" DROP COLUMN "request_path";

ALTER TABLE "audit_logs"
  ADD COLUMN "actor_username" character varying NOT NULL DEFAULT '',
  ADD COLUMN "resource_name" character varying NOT NULL DEFAULT '',
  ADD COLUMN "request_id" character varying NOT NULL DEFAULT '',
  ADD COLUMN "metadata" jsonb NOT NULL DEFAULT '{}'::jsonb;

ALTER INDEX "auditlog_operator_id" RENAME TO "auditlog_actor_user_id";

CREATE INDEX "auditlog_request_id" ON "audit_logs" ("request_id");
CREATE INDEX "auditlog_resource_type" ON "audit_logs" ("resource_type");
CREATE INDEX "auditlog_success" ON "audit_logs" ("success");
