# OpenAPI Strict Server Migration Trace

## 2026-11-19 Batch 0: bootstrap and inventory

- Bootstrapped topic per ADR-003 contract-driven minimal bootstrap; Work Contract v1
  recorded in tracking.
- Inventoried 297 hand-registered route registrations across `server/modules`,
  `server/internal/httpx`, `server/internal/dashboard`, `server/internal/capability`,
  `server/internal/realtime`, `server/internal/mcp`, `server/internal/moduleruntime`,
  `server/internal/app` and 321 OpenAPI bundle operations.
- Classified strict candidates, non-strict endpoints (WebSocket shell, SSE/WS realtime,
  MCP streamable, healthz/docs) and core-owned route registrations outside module scope.
- Design decisions recorded in tracking batch 0: hybrid strict adapter writing envelopes
  through `httpx` for byte compatibility; generated `ServerInterfaceWrapper` mounted per
  route with existing group and fragment constants; per-route `RequirePermission`
  preserved; configurable binding error handlers for envelope compatibility.
- Next: batch 1 generation infrastructure.

## 2026-11-19 Batches 1-3: infrastructure, pilot and module migration

- Batch 1 (commit `6e457d156`): per-domain `oapi-codegen.yaml` configs
  (models + gin-server + strict-server via `include-operation-ids`), `go:generate` line
  switch to config-driven generation, `scripts/openapi_generated_freshness_check.py`
  gained the `backend-openapi-servers` target list, `graft validate openapi` gained the
  single `backend-openapi-servers` target wiring, and the generated-artifact boundary
  audit now exempts `zz_generated.*` files (tool-owned, byte-checked by freshness) while
  blocking hand-written takeover markers.
- Batch 2 (commit `806a1c3c6`): pilot `system-config` migrated; shared
  `internal/contract/openapi/strictsupport` bridge added (GinContext extraction, unified
  envelope error handlers, Dispatch with cached-body reset). Behavior comparison probe:
  7 scenarios (list 200, missing key 404, unauthorized 401, forbidden write 403, invalid
  body 400, update 200, reset 404) produced byte-identical envelopes between the
  baseline hand-wired handlers and the strict-server migration except the natural
  `updated_at` timestamp in the update scenario. Permanent route-contract test committed
  as `route_contract_test.go`.
- Batch 3 commits: task/notification/security/backup/monitor (`9bc7413c1`),
  scheduler/audit/auth (`4f1773e36`), announcement (`28dbd7856`), runtime-target
  (`c5405f9e8`). Each commit mounts generated `ServerInterfaceWrapper` methods on the
  existing group/fragment constants, asserts `var _ <domain>.StrictServerInterface =
  adapter{}`, converts body ops to strict-bound `req.Body`, removes the old
  `X{}.Method()` spike assertions and (where present) the colliding hand-written
  `types.go` spike interfaces, and adds saved-view `CacheRequestBody` where handlers
  re-bind RawMessage bodies.
- Discoveries recorded as follow-ups: audit `/logs/batch-delete` vs spec
  `/logs/deletions` contract drift; `trend-range-query` enum vs documented fallback.
- Batch 3 stops after 11 of 20 route-owning modules (see tracking table for per-module
  recipes); runtime and CI validation is green except the local golangci-lint toolchain
  mismatch (lint stage only; CI owns the pinned lint).

## 2026-11-19 Corrective slice: legacy freshness targets repointed

- Lead review found `go run ./cmd/graft validate openapi` failing on the pre-existing
  `backend-monitor` target; per-target diagnosis reproduced it and identified
  `backend-auth-session` as the only other failing target (backend-health,
  backend-rbac-management, backend-user-write, backend-modules-runtime,
  backend-network-connectivity still pass).
- Root cause: monitor and auth artifacts are now generated through their per-domain
  configs (models + gin-server + strict-server). The legacy targets still regenerated
  types-only output, whose expected file is exactly the tracked file minus the generated
  server section (models/types byte-identical, no semantic drift). Running the legacy
  `--mode fix` would have stripped the strict server code, so the targets were repointed
  instead: `run_backend_monitor` and `run_backend_auth_session` now validate through the
  artifact's own `oapi-codegen.yaml` via the shared strict-server comparison helper, and
  the obsolete types-only arg blocks were removed.
- Earlier full-chain `exit=0` readings in this trace were shell-quoting artifacts of the
  worker harness; the corrective run captures exit codes inside a script file.
- Verification: all 8 freshness targets pass; full `go run ./cmd/graft validate openapi`
  and `go run ./cmd/graft validate backend --stage buildtest` pass.
- Follow-up in the same corrective slice: with freshness repaired the full chain exposed
  the DTO boundary audit flagging the sanctioned strict-server integration in module
  files (`ServerInterfaceWrapper` mounting, `NewStrictHandlerWithOptions` matched via the
  `HandlerWithOptions(` substring). The audit's takeover scan was refined: markers use
  word-boundary regexes (so `NewStrictHandlerWithOptions(` no longer matches
  `HandlerWithOptions(`) and `ServerInterfaceWrapper` is no longer a forbidden marker
  because mounting generated wrapper methods inside hand-written registration is the
  recorded integration pattern. Actual takeover calls (`RegisterHandlers*`,
  `HandlerFromMux`, bare `NewStrictHandler(`, `StrictHandlerFunc`,
  `StrictHTTPServerOptions`) stay forbidden for hand-written files; audit unit tests
  pass.
- Migration note for remaining modules: when user/rbac/network/modules/health domains
  move to per-domain configs, the corresponding legacy targets must be repointed the
  same way (see tracking recipe table).
