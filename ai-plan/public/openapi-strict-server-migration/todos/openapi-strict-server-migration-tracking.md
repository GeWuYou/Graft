# OpenAPI Strict Server Migration Tracking

## Topic

OpenAPI Strict Server Migration

## Scope

Migrate all business modules under `server/modules/` from hand-wired gin routes plus
generated types to oapi-codegen strict server interfaces with generated gin wrappers.
Response envelopes, permission middleware behavior and error mapping stay byte-compatible.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `openapi/openapi.yaml` (semantic authority; not edited by this topic)
- `server/internal/httpx/response.go`, `server/internal/httpx/authz.go` (envelope and
  authorization owners; consumed, not edited)

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: "OpenAPI is the shared wire authority; internal/contract/openapi is the derived consumption surface; internal/httpx owns the response envelope and authorization middleware; server/modules/<name>/contract owns route and permission constants."
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-work-intake
bootstrap:
  targets:
    - ai-plan/public/openapi-strict-server-migration/README.md
    - ai-plan/public/openapi-strict-server-migration/startup-prompt.md
    - ai-plan/public/openapi-strict-server-migration/todos/openapi-strict-server-migration-tracking.md
    - ai-plan/public/openapi-strict-server-migration/traces/openapi-strict-server-migration-trace.md
closeout:
  archive: true
  lessons_review: true
