# Audit Plugin MVP Tracking

## Topic

- Topic: `audit-plugin-mvp`
- Status: `active`
- Goal: establish and close the audit plugin MVP through bounded cross-boundary batches.
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `backend-rbac-contract-audit` topic
  - current plugin registry implementation
  - current user plugin implementation
  - current rbac plugin implementation
  - current OpenAPI/generated contract workflow
  - current web module/bootstrap/route implementation
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`

## Scope

- Owned scope follows the topic README and startup prompt.
- Forbidden scope includes unrelated RBAC expansion, auth redesign, global layout redesign, broad i18n refactor, and
  unrelated generated or formatting churn.

## Startup Receipt

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: `subtopic`
  - `ai-plan/public/README.md`
  - `ai-plan/public/audit-plugin-mvp/README.md`
  - `ai-plan/public/audit-plugin-mvp/todos/audit-plugin-mvp-tracking.md`
  - `ai-plan/public/audit-plugin-mvp/traces/audit-plugin-mvp-trace.md`
  - archived `backend-rbac-contract-audit`
  - current plugin/OpenAPI/web bootstrap implementation
- Loop mode: `topic-completion-loop`

## Batch State

- Current batch: `Batch 5 - Cross-boundary integration and regression`
- Completed batches:
  - `Batch 0 - Exploration and worktree/topic setup`
  - `Batch 1 - Backend audit domain design and schema`
  - `Batch 2 - Backend API, permission, menu, OpenAPI contract`
  - `Batch 3 - Backend recording integration for user and RBAC actions`
  - `Batch 4 - Frontend audit module and page`
- Pending batches:
  - `Batch 5 - Cross-boundary integration and regression`
  - `Batch 6 - Archive-ready closeout`

## Batch 0 Checklist

- [x] Read root `AGENTS.md`
- [x] Read `.ai/environment/tools.ai.yaml`
- [x] Read `server/AGENTS.md`
- [x] Read `web/AGENTS.md`
- [x] Read `ai-plan/public/README.md`
- [x] Check `git status --short`
- [x] Check current branch and worktree list
- [x] Confirm the RBAC source worktree is clean
- [x] Create dedicated worktree `feat/wt-audit-plugin-mvp` from the RBAC baseline
- [x] Re-run startup preflight in the new worktree
- [x] Update `ai-plan/public/README.md` mapping
- [x] Create topic recovery docs
- [x] Record exploration findings
- [ ] Run `git diff --check`
- [ ] Re-check `git status --short`
- [ ] Create docs-only setup commit

## Risks

- The current repository already contains a minimal audit plugin and historical audit-related migrations, so MVP work
  is additive and corrective rather than greenfield.
- Batch 4 now closes only the frontend read surface; Batch 5 still needs end-to-end regression confirmation across
  bootstrap menus, permission visibility, generated schema consumption, and runtime route recovery.
- Request-level fallback and broader auth/session/request-context redesign remain intentionally out of scope.

## Exploration Snapshot

- Plugin registration:
  - `server/plugins/<name>/descriptor.go` owns `plugin.Descriptor` metadata and plugin-owned migration dirs.
  - `server/internal/pluginregistry/generated.go` is the single generated compile-time registry consumed by CLI/runtime.
  - `server/internal/pluginregistry/registry.go` expands ordered descriptors and default migration dirs.
- Audit plugin current baseline:
  - `server/plugins/audit/plugin.go` mounts request-level automatic audit middleware and now also mounts guarded read
    routes, registers plugin-owned permissions/menus/messages, and exports the read service for plugin lifecycle wiring.
  - `server/internal/audit/service.go` remains the canonical read/write service surface; Batch 2 reused `List(ctx, query)`
    instead of adding a parallel read model.
- OpenAPI/generated pattern:
  - Canonical source lives in `openapi/openapi.yaml` plus `openapi/paths/**`.
  - Batch 2 added `/api/audit/logs` plus audit list schemas, refreshed `openapi/dist/openapi.bundle.json`, and refreshed
    `server/internal/contract/openapi/generated/types.gen.go` plus the narrow `server/internal/contract/openapi/audit/**`
    package.
  - Web generated schema was not refreshed in this batch because no owned-scope frontend runtime or contract consumer was
    added yet.
- HTTP/authz pattern:
  - `server/internal/httpx/response.go` remains the uniform success/error envelope and request-id boundary.
  - Audit read routes use `httpx.RequirePermission(..., "audit.read")` and keep the existing localized error behavior.
- Frontend registration and guard pattern:
  - Batch 2 only registered the backend bootstrap menu contract for `/audit/logs`; no frontend page/module work was
    started.

## Batch Implications

- Batch 3 added domain-owned active-audit emission at user/rbac write success points without altering the settled audit
  read contract.
- Batch 4 should consume the existing `audit.read` permission, `/audit/logs` bootstrap menu path, and generated read DTO
  contract rather than redefining page-local equivalents.

## Immediate Next Step

- Start Batch 5 on top of the validated Batch 4 frontend baseline:
  - run bounded cross-boundary integration and regression across backend bootstrap/menu/permission data and the new
    audit web module
  - verify `/audit/logs` dynamic route recovery, `audit.read` visibility gating, and generated contract consumption stay
    aligned end to end
  - keep Batch 5 on regression and integration only instead of widening feature scope

## Batch 1 Snapshot

- Extended the audit persistence contract and plugin-owned SQL repository from request-only fields to a richer audit
  domain:
  - actor user id / username / display name
  - action
  - resource type / id / name
  - success result
  - request id
  - ip / user agent
  - message
  - JSON metadata
  - created at
- Added `internal/audit` service-layer support for:
  - `Record(ctx, input)` with normalization and sensitive-data redaction
  - `List(ctx, query)` with bounded pagination/filter normalization
- Preserved non-blocking audit semantics on both paths:
  - request middleware still logs write failures without breaking the request
  - active event subscription now swallows malformed payload / write failures after logging
- Added plugin-owned migration `202605270001_audit_log_domain_upgrade.sql` and refreshed `plugins/audit/migrations/atlas.sum`.
- Added bounded tests for:
  - service sanitization and pagination normalization
  - SQL repository create/list behavior and filters
  - plugin non-blocking active-audit failure behavior

## Batch 1 Validation

- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `git diff --check`

## Batch 2 Snapshot

- Added plugin-owned audit contract values under `server/plugins/audit/contract/**` for:
  - read permission code `audit.read`
  - menu title key `menu.audit.logs.title`
  - guarded API/menu path alignment on `/audit/logs`
- Completed plugin registration closure for Batch 2:
  - `DependsOn()` now declares `user`, `rbac`
  - `Register()` now registers audit messages, permission, menu, read service, and guarded routes before event-bus
    subscription
  - route guard resolution now consumes the existing `pluginapi.AuthService` and `pluginapi.Authorizer`
- Added guarded read API implementation:
  - plugin-owned route registration at `/api/audit/logs`
  - generated-parameter binding to `auditcore.ListQuery`
  - generated response mapping back to the canonical `httpx` success envelope
- Added root OpenAPI path and schemas for audit list querying and refreshed backend generated artifacts:
  - `openapi/paths/audit.logs.yaml`
  - `openapi/components/schemas/audit-log-list-*.yaml`
  - `openapi/dist/openapi.bundle.json`
  - `server/internal/contract/openapi/generated/types.gen.go`
  - `server/internal/contract/openapi/audit/**`
- Extended audit plugin tests to cover:
  - new authz wiring requirements in plugin registration
  - menu/permission/read-route smoke coverage

## Batch 2 Validation

- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `git diff --check`
- OpenAPI/backend generated step executed:
  - `cd web && bun ../scripts/openapi-bundle.mjs`
  - `cd server && go generate ./internal/contract/openapi`
- Web generated schema intentionally not updated in Batch 2 because no owned-scope frontend runtime or consumer was added.

## Batch 3 Snapshot

- Added bounded richer active-audit emission in `user` success paths:
  - create
  - update
  - status update
  - delete
  - reset password
- Added bounded richer active-audit emission in `rbac` success paths:
  - role create/update/status/delete
  - role-permission replace/add/remove
  - user-role replace/add/remove
- Kept audit write failure non-blocking for business success paths by swallowing event-bus publish failures after
  logging, matching the existing request-level fallback posture.
- Propagated request ids from current user/rbac write routes into active audit events without changing the settled
  `/api/audit/logs` read contract.
- Added bounded tests covering:
  - successful user active-audit publish
  - successful rbac active-audit publish
  - audit publish failure remaining non-blocking on user/rbac success paths

## Batch 3 Validation

- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `git diff --check`

## Batch 4 Snapshot

- Added module-owned frontend audit runtime under `web/src/modules/audit/**`:
  - module registration entry
  - bootstrap route declaration
  - module contract values for route path and permission code
  - API adapter that consumes the settled `/api/audit/logs` generated contract through the existing request adapter
  - read-only audit log list page and locale bundles
- Kept frontend route and permission truth aligned with the existing shell path:
  - `modules/index.ts` module discovery
  - bootstrap menu to dynamic route transform
  - `audit.read` permission continues to come only from the backend bootstrap snapshot
- Refreshed frontend generated OpenAPI schema because Batch 4 introduced a real frontend consumer of the audit read
  contract.
- Added bounded frontend tests for:
  - audit bootstrap route registration
  - module registration map exposure
  - bootstrap route transform coverage for `/audit/logs`
  - locale visible-copy governance inclusion
  - audit page render smoke path

## Batch 4 Validation

- `cd web && bun run openapi:types`
- `cd web && bun run check`
- `git diff --check`
- Retry worker closeout notes:
  - final `bun run check` passed after wiring the audit permission contract into visible page actions, reducing
    page-local duplication to satisfy `jscpd`, and stubbing TDesign/directive dependencies in the page smoke test
  - no backend contract, menu, or permission semantics were changed during retry closeout
