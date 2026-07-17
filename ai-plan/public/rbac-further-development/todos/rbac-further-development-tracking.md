# RBAC Further Development Tracking

## Topic

- Topic: `rbac-further-development`
- Status: `active recovery entry`
- Goal: continue RBAC work with durable topic recovery while using reusable temporary Agent worktrees.
- Recovery source: parent topic archived -> new standalone topic

## Scope

- Owned scope:
  - `server/modules/rbac/**`
  - `web/src/modules/rbac/**`
- A worktree directory and task branch are not standing ownership. Each task declares its bounded scope and returns the
  directory to the reusable pool after developer-confirmed integration.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`

## Current Recovery Point

- Recover through this topic, not through a fixed local directory or branch.
- Before implementation, use Worktree Manager to acquire a clean unique task branch from `main`.
- Shared registry, contract, router, layout, locale, and migration work remains an explicit bounded integration slice.

## Task Checklist

- [x] Replace fixed worktree/branch recovery with reusable numbered worktrees and unique task branches.
- [x] Record the active topic's bounded scope and shared integration hotspots.
- [ ] Continue RBAC implementation from a fresh startup preflight and a bounded task branch.
- [ ] Run the required server/web validation for each cross-boundary RBAC batch before integration.

## Acceptance Conditions

- Recovery uses the topic documents and startup preflight, never a fixed local directory or stale task branch.
- Every RBAC batch declares owned files, avoids shared hotspots unless explicitly integrated, and records validation evidence.
- A batch is complete only when its directly affected server and web validation passes, or a concrete blocker is recorded.
- The topic remains active until all planned RBAC batches are complete and the archive-readiness conditions are verified.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "reusable-worktree-migration-and-recovery-docs"
  ],
  "pending_batches": [
    "rbac-bounded-cross-boundary-implementation"
  ],
  "current_batch": "rbac-bounded-cross-boundary-implementation",
  "next_batch": "rbac-bounded-cross-boundary-implementation",
  "closeout_status": "recovery-docs-complete-implementation-pending"
}
```

## Shared Hotspots

- `ai-plan/public/README.md`
- `server/internal/moduleregistry/generated.go`
- `server/internal/moduleapi/**`
- `server/internal/contract/**`
- `web/src/router/**`
- `web/src/layouts/**`
- `web/src/locales/**`

## Immediate Next Step

- Start from root startup preflight, recover this topic, then acquire a reusable worktree for the bounded RBAC task.
