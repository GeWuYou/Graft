---
name: graft-worktree-manager
description: Manage reusable Graft AI-agent worktrees through guarded status, acquire, and release commands without treating directories or task branches as long-lived ownership.
---

# Graft Worktree Manager

Use this skill whenever an agent needs a reusable local worktree, needs to report pool occupancy, or needs to return a
completed worktree to the pool. Root `AGENTS.md` remains the source of truth for startup, commits, validation, and
final integration.

## Model

- The primary checkout is the developer's integration and review workspace. This manager never resets, switches, or
  releases it.
- `main` is the stable baseline. Numbered directories under `.worktrees/`, such as `.worktrees/01`, are reusable agent workspaces, not
  feature branches or topic ownership records. Each pool slot has a local-only marker branch such as `main-01`.
- Every acquired task gets one globally unique `feature/*`, `fix/*`, `refactor/*`, `docs/*`, `chore/*`, `build/*`, or
  `ci/*` branch. Do not create new `feat/*` or `wt-*` branches.
- An agent may commit its validated work but must not perform the final merge or cherry-pick. The developer integrates
  it in the primary checkout.

## Commands

Run the helper from any checkout in the repository:

```bash
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py status
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py acquire feature/runtime-target
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py release --confirm-integrated <commit-or-ref>
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py reconcile --confirm 01 02 03
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py relocate --confirm
```

`status` reports every registered worktree, its branch or detached ref, uncommitted-change count, derived state, and
pool baseline state. A clean recognized pool slot is `available` even when its cached baseline is stale; `baseline=stale`
indicates that the next acquire will refresh it.

`acquire` serializes pool allocation, fetches `origin`, selects the lowest clean reusable numbered pool directory, or
creates the next one. A stale detached legacy slot is reusable only when its HEAD is an ancestor of `origin/main`.
Before creating the task branch, the manager restores the slot to its local-only `main-XX` marker branch at the current
`origin/main`, then creates the unique task branch and reapplies the tracked `.worktree-shared.json` links.

`release` is a two-step developer-controlled operation. Without `--confirm-integrated` it only prints the review
summary. With the confirmation ref it requires a clean task worktree, fetches the current baseline, restores the
corresponding local-only `main-XX` marker branch, and deletes only the local task branch. It never merges, cherry-picks,
force-pushes, deletes a remote branch, or discards an unconfirmed task branch.

`reconcile --confirm [<slot> ...]` is a developer-confirmed migration for clean pool slots. It converts old detached
slots to their `main-XX` marker branches and refreshes them to `origin/main`. It preflights all selected slots and refuses
to touch dirty, task-branch, or divergent slots. `main-XX` branches are local pool markers, have no upstream, and must
never be passed to `$graft-push`.

`relocate --confirm` is a one-time developer-approved migration from legacy sibling directories such as
`<repo>-wt-01` into `<repo>/.worktrees/01`. It refuses to run unless every legacy pool slot is clean and at `origin/main`,
all destinations are free, and shared-link targets are safe to rebuild. Unrelated changes in the primary integration
checkout do not block relocation.

## Guardrails

- Do not use `git worktree remove` or `git worktree add` as the normal lifecycle; the helper only creates a missing
  numbered pool slot.
- Do not manually delete or push `main-XX` marker branches; they are managed slot identities.
- Pool-mutating commands are serialized by a repository-local lock under the Git common directory.
- Do not use a worktree path as an active-topic identity. Recovery records name the topic and current task branch only
  when that information is useful for resumption.
- Numbered agent worktrees are non-runtime environments: do not start frontend/backend services, development servers,
  Docker/Compose stacks, or other long-running runtime processes from them.
- Do not apply SQL migrations or execute state-changing database operations from a worktree. Static migration
  validation, checksum generation, tests, builds, and lint remain allowed when they do not target a live database.
- Runtime services and migration application belong to the developer-owned primary checkout with the necessary user or
  developer approval. Stop every process started for a task before releasing its worktree.
- Atlas migrations, generated code, OpenAPI clients, lock files, and snapshots are linear resources. Agents may edit
  their sources, but the developer generates and resolves final artifacts in the integration workspace.
- A one-time repository relocation is a separate, developer-approved migration after every legacy pool slot is clean
  and pushed. Use `relocate --confirm`; it is not a `release` operation.
