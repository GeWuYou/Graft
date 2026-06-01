This directory is the plugin-owned migration boundary for `server/modules/user`.

`users` and `refresh_sessions` now rebuild from this directory as the default owner-aligned
baseline. The historical shared chain under `server/internal/ent/migrate/migrations` remains
available only for explicit/manual runs.

`202605190001_user_plugin_boundary_checkpoint.sql` remains a checksum-stable no-op checkpoint for
environments that already applied the original plugin boundary marker.

`202605190004_user_plugin_boundary_baseline.sql` is the first forward-only DDL baseline in this
directory. It establishes the plugin-owned user tables without keeping the historical shared chain
on the default migration path.
