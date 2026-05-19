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
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
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
  - current state has NOT yet reached long-lived feature-worktree functional zero-sharing
  - plugin-first owned scope under `server/plugins/<name>/**`
  - long-lived server feature worktrees should normally own only `server/plugins/<name>/**`
  - shared stable backend boundary under `server/internal/pluginapi/**` and `server/internal/contract/**`
  - centralized generated hotspot limited to `server/internal/pluginregistry/generated.go`
  - shared contracts, registry wiring, CLI wiring, `AGENTS.md`, `ai-plan/**`, and migration-entry changes belong to
    short-lived integration/core slices, not to standing feature-worktree ownership
  - current confirmed blockers are:
    - runtime/core still depends on `server/internal/ent/**`
    - the default migration entry still includes the historical core/shared migration chain
    - `server/internal/ent/**` remains a compatibility shell/shared hotspot
  - `server/internal/ent/**` remains transitional only: no new business truth may land there; it may stay as a
    temporary compatibility shell until import/runtime/test/generation dependencies are cleared
  - physical deletion of `server/internal/ent/**` is allowed only after those dependencies are cleared
  - `user_roles` final owner is `rbac`
  - `user_roles` should stay at a `user_id / role_id` identifier boundary, not cross-plugin Ent edges
  - because the project is still early, whole-database rebuild is an allowed ownership-checkpoint posture as long as
    functionality remains unchanged; historical mixed migration replay compatibility is not required
  - new business logic must not flow back into `server/internal/store/**` or non-core-owned portions of
    `server/internal/ent/**`
- The latest backend ownership slices already landed:
  - `654c791` moved audit persistence into `server/plugins/audit/store/**` and `server/plugins/audit/storeent/**`
  - `5f45b31` removed the shared audit compatibility shim from `internal/store`
  - `799f1ff` removed the shared `user` store compatibility seam
  - `866582a` removed the shared `user/auth` seam, leaving `internal/store` as a doc-only placeholder rather than a
    business persistence owner
  - the remaining backend hotspot is now deeper ownership work around `server/internal/ent/**` and migration
    boundaries, not the already-removed shared store seams

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
- `server/internal/ent/**`
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
  - `server/internal/ent/**`
  - `server/internal/ent/migrate/migrations/**`
  - `server/internal/pluginregistry/generated.go`
  - `ai-plan/**` outside the worktree's own recovery topic
- If a slice needs both plugin-owned files and one of the shared hotspots, prefer either:
  - a separate bounded hotspot-governance slice on the root worktree
  - or serialized hotspot updates after the feature-owned slice lands
- `server/internal/pluginregistry/generated.go` remains the only accepted centralized plugin wiring artifact; parallel
  plugin work may each prepare their own plugin-local changes, but registry regeneration still requires explicit merge
  coordination. The file stays tracked for now, yet long-lived feature worktrees must not directly modify it.
- `server/internal/ent/**` and `server/internal/ent/migrate/migrations/**` remain shared backend hotspots until the
  deeper ownership migration finishes; do not treat them as safe default parallel-worktree surfaces.
- `server/internal/ent/**` is transitional only. Keep it free of new business truth, use it only as a temporary
  compatibility shell where still needed, and delete it only after import/runtime/test/generation dependencies are
  fully removed.
- Fresh DB rebuild is an accepted validation posture for this ownership checkpoint; the topic does not require ongoing
  compatibility with historical mixed migration chains.

## Active Risks

- If future backend slices reopen `server/internal/store/**` or shared `server/internal/ent/**` as business landing
  zones, the first real multi-worktree merge wave will recreate avoidable hotspot churn.
- If future backend slices assume functional zero-sharing is already achieved before the runtime/core and migration-entry
  dependencies are removed, the first long-lived feature worktrees will still collide on `internal/ent/**` and the
  shared migration chain.
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
- Phase 3 is the remaining blocker-clearing phase before long-lived feature-worktree functional zero-sharing:
  - remove runtime/core dependence on `server/internal/ent/**` for business-plugin truth
  - move the default migration entry off the historical core/shared replay chain and onto an owner-aligned baseline
  - shrink `server/internal/ent/**` to core-owned-only residue, then delete it once import/runtime/test/generation
    dependencies are gone
  - preserve `user_roles -> rbac` ownership while keeping the allowed early-phase whole-database rebuild posture

## Latest Validation

- Latest backend validation carried by the active baseline before this compaction:
  - `cd server && go test ./plugins/rbac ./plugins/user`
  - `cd server && go test ./internal/store/...`
  - `cd server && env GOCACHE=/tmp/go-build go run ./cmd/graft validate backend --stage lint`
- This compaction slice rechecked the recovery-path shape with:
  - `git show --stat --oneline --decorate=short 654c791 5f45b31 799f1ff 866582a --`
  - `git diff -- ai-plan/public/multi-worktree-governance`
  - `wc -l ai-plan/public/multi-worktree-governance/todos/multi-worktree-governance-tracking.md ai-plan/public/multi-worktree-governance/traces/multi-worktree-governance-trace.md`

## Archive Pointers

- Pre-compaction tracking snapshot:
  `ai-plan/public/multi-worktree-governance/archive/todos/multi-worktree-governance-tracking-pre-compaction-2026-05-19.md`
- Pre-compaction trace snapshot:
  `ai-plan/public/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`

## Immediate Next Step

- Keep `multi-worktree-governance` limited to shared baseline governance while the repository root remains the only
  active worktree.
- For the next backend slice, continue reducing deeper `internal/ent/**` and migration ownership hotspots without
  weakening the frozen `rbac` ownership over `user_roles`, and keep any shared wiring or migration-entry changes in
  short-lived integration/core slices instead of feature-owned worktrees.
- Before creating the first dedicated long-lived worktree/topic pair, record its owned scope and shared-hotspot policy
  in `ai-plan/public/README.md` and give it its own tracking/trace files instead of extending this governance topic.
