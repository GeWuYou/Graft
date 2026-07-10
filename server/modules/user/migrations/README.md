This directory is the module-owned migration boundary for `server/modules/user`.

`users` rebuilds from this directory as the user-profile baseline. The later
cleanup migration removes the historical credential columns and
`refresh_sessions` table after auth has copied their data into auth-owned
storage.
The old shared Ent/manual replay chain has been removed and is no longer a fallback authority.

`202605190001_user_module_schema.sql` is the canonical user-module baseline on the default
migration path. It already contains the current table structure, indexes, defaults, and comments,
so no module-boundary checkpoint or follow-up comment/status migrations remain in this directory.
