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

- Current batch: `Batch 3 - Backend recording integration for user and RBAC actions`
- Completed batches:
  - `Batch 0 - Exploration and worktree/topic setup`
  - `Batch 1 - Backend audit domain design and schema`
  - `Batch 2 - Backend API, permission, menu, OpenAPI contract`
- Pending batches:
  - `Batch 3 - Backend recording integration for user and RBAC actions`
  - `Batch 4 - Frontend audit module and page`
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
- Batch 2 closed only the guarded read path; richer domain write-path recording for user and RBAC mutations still belongs
  to Batch 3.
- The root OpenAPI spec and backend generated bundle/types are now updated for audit read closure, but frontend audit
  module work remains untouched until Batch 4.

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

- Batch 3 should add domain-owned active-audit emission at user/rbac write success points instead of altering the settled
  audit read contract.
- Batch 4 should consume the existing `audit.read` permission, `/audit/logs` bootstrap menu path, and generated read DTO
  contract rather than redefining page-local equivalents.

## Immediate Next Step

- Start Batch 3 on top of the completed Batch 2 read contract:
  - emit explicit audit events from user-management write flows and RBAC write flows
  - preserve request-level middleware as fallback only
  - keep Batch 3 inside backend write-path integration without widening the API or frontend scope

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
