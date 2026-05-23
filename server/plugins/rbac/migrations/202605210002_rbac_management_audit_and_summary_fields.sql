ALTER TABLE "roles"
  ADD COLUMN IF NOT EXISTS "created_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "updated_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_at" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_by" bigint NOT NULL DEFAULT 0;

ALTER TABLE "permissions"
  ADD COLUMN IF NOT EXISTS "created_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "updated_by" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_at" bigint NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS "deleted_by" bigint NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS "roles_deleted_at_idx" ON "roles" ("deleted_at");
CREATE INDEX IF NOT EXISTS "permissions_deleted_at_idx" ON "permissions" ("deleted_at");
