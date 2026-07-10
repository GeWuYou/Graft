DROP TABLE IF EXISTS "refresh_sessions";

ALTER TABLE "users"
  DROP COLUMN IF EXISTS "password_hash",
  DROP COLUMN IF EXISTS "must_change_password",
  DROP COLUMN IF EXISTS "password_changed_at";
