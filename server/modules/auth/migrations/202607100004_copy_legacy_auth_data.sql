-- This forward-only migration runs while user remains the live runtime owner.
-- It copies the complete legacy state before the later one-way store switch.
INSERT INTO "auth_credentials" (
  "id",
  "user_id",
  "password_hash",
  "must_change_password",
  "password_changed_at",
  "created_at",
  "updated_at"
)
SELECT
  "id",
  "id",
  "password_hash",
  "must_change_password",
  "password_changed_at",
  "created_at",
  "updated_at"
FROM "users";

INSERT INTO "auth_refresh_sessions" (
  "id",
  "user_id",
  "token_id",
  "expires_at",
  "revoked_at",
  "replaced_by_token_id",
  "created_at",
  "updated_at"
)
SELECT
  "id",
  "user_id",
  "token_id",
  "expires_at",
  "revoked_at",
  "replaced_by_token_id",
  "created_at",
  "updated_at"
FROM "refresh_sessions";

-- Explicit legacy IDs require sequence alignment before auth-owned stores write rows.
SELECT setval(
  pg_get_serial_sequence('auth_credentials', 'id'),
  COALESCE((SELECT MAX("id") FROM "auth_credentials"), 1),
  EXISTS (SELECT 1 FROM "auth_credentials")
);

SELECT setval(
  pg_get_serial_sequence('auth_refresh_sessions', 'id'),
  COALESCE((SELECT MAX("id") FROM "auth_refresh_sessions"), 1),
  EXISTS (SELECT 1 FROM "auth_refresh_sessions")
);
