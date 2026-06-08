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
helper remains the deterministic fallback and the only scripted path for idempotent PR creation/update and guarded
auto-merge handling.

## Preconditions

1. Ensure the current turn already has the startup receipt required by `AGENTS.md`.
2. Read `AGENTS.md` `4.1 Startup Governance`, `11. Git Workflow Rules`, and `12. Automation and CI/CD Rules`.
3. Confirm the PR trigger is valid:
   - the user explicitly requested a PR for the current branch
   - or the current slice is blocked on missing PR state and the user asked to diagnose or create it
4. If the current branch is not yet safely pushed to its intended upstream, route through `graft-push` first instead of
   creating a PR from ambiguous local-only state.

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
5. Keep updates idempotent:
   - if the PR already exists, only patch managed metadata or explicitly requested title/base fields
   - preserve user-authored PR body content outside the managed block
6. Treat auto-merge as a separate guarded action:
   - only attempt it when both `--enable-auto-merge` and `--confirm-automerge` are provided
   - if the target base branch has no detectable protection or required-check signal, report `would enable auto-merge`
     instead of changing GitHub state
   - otherwise enable auto-merge using the repository default merge method unless the user explicitly overrides it
7. Report:
   - whether the PR was created, reused, updated, or blocked
   - the PR number and URL when available
   - the head branch and base branch involved
   - the auto-merge disposition and any blockers

## Commands

- Dry run:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --dry-run`
- Create or reuse PR for the current branch:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py`
- Write machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --format json`
- Save machine-readable output:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --json-output /tmp/graft-pr.json`
- Attempt guarded auto-merge enablement:
  - `python3 .agents/skills/graft-pr-create/scripts/ensure_pr.py --enable-auto-merge --confirm-automerge`

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
