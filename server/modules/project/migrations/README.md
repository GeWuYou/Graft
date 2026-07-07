# Project Migrations

This directory contains module-owned SQL migrations for Compose Project Management.

Whenever a `.sql` file in this directory is added, removed, or edited, update the migration checksum in the same
change:

```bash
cd server/modules/project/migrations && atlas migrate hash --dir file://.
cd server && go generate ./internal/moduleregistry
```

`atlas.sum` and `server/internal/moduleregistry/generated.go` are required derived artifacts for this directory.
Leaving either stale is a task bug and will surface later as `validate migration dir modules/project/migrations:
checksum mismatch`.
