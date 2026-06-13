This directory is the module-owned migration boundary for `server/modules/rbac`.

`roles`, `permissions`, `role_permissions`, and RBAC-owned `user_roles` rebuild from this
directory as a clean empty-database baseline. The old shared Ent/manual replay chain has been
removed and is no longer a fallback authority.

`202605190002_rbac_baseline.sql` is the canonical RBAC-module baseline on the default
migration path. It already contains the current table structure, indexes, defaults, and comments,
so no module-boundary checkpoint or follow-up comment/audit-field migrations remain in this
directory.
