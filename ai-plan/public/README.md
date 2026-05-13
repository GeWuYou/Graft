# AI Plan Public Index

`ai-plan/public/README.md` is the shared startup index for `boot`. It should stay short, list only active topics, and
help the current branch or worktree land on the right recovery documents without scanning every public artifact.

## Boot Rules

1. Read this file before scanning `ai-plan/public/<topic>/`.
2. If the current branch or worktree appears in the map below, read the listed topics in priority order.
3. If there is no match, fall back to scanning active topic directories.
4. Ignore `ai-plan/public/archive/**` by default unless historical context is explicitly requested.

## Active Topics

- `mvp-extension-path`
  - Purpose: continue the MVP extension path across server core, platform registries, early plugins, and the web shell.
  - Tracking: `ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/traces/mvp-extension-path-trace.md`

## Worktree To Active Topic Map

- Branch: `feat/mvp-extension-path`
  - Priority 1: `mvp-extension-path`
