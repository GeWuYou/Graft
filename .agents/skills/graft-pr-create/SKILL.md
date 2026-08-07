---
name: graft-pr-create
description: Repository-specific pull-request creation workflow for Graft. Use when the user explicitly wants a PR created or reconciled for the current branch, or when the agent needs to safely create, reuse, or diagnose the current branch PR against the repository default branch without bypassing push, commit, or auto-merge safety gates.
---

# Graft PR Create

Use this skill when the task needs a GitHub pull request for the current `Graft` branch rather than only a local
commit or push.

Shortcut: `$graft-pr-create`

Treat root `AGENTS.md` as the PR-governance source of truth. This skill does not bypass commit, push, ownership, or
validation rules.

GitHub MCP may be used for read-only repository and PR discovery when it is available in `codex mcp list`. The Python
helper remains the deterministic fallback and write path for idempotent PR creation/update and guarded auto-merge
handling; it does not generate PR prose.

## Preconditions

1. Ensure the current turn already has the startup receipt required by `AGENTS.md`.
2. Read `AGENTS.md` `4.1 Startup Governance`, `11. Git Workflow Rules`, and `12. Automation and CI/CD Rules`.
3. Confirm the PR trigger is valid:
   - the user explicitly requested a PR for the current branch
   - or the current slice is blocked on missing PR state and the user asked to diagnose or create it
4. If the current branch is not yet safely pushed to its intended upstream, route through `graft-push` first instead of
   creating a PR from ambiguous local-only state.
5. Before writing a PR body, inspect the merge-base diff, changed-file inventory, commit history, affected modules,
   `AGENTS.md`, `DESIGN.md`, relevant `ai-plan/` design/roadmap/topic material, matching RFCs or ADRs, and issue
   references discoverable from commits or PR metadata.

## Workflow

1. Inspect repository state:
   - `git branch --show-current`
   - `git status --short`
   - current upstream mapping when it exists
2. Resolve GitHub repository state:
   - repository default branch
   - merge-method capabilities
   - whether GitHub auto-merge is allowed
   - branch-protection or required-check signals on the target base branch
   - prefer GitHub MCP for quick read-only discovery when available; fall back to `ensure_pr.py` / GitHub API helper
3. Resolve the current branch PR state:
   - no matching open PR: create one against the default branch unless `--base` overrides it
   - one matching open PR: reuse it
   - multiple candidate open PRs: fail closed and report the ambiguity
4. Keep PR scope explicit:
   - current branch only
   - default base branch unless explicitly overridden
   - do not push, amend, or create commits implicitly
5. Write an evidence-based PR description before invoking the helper:
   - use the exact English headings: `Summary`, `Motivation`, `Scope`, `Key Changes`, `Architecture Impact`, `User
     Impact`, `Compatibility`, `Testing`, `Risks`, `Follow-up Work`, `Reviewer Notes`, and `Checklist`
   - explain engineering intent and behavior by capability, not by file list
   - state whether the implementation follows the discovered architecture/design documents and explain any authority,
     contract, boundary, lifecycle, state-model, event-flow, or dependency change
   - list only validation, screenshots, migrations, compatibility claims, issue references, and risks supported by
     evidence; do not invent missing facts
   - explicitly state when there is no architectural contract or user-visible behavior change, when the diff proves it
   - save the complete Markdown body to a temporary file outside the repository
6. Keep updates idempotent:
   - invoke `ensure_pr.py --description-file <temporary-file>`; the argument is required and rejects empty content
   - the helper replaces only its description and metadata marker regions, preserving human and reviewer-tool content
     outside those regions
   - `--body-file` is removed and must not be used
7. Treat auto-merge as a separate guarded action:
   - only attempt it when both `--enable-auto-merge` and `--confirm-automerge` are provided
   - if the target base branch has no detectable protection or required-check signal, report `would enable auto-merge`
     instead of changing GitHub state
   - otherwise enable auto-merge using the repository default merge method unless the user explicitly overrides it
8. Report:
   - whether the PR was created, reused, updated, or blocked
   - the PR number and URL when available
   - the head branch and base branch involved
   - the auto-merge disposition and any blockers

## PR Description Standard

Write the following Markdown sections in this order. Omit a section only when its required information cannot be
inferred without speculation; never fill a section with generic boilerplate.

- `# Summary`: three to eight bullets stating the product or engineering result, not filenames.
- `# Motivation`: the prior limitation, defect, inconsistency, missing capability, or design goal.
- `# Scope`: affected business capabilities, modules, contracts, or runtime areas.
- `# Key Changes`: `##` subsections grouped by logical behavior, each explaining intent and resulting behavior.
- `# Architecture Impact`: authority, boundary, contract, state, event, or dependency changes. State `No architectural
  contract changes.` when the evidence confirms none.
- `# User Impact`: visible workflows, UI, performance, or an evidenced absence of user-facing change.
- `# Compatibility`: backward compatibility, API compatibility, migrations, and configuration effects.
- `# Testing`: only commands or verification that were actually run. Do not claim a test, build, manual check, or
  screenshot without evidence.
- `# Risks`: concrete review or operational risks, including a justified low-risk assessment where appropriate.
- `# Follow-up Work`: intentionally deferred work only.
- `# Reviewer Notes`: the highest-value review targets, especially authority, contract, compatibility, or behavior
  changes.
- `# Checklist`: context-appropriate unchecked items for self-review, documentation, tests, breaking changes, and
  migrations.

The description must be intelligible without reading the diff. It must not use generic phrases such as "minor cleanup",
"miscellaneous fixes", "update code", or "improve implementation". Do not fabricate screenshots, issue references,
or test outcomes.

## Commands

- Dry run after writing `/tmp/graft-pr-description.md`:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --dry-run --description-file /tmp/graft-pr-description.md`
- Create or reuse PR for the current branch:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md`
- Write machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --format json`
- Save machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --json-output /tmp/graft-pr.json`
- Attempt guarded auto-merge enablement:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --description-file /tmp/graft-pr-description.md --enable-auto-merge --confirm-automerge`

## Refusal Cases

Do not create or modify a PR when any of these are true:

* the current branch is detached or matches the repository default branch
* the branch has no remote upstream and the requested PR would depend on an implicit push
* multiple open PRs match the same branch and the correct target cannot be resolved safely
* auto-merge would require bypassing the guarded confirmation path
* the requested base/head update would overwrite ambiguous user intent

In these cases, explain the blocker and stop at the smallest safe next step.

## Example Triggers

* `$graft-pr-create`
* `Create a PR for this branch`
* `为当前分支创建 PR`
* `检查当前分支 PR 和 auto-merge 条件`
