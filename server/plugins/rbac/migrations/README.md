This directory is the plugin-owned migration boundary for `server/plugins/rbac`.

Historical mixed migrations remain under `server/internal/ent/migrate/migrations` until the
`user_roles` / `roles` / `permissions` ownership history can be split without rewriting the
live Atlas chain. New RBAC-only migration versions should land here.

`202605190002_rbac_plugin_boundary_checkpoint.sql` is the first forward-only checkpoint in this
directory. It is intentionally a no-op baseline marker so the plugin-owned migration chain becomes
runnable without pretending the historical mixed Atlas files were already split.
