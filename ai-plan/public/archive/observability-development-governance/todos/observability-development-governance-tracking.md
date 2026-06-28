# Observability Development Governance Tracking

## Topic

- Topic: `observability-development-governance`
- Status: `archived`
- Goal:
  - define the backend development standard for app log / audit / security event / metric placeholder
  - inventory and roll out bounded compliance fixes against current code
  - expose audit/security governance capability to frontend audit and access-control pages
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `request-correlation-access-logging`
  - archived `logging-unification-rollout`
  - archived `plugin-audit-correlation-governance`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/observability-development-governance`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Startup Receipt

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary governance loop`
- Recovery source: `archive-ready evidence`
- Authority summary:
  - `server/internal/logger/**` is the backend app/error logger authority
  - `server/internal/httpx/**` is the request correlation, access log, and HTTP security-event authority
  - `server/internal/audit/**` plus `server/plugins/audit/**` own audit persistence and audit field normalization
  - audit/security frontend consumption must follow backend authority instead of inventing client-side semantics

## Scope

- Owned scope:
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
  - `ai-plan/public/archive/observability-development-governance/**`
  - `ai-plan/public/README.md`
  - later Phase B/Phase C bounded files only after inventory
- Forbidden scope:
  - OpenTelemetry, Prometheus, Grafana, or full metrics rollout
  - generic log abstraction that erases `App Log / Access Log / Audit Event / Security Event` boundaries
  - repo-wide cleanup unrelated to governance findings

## Batch State

- Completed batches:
  - `phase-a-logging-development-standard`
  - `phase-b-logging-compliance-rollout`
  - `phase-c-audit-console-governance-ux`
- Pending batches:
  - none
- Current batch:
  - none
- Next batch:
  - none

## Phase A Notes

- Completed on `2026-05-29`.
- Established canonical classification:
  - `App Log`
  - `Access Log`
  - `Error Log`
  - `Audit Event`
  - `Security Event`
  - `Metric Candidate / Metric Placeholder`
- Confirmed correlation authority remains:
  - `requestId` from unified `httpx` middleware
  - `traceId == requestId` in MVP
  - actor and request metadata via canonical request context / audit normalization path
- Confirmed asynchronous rules are differentiated:
  - `App Log` and `Error Log` default sync
  - `Audit Event` async allowed with fallback error log
  - `Security Event` async bridge allowed with fallback error log

## Next Phase Entry Criteria

- Phase B must start with inventory only
- no code changes before inventory table exists
- every Phase B fix must map back to a clause in `ai-plan/design/domains/audit/日志治理开发规范.md`

## Phase B Inventory

| Area | File | Line/Call Site | Current Pattern | Expected Pattern | Severity | Fix Now? | Reason |
| --- | --- | --- | --- | --- | --- | --- | --- |
| runtime startup | `server/internal/i18n/service.go` | `registerDefaultCatalogs -> panic(fmt.Sprintf(...))` | production startup path uses `panic(...)` on default catalog registration failure | return error through runtime/bootstrap path or emit canonical error log before bounded exit | high | yes | Phase A `Error Log` / async-shutdown rules forbid runtime panic as normal failure control flow |
| app log canonical fields | `server/plugins/auth/route_errors.go` | `writeResponseMappingError` | error log uses legacy `request_id` field name | use canonical `requestId` and shared correlation field dictionary | medium | yes | Phase A field spec requires canonical fields in new/owned production logs |
| app log canonical fields | `server/plugins/monitor/plugin.go` | `newServerStatusHandler` error branches | error log uses legacy `request_id` field name | use canonical `requestId` and align with route/method correlation keys | medium | yes | same canonical field rule; production path is in owned scope |
| global logger direct use | `server/plugins/user/storeent/runtime.go` | `ent.Log(... zap.L().Debug ...)` | plugin ent runtime logs through global logger directly | reuse injected/shared logger path rather than direct global logger access | medium | yes | Phase A layer matrix expects external/repository-adjacent diagnostics to stay on canonical logger path |
| manual correlation helper | `server/plugins/user/route_user_handlers.go` | `withAuditRequestID(... ginCtx.GetHeader(...))` in write routes | handler manually injects request-id helper into ctx before service call | inherit canonical request ctx directly; no second request-id propagation model | high | yes | Phase A correlation rules forbid hand-built second correlation context |
| manual correlation helper | `server/plugins/rbac/route_write_handlers.go` | `withRBACAuditRequestID(... ginCtx.GetHeader(...))` in write routes | handler manually injects request-id helper into ctx before service call | inherit canonical request ctx directly; no second request-id propagation model | high | yes | same correlation rule; production write path |
| manual audit event fields | `server/plugins/user/plugin.go` | `publishAudit` callers populate `RequestID: currentRequestID(ctx)` | plugin-domain audit event still manually copies request id into event payload | rely on canonical ctx-based audit enrichment; remove request-id side channel where bounded | high | yes | Phase A says compatibility helpers may remain temporarily but must not stay the default development path |
| manual audit event fields | `server/plugins/rbac/write_service.go` | `publishAudit` callers populate `RequestID: currentRBACRequestID(ctx)` | plugin-domain audit event still manually copies request id into event payload | rely on canonical ctx-based audit enrichment; remove request-id side channel where bounded | high | yes | same as above |
| generated ent debug default | `server/plugins/user/ent/client.go` | `newConfig -> log.Println` | generated client still defaults debug logger to stdlib `log.Println` | defer; runtime currently overrides active ent logging through plugin runtime wrapper | future | no | generated artifact and not the active production path in current runtime wiring |
| generated ent debug default | `server/plugins/rbac/ent/client.go` | `newConfig -> log.Println` | generated client still defaults debug logger to stdlib `log.Println` | defer; runtime currently overrides active ent logging through plugin runtime wrapper | future | no | same reason; generated artifact outside bounded fix-now slice |
| CLI stdout output | `server/cmd/graft-jwt-secret/main.go` | `fmt.Println(line)` | command prints generated env line to stdout | preserve; stdout is user-facing command output, not app log | low | no | Phase A CLI rule distinguishes stdout/stderr from app log |
| CLI stdout output | `server/cmd/graft-signing-key/main.go` | `fmt.Println(line)` | command prints generated env line to stdout | preserve; stdout is user-facing command output, not app log | low | no | same as above |

