---
name: graft-worktree-interactive-cleanup
description: Inspect numbered Graft worktrees, compare task branches with the integration checkout, obtain explicit confirmation, and safely release and rebuild only stale or already-integrated slots. Use when cleaning abandoned agent worktrees, resolving unreleased feature slots, or deciding whether a numbered worktree can return to its main-XX pool marker.
---

# Graft Worktree Interactive Cleanup

Use this skill for cleanup decisions involving `.worktrees/<slot>` directories. Keep
`graft-worktree-manager` as the lifecycle authority; this skill adds evidence gathering,
interactive confirmation, and the non-ancestor-but-equivalent patch check.

## Inspect

1. Establish the repository startup receipt from root `AGENTS.md`. Treat the task as
   `docs/automation`, with recovery source `none` unless the user names a topic.
2. Run the manager status command and inspect only the user-named numbered slots.
3. Record each slot's current branch, HEAD, upstream, clean/dirty state, conflict state,
   divergence from the explicit integration HEAD and `origin/main`, and manager lifecycle state.
4. Never inspect `main-XX` pool markers as task branches, and never touch an unnamed slot.

## Classify

Classify a task branch before asking for cleanup confirmation:

- `integrated`: the branch is an ancestor of the integration HEAD.
- `equivalent-integrated`: the branch is not an ancestor, but `git cherry <integration-head> <branch>`
  marks every branch-only commit with `-`; record the matching patch IDs and integration commits.
- `unreleased`: any branch-only commit is not patch-equivalent to the integration HEAD.
- `unsafe`: the worktree is dirty, has conflicts, is not a registered numbered slot, or its
  branch cannot be identified.

Do not release `unreleased` or `unsafe` slots. A deleted upstream branch alone does not prove
that local work is disposable.

For leases created by the current manager, `release-ready` is also required before release. A historical
`legacy-untracked` task branch may use the existing clean-and-integrated release path, but must be reported as such.

## Confirm And Mutate

Present the classification and the exact slots/branches to the user before mutation. Require
explicit confirmation for each cleanup batch, including `equivalent-integrated` slots.

- For `integrated` slots, use:
  `python3 .agents/skills/graft-worktree-manager/scripts/worktree_manager.py release --worktree <slot> --confirm-integrated <integration-ref>`.
- For `equivalent-integrated` slots, do not force the normal release command through a false
  ancestry claim. Recheck cleanliness and patch-ID equality, switch the slot to its matching
  `main-XX` marker at `origin/main`, unset its upstream, then delete only the confirmed local
  task branch. Report the equivalent integration commit and the fact that the history was not
  ancestor-related.
- After releasing, run `reconcile --confirm <slots>` to rebuild the selected pool slots from
  the current `origin/main` and restore shared links.

Never use `git worktree remove` for normal pool cleanup, never delete a task branch before its
worktree has switched away from it, and never remove a worktree's administrative metadata by
hand. Preserve the primary integration checkout and all unselected slots.

## Verify

After mutation, verify all of the following:

- selected slots are registered and point to `main-XX` branches;
- selected worktrees are clean and at the current `origin/main`;
- deleted task branches no longer exist locally;
- no unrelated worktree changed;
- the manager status command reports the selected slots as `available`.

Report any remote branch marked `gone` separately; this does not authorize deleting a local
branch without the integration/equivalence evidence and user confirmation.
