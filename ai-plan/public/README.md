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
- Every long-lived worktree mapping must stay explicit about:
  - `Worktree`
  - `Branch`
  - `Active topic`
  - whether the worktree owns a plugin or governance slice
  - which shared hotspots are still allowed to be touched from that worktree
- Short-lived branches for hotfixes or narrow fixes should not be added as default active-topic mappings.
- When a future topic is planned but its worktree has not been created yet, keep the current active-topic mapping stable
  and record the intended split in the relevant tracking documents first.

## Shared Hotspot Handling

- Long-lived worktrees should default to one bounded owned scope such as one plugin or one governance slice.
- Shared hotspots are not default owned scopes for feature worktrees; they stay exceptional and must be recorded in the
  active topic tracking before a new worktree starts touching them.
- While the repository root is the only active worktree, it remains the temporary coordination point for shared
  governance and hotspot policy.
- Once a dedicated long-lived feature worktree exists, move feature-specific recovery out of the root governance topic
  and into that worktree's own active topic instead of letting the root keep both histories.
- If a task needs both plugin-owned files and a shared hotspot, prefer splitting the hotspot update into its own bounded
  slice or serializing that work rather than letting multiple long-lived worktrees co-own the hotspot by default.

## Active Topics

- `multi-worktree-governance`
  - Purpose: govern the shared recovery truth and final post-compatibility ownership baseline while the repository root
    is currently carrying the topic on branch `refactor/server-module-boundaries`, before dedicated long-lived worktrees
    are created.
  - Tracking: `ai-plan/public/multi-worktree-governance/todos/multi-worktree-governance-tracking.md`
  - Trace: `ai-plan/public/multi-worktree-governance/traces/multi-worktree-governance-trace.md`
  - Recovery note: use this topic only for shared baseline governance on the current repository root; once a
    long-lived worktree/topic pair is actually created, map that worktree to its own active topic with its own
    tracking/trace pair and keep the root governance topic focused on shared baseline policy and hotspot coordination.

## Branch / Worktree To Active Topic Map

- Worktree: repository root
  - Branch: `refactor/server-module-boundaries`
  - Active topic: `multi-worktree-governance`
  - Role: temporary shared-baseline governance only; not a permanent home for future feature-owned recovery history
  - Hotspot policy: may document and coordinate shared hotspots while it is the only active worktree, but future
    dedicated feature worktrees should not treat the root as their default recovery entry once they exist
- Branch: `refactor/server-module-boundaries`
  - Priority 1: `multi-worktree-governance`
  - Note: the repository root is currently checked out to this branch and is the only active worktree reported by
    `git worktree list`; update this mapping when the root returns to `main` or when the first dedicated worktree/topic
    pair is created.
