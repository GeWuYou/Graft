ALTER TABLE "users"
  ADD COLUMN IF NOT EXISTS "status" character varying NOT NULL DEFAULT 'enabled',
  ADD COLUMN IF NOT EXISTS "created_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "updated_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_at" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_by" bigint NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS "users_deleted_at_idx" ON "users" ("deleted_at");
