# Multi Worktree Governance Trace

## 2026-05-19 active recovery compaction

- Archived the previous active tracking and trace files into topic-local snapshots because the default recovery path had
  grown past the point where it was useful as a startup entry:
  - `ai-plan/public/multi-worktree-governance/archive/todos/multi-worktree-governance-tracking-pre-compaction-2026-05-19.md`
  - `ai-plan/public/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`
- Replaced the active tracking file with a short recovery entry that keeps only:
  - current branch/worktree truth
  - frozen ownership baselines
  - shared hotspots
  - active risks
  - latest validation
  - immediate next step

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
- The latest landed backend milestone remains the Phase 3e/3f RBAC persistence cleanup:
  - live RBAC repository wiring is plugin-local
  - transitional shared RBAC adapter/store paths are removed
  - embedded-RBAC tests now consume plugin-local store contracts

## Historical Detail Pointer

- Full milestone history from `2026-05-17` through the pre-compaction `2026-05-19` slices now lives in:
  `ai-plan/public/multi-worktree-governance/archive/traces/multi-worktree-governance-trace-pre-compaction-2026-05-19.md`
- Use that snapshot only when a task explicitly needs older validation logs, intermediate migration notes, or the full
  chronology of the web/server/docs governance slices.

## Immediate Next Step

- Keep this topic focused on shared baseline governance until the first real dedicated worktree/topic pair exists.
- When the next slice becomes feature-owned instead of governance-owned, create a dedicated topic entry rather than
  appending another long phase ledger here.
