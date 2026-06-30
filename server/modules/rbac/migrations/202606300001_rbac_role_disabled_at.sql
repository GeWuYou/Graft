ALTER TABLE "roles"
  ADD COLUMN "disabled_at" bigint NOT NULL DEFAULT 0;

CREATE INDEX "roles_disabled_at_idx" ON "roles" ("disabled_at");

COMMENT ON COLUMN "roles"."disabled_at" IS '禁用时间戳，0 表示启用';
