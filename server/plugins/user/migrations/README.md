This directory is the plugin-owned migration boundary for `server/plugins/user`.

`users` and `refresh_sessions` now rebuild from this directory as the default owner-aligned
baseline. The historical shared chain under `server/internal/ent/migrate/migrations` remains
available only for explicit/manual runs.

`202605190001_user_plugin_boundary_checkpoint.sql` is the first forward-only checkpoint in this
directory. It establishes the plugin-owned baseline for the user tables without keeping the
historical shared chain on the default migration path.
