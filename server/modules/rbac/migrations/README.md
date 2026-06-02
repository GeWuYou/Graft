This directory is the module-owned migration boundary for `server/modules/rbac`.

`roles`, `permissions`, `role_permissions`, and RBAC-owned `user_roles` rebuild from this
directory as a clean empty-database baseline. The historical shared chain under
`server/internal/ent/migrate/migrations` remains available only for explicit/manual runs.

`202605190002_rbac_module_schema.sql` is the canonical RBAC-module baseline on the default
migration path. It already contains the current table structure, indexes, defaults, and comments,
so no plugin-boundary checkpoint or follow-up comment/audit-field migrations remain in this
directory.
