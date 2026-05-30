# Phase D Access Log Runtime Storage

## Status

- Topic: `phase-d-access-log-runtime-storage`
- Status: `in-progress`
- Task class: `server`
- Recovery source: `parent topic`
  - `phase-d-access-log-contract-definition`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- authority summary:
  - `server/internal/httpx/**` owns access-log runtime semantics and the canonical request -> access-log ingestion path
  - `server/internal/httpx/migrations/**` is the canonical live migration authority for `access_logs`
  - `server/internal/ent/migrate/migrations/**` remains archived/manual-only historical replay and is not part of the default migrate-up chain
  - runtime assembly for SQL-backed storage remains in `server/internal/app/**` and is already active in scope

## Authority-First Decision

This round compared two truthful MVP storage candidates for access logs:

| Candidate | Verdict | Reason |
| --- | --- | --- |
| PostgreSQL single table | chosen MVP authority path | matches current shared `*sql.DB` runtime, keeps canonical fields explicit, and is the smallest durable storage shape that fits the current backend |
| PostgreSQL partitioned table | deferred | retention policy, partition lifecycle, and migration/runtime assembly authority are not approved yet; adding partitions now would create policy and operational semantics this topic does not own |

Decision:

- canonical durable shape for this bounded slice is one PostgreSQL table: `access_logs`
- no metadata blob
- no audit/business/security-only fields
- no frontend-owned fields

## In-Scope Implementation Completed

### `server/internal/httpx/**`

- added a core-owned `AccessLogRepository` contract with:
  - `CreateAccessLog`
  - `CreateAccessLogs`
  - `DeleteAccessLogsBefore`
- added SQL repository implementation for canonical access-log fields only
- updated access-log middleware to build one canonical record from the request path and request auth context
- added sensitive-value filtering before persistence for:
  - password
  - token
  - authorization
  - cookie
  - obvious secret-like key/value patterns
- kept absent-by-design fields absent:
  - no request headers blob
  - no response headers blob
  - no body capture
  - no metadata blob
  - no action/resource/result/risk fields

### Migration authority repair

- moved `access_logs` live migration authority to `server/internal/httpx/migrations/202605300001_access_log_foundation.sql`
- kept `server/internal/ent/migrate/migrations/**` as archived/manual-only historical replay
- updated the default migrate registry so `graft migrate up` synthesizes `internal/httpx/migrations` together with live plugin-owned migration directories
- schema remains limited to canonical fields from `Access-Log-Authority-Contract.md`

## Wiring Decision

- `server/internal/app/runtime.go` is the canonical shared-SQL assembly point for core runtime dependencies.
- The runtime now creates `httpx.AccessLogRepository` from the existing shared `*sql.DB` held in `database.Resources`.
- The runtime passes that repository into `httpx.NewServer(...)`, which keeps access-log durable write on the existing HTTP middleware path instead of inventing a second bootstrap path.

## Migration-Chain Status

- `server/internal/pluginregistry` now treats `server/internal/httpx/migrations/**` as a live core-owned default migration directory.
- `graft migrate up`, `graft dev`, and `graft validate smoke` continue to use the same repository-default migration path, but that path now synthesizes `internal/httpx/migrations` before plugin-owned migration directories.
- `server/internal/ent/migrate/migrations/**` is still available for explicit/manual replay only and no longer holds the live `access_logs` baseline.
- Therefore fresh environments can receive `access_logs` through the repository-default migrate-up path without introducing a parallel migration runner.

## Remaining Gaps

1. This slice validates the default migrate-up authority path at unit/integration-test level, but it does not include a fresh-database end-to-end runtime probe against a real PostgreSQL instance.
2. The archived historical directory still contains old mixed migrations for other tables; this slice intentionally does not redesign those broader ownership leftovers beyond `access_logs`.

## Validation Truth

This round validated:

- `cd server && atlas migrate hash --dir file://internal/httpx/migrations`
- `cd server && go test ./internal/cli ./internal/pluginregistry ./internal/httpx ./internal/app`
- `go test ./internal/httpx`
- `go test ./internal/app`
- `git diff --check`

Fresh-environment migrate-up evidence:

- `server/internal/pluginregistry/registry_test.go` now asserts the default chain includes `internal/httpx/migrations` ahead of plugin-owned directories.
- `server/internal/cli/migrate_test.go` now asserts `resolveMigrationDirs(default)` returns the core-owned access-log migration dir and that `runMigrateUp` synthesizes a single Atlas directory containing the `access_logs` migration plus plugin-owned migrations.

This round still cannot truthfully claim:

- a real fresh PostgreSQL database was migrated and exercised by one live HTTP request inside this slice

## Final Verdict

- Verdict: `Archive Ready`

Basis:

- real runtime assembly now wires the existing SQL-backed `AccessLogRepository` into the server process
- access-log middleware durable write is active for actual server requests when the table already exists
- the repository-default migration chain now applies `access_logs` through the owner-aligned `server/internal/httpx/migrations/**` authority
