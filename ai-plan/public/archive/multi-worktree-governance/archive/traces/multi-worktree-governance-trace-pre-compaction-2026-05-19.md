# Multi Worktree Governance Trace

## 2026-05-20 server functional zero-sharing baseline completed

- Closed the remaining server-side blockers and reached the current governance target for long-lived feature-worktree
  functional zero-sharing:
  - runtime/core no longer depends on `server/internal/ent/**`
  - the default migration entry no longer replays the historical shared chain
  - `server/internal/ent/**` Go/schema compatibility residue has been physically removed
- Repaired the owner-aligned default migration baseline after a real fresh-DB failure exposed that Atlas cannot safely
  apply multiple independent versioned directories sequentially against one database.
- Changed the default `graft migrate up` execution model to synthesize one temporary Atlas directory from the ordered
  plugin-owned SQL files, hash it, and apply it as one chain.
- Revalidated the repaired baseline against a real local Docker `postgres:16` fresh database:
  - applied `202605190001`, `202605190002`, `202605190003`
  - built `users`, `refresh_sessions`, `roles`, `permissions`, `user_roles`, `role_permissions`, `audit_logs`
- The only remaining `internal/ent` residue is the explicit/manual historical migration directory under
  `server/internal/ent/migrate/migrations/**`; it is no longer part of the default feature-worktree path.

## 2026-05-19 shared internal-ent Go residue removed

- Deleted the remaining `server/internal/ent/**` Go packages, generated code, forwarder schema shells, and tests after
  verifying no non-doc Go imports outside that tree still depended on them.
- Preserved only `server/internal/ent/migrate/migrations/**` as the explicit/manual historical migration path:
  - the default migration chain already points at the owner-aligned plugin registry
  - `pluginregistry.HistoricalSharedMigrationDir` remains the only live non-doc reference into that historical path
- Updated the active governance truth to match the new boundary:
  - `internal/ent/**` is no longer a live compatibility shell or runtime dependency surface
  - future work must not silently recreate shared Ent code under `server/internal/ent/**`
  - the remaining `internal/ent` residue is historical SQL state, not active plugin/runtime truth

## 2026-05-19 Phase 1 server functional zero-sharing governance freeze

- Tightened the active server governance truth for Phase 1 multi-worktree ownership without changing code or roadmap:
  - clarified that zero-shared means functional worktree zero-sharing, not absolute tracked-file zero-sharing
  - froze long-lived server feature worktrees to normally owning only `server/plugins/<name>/**`
  - recorded that shared contracts, registry wiring, CLI wiring, `AGENTS.md`, `ai-plan/**`, and migration-entry
    changes belong to short-lived integration/core slices
- Made the remaining shared backend hotspots explicit instead of leaving them implied:
  - `server/internal/ent/**` stays transitional only, cannot accept new business truth, may remain as a compatibility
    shell, and can be physically deleted only after import/runtime/test/generation dependencies are cleared
  - `server/internal/pluginregistry/generated.go` remains tracked for now, but long-lived feature worktrees must not
    directly modify it
- Locked the ownership semantics that still matter before any deeper implementation slice:
  - `user_roles` stays owned by `rbac` at a `user_id / role_id` identifier boundary rather than cross-plugin Ent edges
  - fresh DB rebuild is acceptable for this ownership checkpoint; historical mixed migration compatibility is not a
    Phase 1 requirement

## 2026-05-19 long-lived worktree mapping rules made explicit

- Expanded the active public recovery docs so the root-only state is operationally clear:
  - the repository root remains the only active worktree
  - the root is acting as shared-baseline governance, not as a permanent feature-owned worktree
  - future long-lived worktrees must be mapped with their own active topic plus tracking/trace pair before feature
    recovery moves there
- Added an explicit shared-hotspot handling policy to the active recovery path:
  - shared hotspots remain limited and opt-in
  - plugin-owned worktrees should not take standing ownership of `internal/ent/**`,
    `internal/ent/migrate/migrations/**`, or `internal/pluginregistry/generated.go`
  - hotspot updates should be split into bounded governance slices or serialized instead of being co-owned by default

## 2026-05-19 active recovery compaction

- Archived the previous active tracking and trace files into topic-local snapshots because the default recovery path had
  grown past the point where it was useful as a startup entry:
  - `ai-plan/public/archive/multi-worktree-governance/archive/todos/multi-worktree-governance-tracking-pre-compaction-2026-05-19.md`
  - `ai-plan/public/archive/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`
- Replaced the active tracking file with a short recovery entry that keeps only:
  - current branch/worktree truth
  - frozen ownership baselines
  - shared hotspots
  - active risks
  - latest validation
  - immediate next step

## 2026-05-19 backend boundary cleanup landed

- `654c791` moved audit persistence into plugin-local storage surfaces under `server/plugins/audit/**`, so audit no longer
  relies on the shared store path for its live repository wiring.
- `5f45b31` removed the shared audit compatibility shim from `internal/store`, closing the last shared audit transition path.
- `799f1ff` removed the shared user store compatibility seam, so user tests and reset helpers now exercise plugin-local
  store contracts instead of the old shared adapter.
- `866582a` removed the shared user/auth seam and left `internal/store` as documentation-only scaffolding, which means the
  active backend hotspot is no longer shared store cleanup but the deeper `internal/ent/**` and migration ownership boundary.

## Active Baseline After Compaction

- `mvp-extension-path` stays archived and is no longer part of the active recovery path.
- The repository root on `refactor/server-module-boundaries` remains the only active worktree.
- `web` baseline stays frozen on:
  - shell-owned `app/layouts/router/config/locales/platform-contract` surfaces
  - module-owned `web/src/modules/<name>/**`
  - shared-owned `web/src/shared/**`
- `server` baseline stays frozen on:
  - compile-time modular monolith
  - plugin-first ownership under `server/plugins/<name>/**`
  - shared backend boundary at `internal/pluginapi/**` and `internal/contract/**`
  - generated shared hotspot at `internal/pluginregistry/generated.go`
  - `user_roles -> rbac` ownership
- The latest landed backend milestone is now the full shared-store seam cleanup for audit, user, and user/auth:
  - live RBAC persistence is plugin-local
  - transitional shared audit/user compatibility paths are removed
  - `internal/store` is no longer a business persistence landing zone

## Historical Detail Pointer

- Full milestone history from `2026-05-17` through the pre-compaction `2026-05-19` slices now lives in:
  `ai-plan/public/archive/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`
- Use that snapshot only when a task explicitly needs older validation logs, intermediate migration notes, or the full
  chronology of the web/server/docs governance slices.

## Immediate Next Step

- Keep this topic focused on shared baseline governance until the first real dedicated worktree/topic pair exists.
- When the next slice becomes feature-owned instead of governance-owned, create a dedicated topic entry rather than
  appending another long phase ledger here.
