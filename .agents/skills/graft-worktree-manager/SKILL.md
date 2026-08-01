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
- An agent may commit its validated work but must not perform the final merge or cherry-pick by default. After the
  developer explicitly authorizes the exact integration operation in the current task, the agent may perform it in the
  primary checkout without assuming final repository ownership.

## Commands

Run the helper from any checkout in the repository:

```bash
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py status
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py doctor
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py acquire feature/runtime-target
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py closeout
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py release --confirm-integrated <commit-or-ref>
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py repair --confirm 03
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py reconcile --confirm 01 02 03
python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py relocate --confirm
```

`status` reports registered slots plus recoverable and broken numbered-slot evidence. `doctor` adds the concrete reason.
A clean recognized pool slot is `available` even when its cached baseline is stale; `baseline=stale` indicates that the
next acquire will refresh it.

`acquire` serializes allocation, fetches `origin`, reuses the lowest clean registered slot, then the lowest safe
recoverable marker slot, then the lowest missing number. A marker-only slot is recoverable only when its `main-XX` ref
is an ancestor of `origin/main`; it is restored before the task branch is created. Any directory/registration mismatch
or divergent marker is `broken` and blocks allocation until explicitly repaired.

`acquire` writes a local lease in the Git common directory. After the task's owned changes are committed, validated, and
the worktree is clean, `graft-task-closeout` invokes `closeout` to mark the lease release-ready. The local lease is
operational metadata, not active-topic recovery truth.

`release` is a two-step developer-controlled operation. Without `--confirm-integrated` it only prints the review
summary. With the confirmation ref it requires a clean, release-ready leased task worktree, fetches the current
baseline, restores the corresponding local-only `main-XX` marker branch, and deletes only the local task branch.
Existing historical task branches without a lease remain releasable under the clean and integration-confirmation checks,
and are reported as `legacy-untracked` until the pool is fully migrated.

`reconcile --confirm [<slot> ...]` is a developer-confirmed migration for clean pool slots. It converts old detached
slots to their `main-XX` marker branches and refreshes them to `origin/main`. It preflights all selected slots and refuses
to touch dirty, task-branch, or divergent slots. `main-XX` branches are local pool markers, have no upstream, and must
never be passed to `$graft-push`.

`repair --confirm <slot>` restores only a safe `recoverable` marker slot. Other `broken` states are fail-closed: inspect
their `doctor` reason and resolve the directory, registration, or divergent branch explicitly before retrying acquire.

`relocate --confirm` is a one-time developer-approved migration from legacy sibling directories such as
`<repo>-wt-01` into `<repo>/.worktrees/01`. It refuses to run unless every legacy pool slot is clean and at `origin/main`,
all destinations are free, and shared-link targets are safe to rebuild. Unrelated changes in the primary integration
checkout do not block relocation.

## Guardrails

- Do not use `git worktree remove` or `git worktree add` as the normal lifecycle; the helper only creates a missing
  numbered pool slot.
- Do not manually delete or push `main-XX` marker branches; they are managed slot identities.
- Pool-mutating commands are serialized by a repository-local lock under the Git common directory.
- Do not declare an acquired task complete with owned uncommitted changes. A task that reached closeout must commit,
  validate, leave its worktree clean, and record manager `closeout` before developer integration and `release`.
- Do not use a worktree path as an active-topic identity. Recovery records name the topic and current task branch only
  when that information is useful for resumption.
- Numbered agent worktrees are non-runtime environments: do not start frontend/backend services, development servers,
  Docker/Compose stacks, or other long-running runtime processes from them.
- Do not apply SQL migrations or execute state-changing database operations from a worktree. Static migration
  validation, checksum generation, tests, builds, and lint remain allowed when they do not target a live database.
- Runtime services and migration application belong to the developer-owned primary checkout with the necessary user or
  developer approval. Stop every process started for a task before releasing its worktree.
- Atlas migrations, non-OpenAPI generated code, lock files, and snapshots remain linear resources. Agents may edit
  their sources, but the developer generates and resolves final artifacts in the integration workspace.
- OpenAPI source and its deterministic generated artifacts are one task-owned contract closure. Agents must generate,
  validate, and commit the affected source, bundle, server bindings, runtime embedded bundle, and web schema in the
  task branch by running `just generate`, `just openapi-check`, and the task class's normal completion validation. If
  parallel branches conflict, the developer resolves canonical source first. The merged canonical OpenAPI source then
  drives regeneration and rerun of the OpenAPI checks instead of hand-merging generated files.
- A one-time repository relocation is a separate, developer-approved migration after every legacy pool slot is clean
  and pushed. Use `relocate --confirm`; it is not a `release` operation.
