This directory is the plugin-owned migration boundary for `server/plugins/rbac`.

`roles`, `permissions`, `role_permissions`, and RBAC-owned `user_roles` now rebuild from this
directory as the default owner-aligned baseline. The historical shared chain under
`server/internal/ent/migrate/migrations` remains available only for explicit/manual runs.

`202605190002_rbac_plugin_boundary_checkpoint.sql` is the first forward-only checkpoint in this
directory. It establishes the RBAC-owned baseline, including `user_roles`, without leaving the
historical shared chain on the default migration path.
