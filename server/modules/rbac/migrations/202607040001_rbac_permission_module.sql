ALTER TABLE "permissions"
  RENAME COLUMN "category" TO "module";

ALTER TABLE "permissions"
  ALTER COLUMN "module" SET DEFAULT '';

COMMENT ON COLUMN "permissions"."module" IS '权限归属模块标识，例如 user、rbac、core.httpx';
