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
- Recovery source: `parent topic`
  - `ai-plan/public/README.md`
  - archived `backend-rbac-contract-audit`
  - current plugin/OpenAPI/web bootstrap implementation
- Loop mode: `topic-completion-loop`

## Batch State

- Current batch: `Batch 0 - Exploration and worktree/topic setup`
- Completed batches:
  - none
- Pending batches:
  - `Batch 1 - Backend audit domain design and schema`
  - `Batch 2 - Backend API, permission, menu, OpenAPI contract`
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
- The existing audit plugin is write-only today; query API, permissions, menu, OpenAPI path additions, and web page
  still need explicit implementation.
- The first audit worktree attempt was incorrectly branched from `main`; Batch 0 corrected this by rebuilding the
  worktree from the clean RBAC baseline before continuing.

## Exploration Snapshot

- Plugin registration:
  - `server/plugins/<name>/descriptor.go` owns `plugin.Descriptor` metadata and plugin-owned migration dirs.
  - `server/internal/pluginregistry/generated.go` is the single generated compile-time registry consumed by CLI/runtime.
  - `server/internal/pluginregistry/registry.go` expands ordered descriptors and default migration dirs.
- Audit plugin current baseline:
  - `server/plugins/audit/plugin.go` already mounts request-level middleware and subscribes to
    `pluginapi.AuditRecordEventName`.
  - `server/internal/audit/service.go` and `server/plugins/audit/store*` are write-only today; there is no read/query
    service or HTTP API yet.
  - Current stored fields are request-oriented `operator_*`, `action`, `resource_*`, `request_*`, `ip`, `user_agent`,
    `success`, `error_message`, `created_at`.
- Migration pattern:
  - `graft migrate up` defaults to `pluginregistry.DefaultMigrationDir`, which synthesizes a temporary Atlas chain from
    ordered plugin-owned migration dirs.
  - `server/internal/ent/migrate/migrations` is historical shared migration storage only, not the default apply chain.
- OpenAPI/generated pattern:
  - Canonical source lives in `openapi/openapi.yaml` plus `openapi/paths/**`.
  - Backend generated types live under `server/internal/contract/openapi/**`.
  - Web generated types live under `web/src/contracts/openapi/generated/schema.ts` and are produced by
    `bun run openapi:types`.
- HTTP/authz pattern:
  - `server/internal/httpx/response.go` defines the uniform success/error envelope and request-id handling.
  - `server/internal/httpx/authz.go` attaches `pluginapi.RequestAuthContext` after authentication and uses
    `RequirePermission` for guarded routes.
- Frontend registration and guard pattern:
  - `web/src/modules/index.ts` auto-registers only modules that provide both `index.ts` and `bootstrap-routes.ts`.
  - `web/src/store/modules/permission.ts` stores bootstrap menus/permissions and derives async routes.
  - `web/src/app/bootstrap/route-guards.ts` restores bootstrap state and mounts dynamic routes.
  - `web/src/app/bootstrap/permission-directive.ts` implements `v-permission`, while pages often add local computed
    permission checks before issuing requests.
- Likely future audit insertion points:
  - `server/plugins/user/plugin.go` service success points for create/update/status/delete/reset-password.
  - `server/plugins/rbac/write_service.go` management writer success points for role and binding mutations.

## Batch Implications

- Batch 1 should evolve the existing audit plugin instead of creating a second audit baseline.
- Batch 2 should add audit OpenAPI path fragments into the canonical root `openapi/**` chain and keep generated outputs
  aligned through the existing backend/web workflows.
- Batch 3 should keep current request-level auto audit as fallback and add richer domain events at user/rbac
  service-writer success points.
- Batch 4 should follow the existing `index.ts + bootstrap-routes.ts + module api/types + list page` pattern and avoid
  inventing a new route-meta permission system.

## Immediate Next Step

- Finish Batch 0 validation and create the docs-only setup commit before any runtime implementation batch starts.
