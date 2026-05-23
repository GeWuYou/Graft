# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

## Active Topics

- `openapi-contract-governance`
  - Purpose: hold the long-lived OpenAPI First contract-governance worktree for `server`/`web` spec ownership,
    generated TypeScript types, and CI drift control.
  - Tracking: `ai-plan/public/openapi-contract-governance/todos/openapi-contract-governance-tracking.md`
  - Trace: `ai-plan/public/openapi-contract-governance/traces/openapi-contract-governance-trace.md`
  - Recovery note: this topic runs from dedicated worktree `feat/wt-openapi-contract-governance` on branch
    `feat/wt-openapi-contract-governance`; standing ownership is limited to governance docs, OpenAPI planning, and
    contract/SDK coordination. It does not own plugin implementation slices.
