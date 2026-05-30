This directory is the core-owned migration boundary for `server/internal/httpx`.

`access_logs` rebuilds from this directory through the repository-default `graft migrate up`
chain because access-log runtime semantics and durable storage authority both belong to
`server/internal/httpx/**`.

The historical shared chain under `server/internal/ent/migrate/migrations` remains available
only for explicit/manual replay and is not the canonical owner for new live migrations.
