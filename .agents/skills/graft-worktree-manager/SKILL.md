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
- `main` is the stable baseline. Numbered directories under `/.worktrees/`, such as `/.worktrees/01`, are reusable agent workspaces, not
  feature branches or topic ownership records.
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
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py relocate --confirm
```

`status` reports every registered worktree, its branch or detached ref, uncommitted-change count, and derived state.
`available` means a clean numbered pool directory at the current `main` baseline. It is not a persisted lease.

`acquire` fetches `origin`, selects the lowest available numbered pool directory, or creates the next one. It refuses a
dirty directory and refuses branch names that already exist locally or on `origin`. It creates the task branch from
`origin/main` and reapplies the tracked `.worktree-shared.json` links.

`release` is a two-step developer-controlled operation. Without `--confirm-integrated` it only prints the review
summary. With the confirmation ref it requires a clean task worktree, returns it to `main` (or detached `origin/main`
when `main` is already checked out elsewhere), and deletes only the local task branch. It never merges, cherry-picks,
force-pushes, deletes a remote branch, or discards an unconfirmed task branch.

`relocate --confirm` is a one-time developer-approved migration from legacy sibling directories such as
`<repo>-wt-01` into `/.worktrees/01`. It refuses to run unless every legacy pool slot is clean and at `origin/main`,
all destinations are free, and shared-link targets are safe to rebuild. Unrelated changes in the primary integration
checkout do not block relocation.

## Guardrails

- Do not use `git worktree remove` or `git worktree add` as the normal lifecycle; the helper only creates a missing
  numbered pool slot.
- Do not use a worktree path as an active-topic identity. Recovery records name the topic and current task branch only
  when that information is useful for resumption.
- Atlas migrations, generated code, OpenAPI clients, lock files, and snapshots are linear resources. Agents may edit
  their sources, but the developer generates and resolves final artifacts in the integration workspace.
- A one-time repository relocation is a separate, developer-approved migration after every legacy pool slot is clean
  and pushed. Use `relocate --confirm`; it is not a `release` operation.
