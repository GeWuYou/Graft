---
name: graft-push
description: Repository-specific push workflow for Graft. Use when the user explicitly wants the current branch pushed, or when a local push path is blocked and the agent needs to diagnose hook failures, upstream ambiguity, or remote rejection without inventing a second commit workflow.
---

# Graft Push

Use this skill when the user explicitly asks to push the current `Graft` branch, for example with
`$graft-push`, `push this branch`, `推送当前分支`, or when the current local push path is blocked and the agent needs to
diagnose the blocker before the user retries.

Treat root `AGENTS.md` as the push-governance source of truth. This skill does not bypass commit, ownership, or
validation rules.

## Preconditions

1. Ensure the current turn already has the startup receipt required by `AGENTS.md`.
2. Read `AGENTS.md` `13. Git Workflow Rules` before pushing or diagnosing a push failure.
3. Confirm the push trigger is valid:
   - either the user explicitly requested a push
   - or the current task is blocked on a local push failure that the user asked to diagnose
4. If the current slice is not yet safely committed, route through `graft-commit` first instead of pushing mixed or
   uncommitted work.

## Workflow

1. Inspect repository state before pushing:
   - `git status --short`
   - current branch or detached HEAD state
   - current upstream mapping when it exists
   - the local commit range that would actually be pushed:
     - prefer `git log --oneline @{upstream}..HEAD` when an upstream exists
     - otherwise compare `HEAD` against the merge-base with the intended base branch, normally `main`
2. Validate branch-name fit before pushing:
   - branch names must follow `<type>/<topic-or-scope>`
   - `type` should use an established repository prefix such as `feat`, `fix`, `refactor`, `docs`, `chore`, `build`,
     or `ci`
   - `topic-or-scope` must be lowercase kebab-case and summarize the commits that are about to be pushed
   - avoid stale names, unrelated names, or generic `wt-*` placeholders unless the branch is intentionally the tracked
     long-lived topic/worktree branch
   - if the current branch name does not fit the local-only commit range well, rename the local branch before pushing
     and continue with the renamed branch as the only push target
3. Classify the blocker or next action:
   - uncommitted or unstaged local scope
   - local Husky / hook failure
   - missing or wrong upstream branch
   - remote rejection, branch protection, or non-fast-forward
   - network or authentication failure
4. Keep push scope explicit:
   - push the current confirmed branch only
   - do not create extra commits, amend history, or push unrelated refs unless the user explicitly asks
   - if detached HEAD is intentional, require an explicit destination ref before pushing
5. Reuse repository truth before diagnosing remote issues:
   - if a commit is missing, use `graft-commit`
   - if local validation is the real blocker, use `graft-validation-runner`
   - if the failure is a local hook, reproduce the exact hook and fix that path first
6. Push safely:
   - prefer the existing upstream when configured
   - if the branch was renamed for push hygiene, use the renamed branch for the upstream mapping
   - otherwise use an explicit `git push --set-upstream origin <branch>`
   - do not auto-delete the old remote branch after a rename unless the user explicitly asks
   - do not use force push unless the user explicitly asks and the repository state justifies it
7. Report:
   - what blocked the push or what was pushed
   - whether a branch-name check ran, what commit range it used, and whether a rename happened
   - the branch and upstream involved
   - any hook or remote command that was reproduced
   - the exact next retry command when the push is not completed

## Refusal Cases

Do not push when any of these are true:

* the current slice is still uncommitted and the user did not explicitly authorize the needed commit step
* ownership is mixed and the push would depend on an unsafe commit
* the branch rename target or destination ref is ambiguous
* the only available path would require force push without explicit user approval

In these cases, explain the blocker and stop at the smallest safe next step.

## Example Triggers

* `$graft-push`
* `Push the current branch`
* `排查这次 push 失败`
* `推送这次已提交的改动`
