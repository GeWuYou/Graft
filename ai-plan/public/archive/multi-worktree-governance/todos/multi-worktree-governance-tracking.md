# Multi Worktree Governance Tracking

## Topic

- Topic: `multi-worktree-governance`
- Branch: `refactor/server-module-boundaries`
- Worktree: repository root only; no dedicated long-lived worktree exists yet
- Scope: keep the shared recovery baseline short, freeze cross-worktree ownership truth, and prepare the first
  dedicated long-lived worktree/topic pair without reopening archived MVP recovery state.

## Goal

- Keep the active recovery entry focused on the current shared baseline, not on completed milestone history.
- Preserve the final `web` ownership model and the current `server` compile-time modular-monolith ownership model until
  dedicated worktrees are created.
- Keep historical detail available in topic-local archive snapshots instead of carrying it in the default recovery path.
- Land the confirmed server multi-worktree truth so future implementation rounds do not drift on what still blocks
  long-lived feature-worktree `functional zero-sharing`.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/roadmap/MVP实施计划.md`
- `ai-plan/public/README.md`
- `ai-plan/public/multi-worktree-governance/roadmap/server-module-boundaries-plan.md`

## Current Recovery Point

- `mvp-extension-path` is complete and remains archived under `ai-plan/public/archive/mvp-extension-path/`; it is not
  part of the active recovery path.
- The repository root is still the only active worktree and is currently on branch
  `refactor/server-module-boundaries`.
- The root worktree is currently a governance-only recovery entry, not a permanent feature-owned worktree; until a
  dedicated long-lived worktree/topic pair exists, feature-specific history should stay out of this topic unless it is
  directly about shared baseline governance or hotspot policy.
- The frozen `web` ownership model is:
  - `shell-owned`: `web/src/app/**`, `web/src/layouts/**`, `web/src/router/**`, `web/src/config/**`,
    `web/src/utils/route/**`, `web/src/store/modules/{user,permission}.ts`, `web/src/locales/**`, and platform
    `web/src/contracts/**`
  - `module-owned`: `web/src/modules/<name>/**`
  - `shared-owned`: `web/src/shared/**`
- Root-level module-specific files under `web/src/api/**`, `web/src/api/model/**`, and
  `web/src/contracts/{user,rbac}/**` are not valid steady-state ownership surfaces and must not return.
- The frozen `server` ownership model is:
  - compile-time modular monolith only; no runtime plugin loading, discovery, hot-load lifecycle, or generalized
    service locator
  - zero-shared means functional worktree zero-sharing, not absolute zero-sharing of every tracked file
  - the current server baseline has reached the governance target for long-lived feature-worktree functional
    zero-sharing
  - plugin-first owned scope under `server/plugins/<name>/**`
  - long-lived server feature worktrees should normally own only `server/plugins/<name>/**`
  - shared stable backend boundary under `server/internal/pluginapi/**` and `server/internal/contract/**`
  - centralized generated hotspot limited to `server/internal/pluginregistry/generated.go`
  - shared contracts, registry wiring, CLI wiring, `AGENTS.md`, `ai-plan/**`, and migration-entry changes belong to
    short-lived integration/core slices, not to standing feature-worktree ownership
  - the latest boundary cleanup confirmed:
    - runtime/core no longer depends on `server/internal/ent/**`
    - the default migration entry no longer includes the historical core/shared migration chain
    - the `server/internal/ent/**` Go/schema compatibility shell has been physically deleted
  - only `server/internal/ent/migrate/migrations/**` remains, and only for explicit/manual historical migration access
  - `user_roles` final owner is `rbac`
  - `user_roles` should stay at a `user_id / role_id` identifier boundary, not cross-plugin Ent edges
  - because the project is still early, whole-database rebuild is an allowed ownership-checkpoint posture as long as
    functionality remains unchanged; historical mixed migration replay compatibility is not required
  - new business logic must not flow back into `server/internal/store/**` or into `server/internal/ent/**`
- The latest backend ownership slices already landed:
  - `654c791` moved audit persistence into `server/plugins/audit/store/**` and `server/plugins/audit/storeent/**`
  - `5f45b31` removed the shared audit compatibility shim from `internal/store`
  - `799f1ff` removed the shared `user` store compatibility seam
  - `866582a` removed the shared `user/auth` seam, leaving `internal/store` as a doc-only placeholder rather than a
    business persistence owner
  - the remaining backend hotspot is now limited to the historical shared migration directory and broader worktree
    mapping governance, not the already-removed shared store seams or deleted `internal/ent` Go layer

## Long-Lived Worktree Mapping Policy

- A future long-lived worktree must not become active by implication alone; before feature recovery moves there, record:
  - its `Worktree` identity
  - its `Branch`
  - its dedicated active topic name
  - its owned scope
  - any shared hotspot exceptions it is still allowed to touch
- The first dedicated long-lived worktree/topic pair should be created only when one bounded slice is stable enough to
  own its own recovery path, such as one plugin or one hotspot-governance slice.
- Once a dedicated worktree/topic pair exists, give it its own tracking and trace files and stop appending that
  feature's phase ledger to `multi-worktree-governance`.
- If the repository root returns to `main`, update `ai-plan/public/README.md` in the same slice so the governance entry
  does not keep stale branch assumptions.
- If the repository root remains on `refactor/server-module-boundaries` temporarily, keep treating it as the shared
  baseline coordination point rather than as a long-lived feature-owned worktree.

## Shared Hotspots

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/**`
- `server/internal/pluginapi/**`
- `server/internal/contract/**`
- `server/internal/pluginregistry/generated.go`
- `server/cmd/graft/**`
- `server/internal/ent/migrate/migrations/**`
- `server/plugins/*/ent/**`
- `server/plugins/*/migrations/**`
- `web/src/app/**`
- `web/src/shared/**`
- `web/src/router/index.ts`
- `web/src/layouts/**`
- `web/src/store/modules/user.ts`
- `web/src/store/modules/permission.ts`
- `web/src/locales/**`

## Shared Hotspot Handling Policy

- Shared hotspots stay opt-in and limited; they are not default owned scopes for long-lived feature worktrees.
- A dedicated feature worktree should prefer plugin-owned or module-owned paths and avoid taking standing ownership of:
  - `server/internal/ent/migrate/migrations/**`
  - `server/internal/pluginregistry/generated.go`
  - `ai-plan/**` outside the worktree's own recovery topic
- If a slice needs both plugin-owned files and one of the shared hotspots, prefer either:
  - a separate bounded hotspot-governance slice on the root worktree
  - or serialized hotspot updates after the feature-owned slice lands
- `server/internal/pluginregistry/generated.go` remains the only accepted centralized plugin wiring artifact; parallel
  plugin work may each prepare their own plugin-local changes, but registry regeneration still requires explicit merge
  coordination. The file stays tracked for now, yet long-lived feature worktrees must not directly modify it.
- `server/internal/ent/migrate/migrations/**` remains the only shared backend hotspot under `internal/ent`; keep it
  out of default feature-worktree ownership and use it only for explicit/manual historical migration runs.
- Do not recreate `server/internal/ent/**` Go/schema compatibility code as a convenience layer.
- Fresh DB rebuild is an accepted validation posture for this ownership checkpoint; the topic does not require ongoing
  compatibility with historical mixed migration chains.

## Active Risks

- If future backend slices reopen `server/internal/store/**` or recreate shared `server/internal/ent/**` as a business landing
  zones, the first real multi-worktree merge wave will recreate avoidable hotspot churn.
- If future backend slices silently pull the default path back onto the historical shared migration chain, long-lived
  feature worktrees will regain a centralized migration hotspot that this slice intentionally removed.
- If future frontend slices reintroduce module truth outside `web/src/modules/<name>/**`, the `web` ownership freeze
  becomes unenforceable.
- If the repository root branch changes again and `ai-plan/public/README.md` is not updated in the same slice, future
  startup recovery will land on stale branch/worktree assumptions.
- If the first dedicated worktree/topic pair is created without an explicit owned-scope definition, this governance
  topic will continue accumulating feature-specific history that belongs elsewhere.

## Phased Path To Functional Zero-Sharing

- Phase 1 is already acceptable as a short-lived integration hotspot posture:
  - keep compile-time generated plugin registry in place
  - keep registry and CLI wiring in bounded shared slices only
- Phase 2 continues plugin-local ownership hardening:
  - avoid reopening shared store seams
  - keep new business logic inside `server/plugins/<name>/**`
  - keep cross-plugin collaboration on capability/contract boundaries
- Phase 3 has landed and established the current functional zero-sharing baseline:
  - runtime/core stays free of new `server/internal/ent/**` business-plugin truth dependencies
  - the default migration entry stays off the historical core/shared replay chain and on the owner-aligned baseline
  - `server/internal/ent/**` stays deleted as a live Go/schema surface; only the historical manual migration directory may remain
  - `user_roles -> rbac` ownership remains preserved under the allowed early-phase whole-database rebuild posture

## Latest Validation

- Latest backend validation for the functional zero-sharing baseline:
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
  - `cd server && go test ./...`
  - `cd server && go build ./cmd/graft`
  - fresh DB proof on local Docker `postgres:16`:
    - `cd server && GRAFT_DATABASE_URL='postgres://graft:graft@localhost:55432/graft_repair_round?sslmode=disable' GRAFT_REDIS_ADDR='127.0.0.1:6379' GRAFT_AUTH_JWT_SECRET='repair-round-secret' go run ./cmd/graft migrate up`
    - resulting tables include `users`, `refresh_sessions`, `roles`, `permissions`, `user_roles`, `role_permissions`, `audit_logs`
    - revision rows include `202605190001`, `202605190002`, `202605190003`

## Archive Pointers

- Pre-compaction tracking snapshot:
  `ai-plan/public/archive/multi-worktree-governance/archive/todos/multi-worktree-governance-tracking-pre-compaction-2026-05-19.md`
- Pre-compaction trace snapshot:
  `ai-plan/public/archive/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`

## Immediate Next Step

- Keep `multi-worktree-governance` limited to shared baseline governance while the repository root remains the only
  active worktree.
- For the next backend slice, keep `server/internal/ent/**` deleted as a live code surface, keep the owner-aligned
  default migration baseline intact, and decide whether the historical manual migration directory should stay in place
  or be archived behind a later governance slice.
- Before creating the first dedicated long-lived worktree/topic pair, record its owned scope and shared-hotspot policy
  in `ai-plan/public/README.md` and give it its own tracking/trace files instead of extending this governance topic.
