# Phase D Access Log Runtime Storage

## Status

- Topic: `phase-d-access-log-runtime-storage`
- Status: `archived`
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
  - runtime assembly for SQL-backed storage remains in `server/internal/app/**` and is active in scope

## Authority-First Decision

This topic chose PostgreSQL single-table storage for MVP access logs:

| Candidate | Verdict | Reason |
| --- | --- | --- |
| PostgreSQL single table | chosen MVP authority path | matches current shared `*sql.DB` runtime, keeps canonical fields explicit, and is the smallest durable storage shape that fits the current backend |
| PostgreSQL partitioned table | deferred | partition lifecycle and operational semantics are not required for the bounded MVP retention model |

Decision:

- canonical durable shape is one PostgreSQL table: `access_logs`
- no metadata blob
- no audit/business/security-only fields
- no frontend-owned fields

## Final Result

- `server/internal/httpx/**` owns the `AccessLogRepository`, SQL-backed persistence, middleware record construction, retention cleanup job, explorer read handlers, permission/menu registration, and live migrations.
- `server/internal/app/**` wires the shared SQL-backed repository into `httpx.NewServer(...)`, registers the access-log explorer after auth/rbac services are available, and registers the access-log retention cleanup job.
- `openapi/**` owns the access-log list/detail wire contract; generated server/web OpenAPI artifacts are derived consumers.
- `web/src/modules/access-log/**` consumes the canonical API through module-owned API/types/contracts/pages and remains a downstream explorer UI only.
- `Access Log` remains request-fact authority only; audit/security/app-log semantics remain outside the `access_logs` table.

## Validation Truth

This topic was validated through:

- `cd server && atlas migrate hash --dir file://internal/httpx/migrations`
- `cd server && go test ./internal/cli ./internal/pluginregistry ./internal/httpx ./internal/app`
- `go test ./internal/httpx`
- `go test ./internal/app`
- `git diff --check`

Fresh-environment runtime proof completed on 2026-06-04:

- Disposable PostgreSQL: `postgres:16` on `localhost:55433`, database `graft_access_log_proof`
- Disposable Redis: `redis:7.2-alpine` on `localhost:56380`
- Command:
  - `cd server && GRAFT_APP_ENV=local GRAFT_HTTP_ADDR=127.0.0.1:18080 GRAFT_DATABASE_DRIVER=postgres GRAFT_DATABASE_URL='postgres://graft:graft@localhost:55433/graft_access_log_proof?sslmode=disable' GRAFT_REDIS_ADDR=127.0.0.1:56380 GRAFT_REDIS_PASSWORD= GRAFT_REDIS_DB=0 GRAFT_AUTH_JWT_SECRET='<local-secret>' go run ./cmd/graft validate smoke --timeout 20s`
- Secret note:
  - `<local-secret>` is a disposable local smoke-test value only.
- Result:
  - all live migrations applied against a fresh PostgreSQL database
  - runtime started and served `/healthz`
  - `access_logs` contained one persisted row: `GET /healthz 200`
  - disposable containers were removed after verification

## Remaining Risks

- The historical shared migration directory still contains older mixed migrations for other tables; this topic intentionally does not redesign those broader ownership leftovers beyond `access_logs`.
- Production-scale retention tuning and partitioning remain future operational topics, not blockers for the MVP access-log runtime-storage closure.

## Final Verdict

- Verdict: `Archived`

Basis:

- real runtime assembly wires the SQL-backed `AccessLogRepository` into the server process
- access-log middleware durable write is active for actual server requests
- the repository-default migration chain applies `access_logs` through the owner-aligned `server/internal/httpx/migrations/**` authority
- fresh PostgreSQL smoke proof confirms one live request is persisted to `access_logs`
