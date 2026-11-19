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
5. Route-local compatibility callbacks run only if generated binding fails after the
   original guards. This preserves diagnostics, defaulted filters, and authorization or
   resource lookup order on existing endpoints. Normal request binding goes through
   oapi-codegen. Shared locale and request ID headers retain first-value semantics.
6. Where the HTTP body is still interpreted by an existing module parser, capture the
   bytes actually consumed during generated decoding and replay them without pre-reading
   the whole stream. Keep this on routes that need old input semantics or raw JSON.
   Multipart build input uses the generated reader under the existing byte limit.

### Follow-ups (recorded, not fixed here)

- Contract drift: audit module mounts `POST /api/audit/logs/batch-delete` while the spec
  declares `POST /api/audit/logs/deletions`. Owner: OpenAPI contract / audit module.
  The live fragment mounts the generated deletion operation. The URL stays the same for
  callers until the contract owner resolves the documented drift.
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

## Migration State

- Initial modules: announcement, audit, auth, backup, monitor, notification,
  runtime-target, scheduler, security, system-config and task.
- Remaining module routes migrated: build, container, network, project, rbac,
  registry, update and user.
- Coverage: 295 business operations with one generated method mount per operation and
  matching HTTP methods. The source check does not infer the final Gin path from group
  expressions. Existing route fragments and permission wiring are preserved by the
  registration diff, with the update diagnostic path placeholder aligned to its
  canonical name. The placeholder change is internal to Gin.
- Exceptions: 25 core-owned operations and the container Shell WebSocket operation.
  Credential vault, deployment and saved-view supply services without HTTP routes.

## Task Checklist

- [x] Classify route and operation owners.
- [x] Add per-domain strict generation and freshness checks.
- [x] Validate the system-config pilot and migrate the remaining module routes.
- [x] Compare business operations with strict mount coverage.
- [ ] Complete final validation and review.

## Acceptance Conditions

- Every module-owned JSON route is registered through generated wrapper methods and its
  handler is behind the generated `StrictServerInterface` (compile-checked).
- Non-strict endpoints carry an explicit whitelist comment in
  `route_coverage_test.go`.
- `just openapi-check` and `cd server && go run ./cmd/graft validate backend` pass.
- Freshness targets cover all new generated files; regeneration is deterministic.
- No response envelope, permission behavior, status code or error key change on migrated
  routes.
