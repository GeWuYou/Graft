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

- `rbac-further-development`
  - Purpose: hold the standalone recovery entry for the next bounded `rbac` implementation slices after the shared
    multi-worktree governance baseline was archived.
  - Tracking: `ai-plan/public/rbac-further-development/todos/rbac-further-development-tracking.md`
  - Trace: `ai-plan/public/rbac-further-development/traces/rbac-further-development-trace.md`
  - Recovery note: this topic does not own a dedicated worktree yet; create one only after the `rbac` slice keeps a
    stable owned scope at `server/plugins/rbac/**` plus `web/src/modules/rbac/**`.
- `monitor-server-status`
  - Purpose: hold the standalone recovery entry and design baseline for the future `monitor` plugin and `monitor`
    frontend module before any dedicated worktree is created.
  - Tracking: `ai-plan/public/monitor-server-status/todos/monitor-server-status-tracking.md`
  - Trace: `ai-plan/public/monitor-server-status/traces/monitor-server-status-trace.md`
  - Recovery note: this topic is design-only for now and should not be treated as source-implementation ownership until
    a later slice explicitly creates `server/plugins/monitor/**` and `web/src/modules/monitor/**`.

## Branch / Worktree To Active Topic Map

- Worktree: repository root
  - Branch: `main`
  - Active topic: none by default
  - Role: shared coordination point for active-topic governance only; feature recovery should enter through an explicit
    startup prompt naming one of the active topics above until a dedicated worktree/topic pair is created
  - Hotspot policy: shared hotspots such as `ai-plan/public/README.md` remain serialized governance slices and do not
    grant standing feature ownership to the root worktree
- Branch: `main`
  - Priority 1: `rbac-further-development`
  - Priority 2: `monitor-server-status`
  - Note: no dedicated long-lived worktree exists yet for either topic; treat this branch-only mapping as a fallback
    discovery hint and move each topic to its own explicit worktree mapping when dedicated worktrees are created.
