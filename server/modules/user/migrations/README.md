This directory is the module-owned migration boundary for `server/modules/user`.

`users` and `refresh_sessions` rebuild from this directory as a clean empty-database baseline.
The historical shared chain under `server/internal/ent/migrate/migrations` remains available only
for explicit/manual runs.

`202605190001_user_module_schema.sql` is the canonical user-module baseline on the default
migration path. It already contains the current table structure, indexes, defaults, and comments,
so no plugin-boundary checkpoint or follow-up comment/status migrations remain in this directory.
