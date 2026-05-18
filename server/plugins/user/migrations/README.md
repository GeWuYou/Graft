This directory is the plugin-owned migration boundary for `server/plugins/user`.

Historical mixed migrations remain under `server/internal/ent/migrate/migrations` until the
`users` / `refresh_sessions` / `user_roles` ownership history can be split without rewriting
the live Atlas chain. New user-only migration versions should land here.
