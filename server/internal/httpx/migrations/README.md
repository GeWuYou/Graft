This directory is the core-owned migration boundary for `server/internal/httpx`.

`access_logs` rebuilds from this directory through the repository-default `graft migrate up`
chain because access-log runtime semantics and durable storage authority both belong to
`server/internal/httpx/**`.

Live migration rules in this directory follow `server/AGENTS.md` exactly:

- every new live table must have a Chinese `COMMENT ON TABLE`
- every live column must have a Chinese `COMMENT ON COLUMN`
- handwritten migration SQL must persist those comments explicitly rather than relying on
  comments declared elsewhere
- completion validation must include a check that live table and column comments are complete

The old shared Ent/manual replay chain has been removed. During early development, this
owner-aligned directory is the only migration authority for HTTP access-log storage.