## Phase B Scope Decision

- `Fix Now = yes` current batch focus:
  - `server/internal/i18n/service.go`
  - `server/plugins/auth/route_errors.go`
  - `server/plugins/monitor/plugin.go`
  - `server/plugins/user/storeent/runtime.go`
  - `server/plugins/user/route_user_handlers.go`
  - `server/plugins/rbac/route_write_handlers.go`
  - `server/plugins/user/plugin.go`
  - `server/plugins/rbac/write_service.go`
- Explicitly deferred in this batch:
  - generated `ent/**` fallback defaults not exercised by the current runtime path
  - tests
  - intentional CLI stdout helpers

## Phase B Rollout Result

- Fixed now:
  - removed manual request-id helper propagation from `user` and `rbac` HTTP write paths
  - removed manual `RequestID` payload population from `user` and `rbac` plugin-domain audit event publishers
  - changed owned production error logs from legacy `request_id` to canonical `requestId`
  - replaced one remaining runtime `panic` startup path in `i18n` with explicit error return
  - replaced `user` plugin storeent runtime global logger direct use with explicit injected logger
- Preserved intentionally:
  - audit read/query compatibility fields such as `request_id`
  - generated OpenAPI / generated Ent compatibility surfaces
  - CLI stdout env-line helpers

## Phase B Validation

- Passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`

## Phase C Scope Decision

- Current batch focus:
  - `web/src/modules/audit/**`
  - `web/src/modules/rbac/pages/index.vue`
  - matching locale updates under owned module scope
- Authority rule:
  - frontend only consumes existing backend audit/correlation semantics
  - no new backend audit model, schema, or metrics implementation introduced

## Phase C Rollout Result

- Added route-query-driven audit-log filter hydration and URL sync for shareable governance views.
- Added first-class audit correlation visibility: `requestId` / `traceId` display, copy, URL filters, and server-side request-id query mapping.
- Added first-class audit context fields: `source` (`audit event` vs `security event`), `actor`, `action`, `resource`, `result`, and `reason`.
- Added RBAC role-list, RBAC permission-list, and user-list navigation into `/audit/logs` with bounded related-audit queries.
- Added correlation-aware success/error hints on user and RBAC write operations so operators can retain the latest troubleshooting id.
- Preserved intentionally:
  - existing audit backend query contract
  - current page structure and shell layout
  - no broad audit-console redesign
  - P2 risk summary, trend, and timeline remain future scope; fake overview risk-watch runtime copy was removed

## Phase C Validation

- Passed:
  - `cd web && bun run check`

## Closeout Decision

- Topic status moved to `archive-ready` after Phase A, Phase B, and Phase C all reached acceptance.
- No further batch remains inside the approved three-phase loop.
- Any follow-up on metrics governance, deeper audit UX, or broader observability productization must open a new bounded topic.

## P2 Follow-Up Startup Receipt

- Date: `2026-05-29`
- Follow-up: `audit-console-analytics-p2`
- Governance source: `root AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Authority summary:
  - `server/plugins/audit/store/**` and `server/internal/audit/**` own the read-model semantics for audit analytics
  - `openapi/**` owns the shared API contract consumed by `web`
  - `web/src/modules/audit/**` remains a downstream presentation layer and must not infer analytics from paged list data

## P2 Follow-Up Scope

- Trigger:
  - Phase C intentionally stopped after governance UX and removed fake overview risk-watch runtime content.
  - P2 now covers only the missing real analytics needed by the audit overview console.
- Owned implementation scope for the follow-up:
  - `server/internal/audit/**`
  - `server/plugins/audit/**`
  - `openapi/**`
  - `web/src/modules/audit/**`
- Explicit non-goals:
  - general observability dashboards
  - metrics/tracing rollout
  - compatibility shim that derives analytics from frontend log pages
  - broad audit-console IA or shell changes

## P2 Follow-Up Contract Decision

- `high-risk event summary`
  - no new field accepted
  - canonical count remains `summary.high_risk_events` on `/audit/overview`
- `/audit/overview` accepted additions:
  - `risk_groups`
    - bounded grouped summary owned by backend audit authority
    - shape: array of group objects with stable `key`, display `label_key`, `count`, and `risk_level`
    - initial groups stay bounded to operator-facing audit categories instead of arbitrary user-defined aggregations
  - `trend`
    - bounded server-computed time series owned by backend audit authority
    - shape: object with `bucket_unit`, `bucket_size`, and ordered `points`
    - each point carries `bucket_start`, `bucket_end`, `total`, `failed`, `high_risk`, and `security_events`
    - bucket plan is derived from the existing `window` query rather than a new free-form range parameter
  - `security_timeline`
    - bounded recent-event collection owned by backend audit authority
    - shape: array of timeline items with `id`, `created_at`, `source`, `risk_level`, `action`, `result`, `request_id`, and optional actor/resource labels
    - this collection is for recent security-relevant events, not a second paginated log endpoint
- `/audit/logs` accepted addition:
  - `source`
    - first-class query filter
    - backed by the existing backend `AuditSource` authority values
    - initial enum remains bounded to `REQUEST`, `SECURITY_EVENT`, and `DOMAIN_EVENT`
- Query/governance note:
  - no update to `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` is required in this batch because the governance model is unchanged; this round only records the accepted canonical owner and the smallest new contract fields under existing authority-first rules.

## Expected Batch 3 Implementation Scope

- Backend:
  - extend audit store/service/repository read DTOs for `risk_groups`, `trend`, `security_timeline`, and log-list `source` query
  - implement the smallest SQL/repository aggregation needed for the accepted overview fields
  - surface the new fields through audit HTTP mappers and read handlers
- OpenAPI:
  - add the new `/audit/overview` schemas
  - add `/audit/logs` `source` query parameter and list-item/timeline enums as needed
- Frontend:
  - consume the canonical overview additions in the existing overview page layout
  - keep current shell and page structure stable
- Deferred beyond batch 3:
  - custom analytics drill-down
  - configurable trend windows
  - additional backend filtering not required by the accepted fields

## P2 Follow-Up Implementation Result

- Implemented backend authority additions:
  - `/audit/overview` now returns `risk_groups`, `trend`, and `security_timeline`
  - `/audit/logs` now accepts first-class `source` query filtering
- Implemented consumer updates:
  - `web/src/modules/audit/pages/overview/index.vue` renders backend-owned grouped risk analytics, trend bars, and the security timeline inside the existing page structure
  - `web/src/modules/audit/pages/logs/index.vue` and `AuditFilters.vue` now consume the canonical `source` query/filter semantics
- Preserved intentionally:
  - existing shell/layout and page IA
  - existing `summary.high_risk_events` as the canonical high-risk count
  - no frontend-derived fallback analytics
  - no metrics/tracing or broader observability rollout

## P2 Follow-Up Validation

- Passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`
  - `cd web && bun run check`
  - `git diff --check`

## P2 Follow-Up Closeout Decision

- Follow-up status moved to `archive-ready`.
- No remaining batch stays in scope for `audit-console-analytics-p2`.
- Any later audit-console analytics expansion must open a new bounded topic instead of reusing this closed follow-up.

## Final Status

- Result: `archived`
- Commit eligibility:
  - owned scope is clear
  - validation is complete for directly changed code
  - this archive move only closes the already accepted topic and follow-up evidence
- Archive notes:
  - future `metrics-governance` work must open a separate bounded topic
  - further audit-console analytics work must start as a new bounded topic instead of extending this archived line