```

## Batch 0: Inventory Classification (2026-11-19)

Route inventory: 297 hand-registered route registrations; OpenAPI bundle: 321 operations.
Method: scripted extraction of route registration calls and bundle operations, joined per
module during migration.

### Module to operation domain mapping

| Module (`server/modules/`) | Operation tags | Contract package (`internal/contract/openapi/`) |
| --- | --- | --- |
| announcement | announcements | `announcement` |
| audit | audit | `audit` |
| auth | auth | `auth` |
| backup | platform-backup | `backup` (new) |
| build | build | `build` (new) |
| container | container | `container` |
| monitor | monitor | `monitor` |
| network | platform-network | `network` |
| notification | notifications | `notification` |
| project | application | `project` (new) |
| rbac | rbac (incl. `/api/users/*/roles` ops) | `rbac` |
| registry | registry | `registry` (new) |
| runtime-target | runtime-target | `runtimetarget` |
| scheduler | scheduled-tasks | `scheduler` |
| security | security | `security` (new) |
| system-config | system-config | `systemconfig` |
| task | tasks | `task` (new) |
| update | platform-update | `update` (new) |
| user | users | `user` |

Modules without HTTP routes: `credential-vault` (agent PKI trust surface),
`deployment` (task runtime execution), `saved-view` (cross-module service only).

### Classification

(a) Strict-interface candidates: every JSON operation owned by the modules above,
including paginated lists, saved-view CRUD and batch actions. Container log streaming
declares `application/json` and works inside a strict method body because handler bodies
write directly through `*gin.Context`.

(b) Cannot enter strict interfaces:

- `getContainerShellWebSocket` (`GET /api/ops/containers/{id}/shell/ws`, 101 upgrade,
  container module keeps hand-written registration).
- Core route owners outside module scope (stay hand-written, whitelisted in
  `route_coverage_test.go`): `internal/httpx` access-log/app-log explorer and saved views,
  agent transport routes (no spec), `internal/dashboard` (`/api/dashboard/summary`,
  `/api/dashboard/widgets/{widget_id}`), `internal/moduleruntime` (`/api/modules/runtime`),
  `internal/capability` (`/api/platform/capabilities`), `internal/realtime`
  (`GET /sse`, `GET /ws`), `internal/mcp` (`/mcp` streamable, `text/event-stream`),
  `internal/app` (`/healthz`, docs endpoints).

(c) Design decisions:

1. Handler bodies keep writing responses through `httpx.WriteSuccess` /
   `AbortLocalizedError` / `AbortAppError` on the gin context and strict methods return
   `nil, nil`; generated response objects are not serialized because the generated
   envelope field order and `json.Encoder` trailing newline would change response bytes.
2. Route registration keeps module group plus fragment constants and mounts generated
   `ServerInterfaceWrapper` methods per route instead of `RegisterHandlers`, because the
   bundle paths are absolute and modules register under the shared `/api` group. This
   preserves middleware order exactly: group middlewares, per-route `RequirePermission`,
   generated param binding, strict body binding.
3. `RequirePermission` stays per-route with module contract permission constants instead
   of an operation-lookup map; the compile-time check is interface satisfaction
   (`var _ <domain>.StrictServerInterface = adapter{}`) plus route coverage tests
   asserting engine routes against generated operation sets.
4. Wrapper/strict binding errors route through configurable error handlers that write the
   unified envelope (module invalid-request key where one exists) instead of the
   oapi-codegen default `{"msg": ...}` body.
5. Known edge shifts recorded as accepted contract alignment: malformed typed path/query
  /header parameters now fail at the generated binding layer with the unified envelope
   before reaching handler bodies; duplicated request-id headers fail binding instead of
   being silently truncated.

### Follow-ups (recorded, not fixed here)

- Contract drift: audit module mounts `POST /api/audit/logs/batch-delete` while the spec
  declares `POST /api/audit/logs/deletions`. Owner: OpenAPI contract / audit module.
  The strict adapter already covers the operation binding; the live route stays on the
  legacy fragment until the contract owner fixes the path.
- `trend-range-query` parameter documents a lenient fallback while its schema enum
  constrains values; generated binding does not validate enums so behavior matches
  today, but the schema/description conflict should be resolved by the contract owner.
- `server/internal/cli/validate_openapi.go` gained one target entry
  (`backend-openapi-servers`) as required wiring for freshness coverage; it is outside
  the worker owned-scope and was recorded here deliberately.
- `scripts/openapi_generated_backend_boundary_audit.py` exempts `zz_generated.*` files
  from the generated-runtime-takeover scan; generated strict-server wrappers legitimately
  contain runtime markers and are byte-validated by the freshness targets, while
  hand-written takeover remains blocked. Accepted by lead review as a necessary
  cross-boundary touch. The corrective slice additionally refined the takeover markers to
  word-boundary regexes and allows the sanctioned per-route `ServerInterfaceWrapper`
  mounting plus `NewStrictHandlerWithOptions` in hand-written registrations; actual
  takeover calls (`RegisterHandlers*`, `HandlerFromMux`, bare `NewStrictHandler(`,
  `StrictHandlerFunc`, `StrictHTTPServerOptions`) stay forbidden.

## Migration State (batch 3, 2026-11-19)

Migrated modules (strict wrapper mounted, `StrictServerInterface` adapter asserted,
tests green): `system-config`, `task`, `notification`, `security`, `backup`, `monitor`,
`scheduler`, `audit`, `auth`, `announcement`, `runtime-target` (11 modules).

Not migrated (registration still hand-wired; regeneration for their domains not yet
applied). Per-module recovery recipe, identical mechanics to the migrated set:

| Module | Ops | Notes for migration |
| --- | --- | --- |
| user | 14 | 10 inline closures in route_user_handlers.go / route_admin_session_handlers.go to convert to named registrar methods; body ops postUsers/postUserUpdate/postUserStatus/postUserResetPassword must take strict bodies; remove `useropenapi.WriteServerInterface` / `ReadServerInterface` spike assertions and colliding contract spike types.go; saved-view create/update need `CacheRequestBody`. |
| update | 10 | routes.go mounts; body ops postPlatformUpdateCheck/postPlatformUpdateOperation/postPlatformUpdateOperationRecovery need strict bodies. |
| build | 11 | route_registration.go; body ops postBuildJob/postBuildInputSnapshot/postBuildArtifactPromotion need strict bodies; domain dir is new (`build`). |
| registry | 18 | route_registration.go; body ops postRegistry/postRegistryVerify/postRegistryArtifactRepository/putRegistry variants and assignment batch ops need strict bodies; public available-destinations uses build permission. |
| network | 17 | routes.go; connectivity body ops plus postPlatformNetworkDiagnostic/putPlatformNetworkOutbound/resetPlatformNetworkOutbound need strict bodies; existing `network` domain include list must extend beyond connectivity ops. |
| rbac | 22 | route_registration.go + route_write_handlers.go + route_read_handlers.go + saved_views.go; remove rbacopenapi ReadServerInterface / WriteServerInterface / UserRoleServerInterface spikes and types.go; body ops need strict bodies; saved views need `CacheRequestBody`. |
| project | 58 | largest module; route_registration.go; many body ops (application create/import/template/workspace/file content); public template endpoints keep explicit no-guard mounts; run param-name reconciliation against spec (`applicationId` etc.). |
| container | 45 plus whitelisted WS | route_registration.go + saved_views.go; exclude `getContainerShellWebSocket` from include list (whitelisted hand-written); realtime ops are core-owned (already excluded); many body ops (actions, pull/tag, shell session, batch); saved views need `CacheRequestBody`. |

## Current Recovery Point

- Current batch: 3 (11 modules migrated; 9 modules remain with per-module recipes above).
- Generation infrastructure, freshness targets (`backend-openapi-servers`), CLI wiring,
  whitelist documentation test and shared `strictsupport` bridge are in place.
- Next step: continue module migration following the recipe table; dead hand-wired
  binding code (spike `X{}.Method()` assertions) for migrated modules is already removed
  inside each migration commit.

## Task Checklist

- [x] Batch 0: topic bootstrap and route/operation classification.
- [x] Batch 1: generation infrastructure and freshness coverage.
- [x] Batch 2: permission middleware strategy + pilot module `system-config`
      (byte-identical probe comparison on 7 scenarios except natural timestamps).
- [x] Batch 3 (partial): migrated system-config, task, notification, security, backup,
      monitor, scheduler, audit, auth, announcement, runtime-target.
- [ ] Batch 4: migrate user, update, build, registry, network, rbac.
- [ ] Batch 5: migrate project, container (largest; WS whitelisted).
- [ ] Closeout: full `go run ./cmd/graft validate backend` (lint stage blocked by local
      golangci-lint/go1.26 toolchain mismatch, CI owns final lint), lessons review,
      archive.

## Acceptance Conditions

- Every module-owned JSON route is registered through generated wrapper methods and its
  handler is behind the generated `StrictServerInterface` (compile-checked).
- Non-strict endpoints carry an explicit whitelist comment in
  `route_coverage_test.go`.
- `just openapi-check` and `cd server && go run ./cmd/graft validate backend` pass.
- Freshness targets cover all new generated files; regeneration is deterministic.
- No response envelope, permission behavior, status code or error key change on migrated
  routes.
