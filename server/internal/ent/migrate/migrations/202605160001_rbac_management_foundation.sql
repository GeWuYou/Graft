ALTER TABLE "roles"
  ADD COLUMN "builtin" boolean NOT NULL DEFAULT false;

ALTER TABLE "permissions"
  ADD COLUMN "category" character varying NOT NULL DEFAULT 'api';
