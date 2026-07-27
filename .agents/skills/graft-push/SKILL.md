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
5. If the push follows an active `$graft-pr-review` remediation run, confirm the latest PR finding inventory was built
   exhaustively and still includes `Outside diff range comments (N)`, `Nitpick comments (N)`, and other folded
   latest-review sections in scope; do not treat push as permission to leave those findings informal or unclassified.
6. When the user explicitly triggers `$graft-push`, treat it as permission to finish the local push path end to end:
   - if the branch is blocked on a bounded local issue, such as missing commit, generated-artifact drift, stale
     snapshots, local validation failure, or hook failure, repair it continuously only when its root cause is diagnosed
     and directly addressed, ownership is unambiguous, every file and hunk remains inside the confirmed scope, and
     authority and behavior are unchanged
   - invoke the root `AGENTS.md` `Repair Confirmation Interaction Contract` whenever a repair misses any
     continuous-repair condition. Present its `Repair required` proposal through native structured approval when available; otherwise stop and use its next-turn `1 / 2 / 3 / 4` fallback with all four visible option descriptions. Only `execute_repair` authorizes that proposed repair and its subsequent commit or push
   - report when the branch or destination is ambiguous, the failure has no concrete repair proposal, or the required
     validation is infeasible
7. When the push follows PR-review remediation, do not reply to a PR thread or update the managed review ledger before
   the remote branch ref has been verified to resolve to the exact local `HEAD`. A local commit, successful pre-push
   hook, or accepted `git push` process is not publication proof.

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
   - `type` should use an established repository prefix such as `feature`, `fix`, `refactor`, `docs`, `chore`, `build`,
     or `ci`
   - `topic-or-scope` must be lowercase kebab-case and summarize the commits that are about to be pushed
   - reject the push when the branch type is outside the established prefixes, the topic is not lowercase kebab-case,
     or the topic does not describe the local commit range being pushed
   - reject `feat/*`, `wt-*`, stale names, and unrelated names; generic `wt-*` placeholders are naming or scope
     failures, not advisory style suggestions
   - compare the branch name with the commits selected in step 1 and reject the push when that relationship cannot be
     established; report the mismatch before choosing a rename target
   - reject local-only worktree pool marker branches such as `main-01`; push must run from the primary integration
     checkout or a legal task branch, never from a reusable pool marker
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
   - if the failure is a local hook, reproduce the exact hook and diagnose its repair. Repair it continuously only
     when every continuous-repair condition holds; otherwise present the required structured proposal
   - use the root `Repair Confirmation Interaction Contract` for any bounded local blocker that does not meet every
     continuous-repair condition. `show_detailed_diff` shows the patch and repeats the native choice control;
     `continue_current_scope` and `cancel_workflow` stop repair work, and `execute_repair` resumes this push workflow
     only for the declared scope
   - if the branch contains `$graft-pr-review` fixes, preserve the review run's exhaustive finding-disposition
     requirement; a successful push does not downgrade `Outside diff range comments` or other folded review findings to
     optional
6. Push safely:
   - prefer the existing upstream when configured
   - if the branch was renamed for push hygiene, use the renamed branch for the upstream mapping
   - otherwise use an explicit `git push --set-upstream origin <branch>`
   - do not auto-delete the old remote branch after a rename unless the user explicitly asks
   - do not use force push unless the user explicitly asks and the repository state justifies it
7. Verify remote publication before any PR-review write:
   - run `git ls-remote --exit-code origin refs/heads/<current-branch>` and compare the returned SHA with `git rev-parse HEAD`
   - only an exact match proves that the fixing commit is visible to reviewers; a mismatch, missing ref, or transport
     failure leaves PR threads and the managed ledger untouched
   - after exact-match confirmation, return to `$graft-pr-review` to reply to fixed findings and append the disposition
     ledger; never reverse this order
8. Report:
   - what blocked the push or what was pushed
   - whether a branch-name check ran, what commit range it used, and whether a rename happened
   - the branch and upstream involved
   - the remote ref and SHA used to prove publication before any PR-review write
   - any hook or remote command that was reproduced
   - the exact next retry command when the push is not completed

## Refusal Cases

Do not push when any of these are true:

* the current slice is still uncommitted and the user did not explicitly authorize the needed commit step
* ownership is mixed and the push would depend on an unsafe commit
* the branch fails the established type, lowercase kebab-case topic, or commit-range relevance check, including
  `feat/*`, `wt-*`, stale, or unrelated branch names
* the branch rename target or destination ref is ambiguous
* the only available path would require force push without explicit user approval
* the push would be used to imply closure of a `$graft-pr-review` run whose latest finding inventory still leaves
  `Outside diff range comments`, `Nitpick comments`, or other folded latest-review findings unclassified
* a PR-review reply or ledger write would occur before `git ls-remote --exit-code origin refs/heads/<current-branch>`
  proves the remote ref resolves exactly to `HEAD`

In these cases, explain the blocker and stop at the smallest safe next step.

## Example Triggers

* `$graft-push`
* `Push the current branch`
* `排查这次 push 失败`
* `推送这次已提交的改动`
