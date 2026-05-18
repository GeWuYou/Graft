# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

## Recovery Rules

1. Read this file only after startup preflight from the root `AGENTS.md`.
2. If the current branch or worktree appears in the map below, read the listed topics in priority order.
3. Read the parent topic tracking and trace files before reading any subtopic files.
4. If the parent topic defines subtopics, continue into the relevant `server` / `web` / other bounded subtopic based on
   the current task shape.
5. If there is no match, fall back to scanning active topic directories.
6. Ignore `ai-plan/public/archive/**` by default unless historical context is explicitly requested.

## Mapping Conventions

- Prefer recording both `Worktree` and `Branch` when a long-lived local worktree already exists.
- If a long-lived worktree has not been created yet, branch-only mapping is acceptable as a temporary state.
- If the repository root is temporarily carrying the active topic on a non-`main` branch and no dedicated long-lived
  worktree exists yet, record that branch-only mapping explicitly and update it again once the root returns to `main`
  or a dedicated worktree/topic pair is created.
- Short-lived branches for hotfixes or narrow fixes should not be added as default active-topic mappings.
- When a future topic is planned but its worktree has not been created yet, keep the current active-topic mapping stable
  and record the intended split in the relevant tracking documents first.

## Active Topics

- `multi-worktree-governance`
  - Purpose: govern the shared recovery truth and final post-compatibility ownership baseline while the repository root
    is currently carrying the topic on branch `refactor/web-module-boundaries`, before dedicated long-lived worktrees
    are created.
  - Tracking: `ai-plan/public/multi-worktree-governance/todos/multi-worktree-governance-tracking.md`
  - Trace: `ai-plan/public/multi-worktree-governance/traces/multi-worktree-governance-trace.md`
  - Recovery note: use this topic only for shared baseline governance on the current repository root; once a
    long-lived worktree/topic pair is actually created, map that worktree to its own active topic instead of keeping
    feature recovery on the root branch.

## Branch / Worktree To Active Topic Map

- Branch: `refactor/web-module-boundaries`
  - Priority 1: `multi-worktree-governance`
  - Note: the repository root is currently checked out to this branch and is the only active worktree reported by
    `git worktree list`; update this mapping when the root returns to `main` or when the first dedicated worktree/topic
    pair is created.
