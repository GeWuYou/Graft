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

## Active Topics

- `mvp-extension-path`
  - Purpose: continue the MVP extension path across server core, platform registries, early plugins, and the web shell.
  - Tracking: `ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/traces/mvp-extension-path-trace.md`
  - Subtopics:
    - `server`: `ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
    - `web`: `ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
  - Recovery note: always read the parent `mvp-extension-path` entry first, then continue into the relevant subtopic.

## Worktree To Active Topic Map

- Branch: `feat/mvp-extension-path`
  - Priority 1: `mvp-extension-path`
