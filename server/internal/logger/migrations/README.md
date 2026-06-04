# logger migrations

This directory is the core-owned migration boundary for `server/internal/logger`.

`app_logs` rebuilds from this directory through the repository-default `graft migrate up`
chain because App Log runtime semantics, durable storage authority, and retention cleanup
belong to `server/internal/logger/**`.

Live migration rules in this directory follow `server/AGENTS.md` exactly:

- every new live table must have a Chinese `COMMENT ON TABLE`
- every live column must have a Chinese `COMMENT ON COLUMN`
- handwritten migration SQL must persist those comments explicitly rather than relying on
  comments declared elsewhere
- completion validation must include a check that live table and column comments are complete

The historical shared chain under `server/internal/ent/migrate/migrations` remains available
only for explicit/manual replay and is not the canonical owner for new live migrations.
