---
name: graft-commit
description: Repository-specific scoped commit workflow for Graft. Use when the user explicitly wants the current validated task slice committed, or when `graft-task-closeout` decides a validated owned scope should be committed, and the agent needs to classify ownership, verify scope, choose a compliant Conventional Commit message, and create a safe git commit without bundling unrelated changes.
---

# Graft Commit

Use this skill when the user explicitly asks to commit the current `Graft` worktree, for example with
`$graft-commit`, `commit this slice`, or `提交当前这次改动`, or when `graft-task-closeout` concludes that the current
validated owned scope should be committed before handoff. An explicit `$graft-commit` authorizes continuous, scoped
batch completion only for explicitly user-confirmed, currently owned slices captured in the initial worktree inventory;
that inventory is an upper boundary, not commit authority.

Treat root `AGENTS.md` as the commit-governance source of truth. This skill does not loosen ownership, staging, or
validation rules.

## Preconditions

1. Ensure the current turn already has the startup receipt required by `AGENTS.md`.
2. Read `AGENTS.md` `11. Git Workflow Rules` before staging or committing anything.
3. Confirm the commit trigger is valid:
   - either the user explicitly requested a commit
   - or `graft-task-closeout` decided the validated owned scope should be committed
4. If the correct validation scope is unclear, use `graft-validation-runner` before committing.
5. If the commit is part of an active `$graft-pr-review` run, confirm that the latest PR finding inventory is already
   exhaustive before staging:
   - `Outside diff range comments (N)`, `Nitpick comments (N)`, and other folded latest-review sections are still
     mandatory review scope, not optional follow-up
   - do not use an intermediate commit to imply PR-review closure while any verified finding from that inventory is
     still unclassified, including outside-diff findings
6. When the user explicitly triggers `$graft-commit`, capture the initial working-tree inventory as an upper boundary,
   identify the explicitly user-confirmed, currently owned slices in it, and finish only those slices end to end without
   repeated confirmation:
   - if required validation fails on a concrete issue inside an explicitly user-confirmed owned slice and the fix is
     reasonably bounded,
     fix that issue first, rerun validation, and continue the commit workflow without waiting for another user reminder
   - if the current slice is blocked by a local generated-artifact drift, stale snapshot, test expectation drift, or
     similar commit-path issue that is clearly inside an explicitly user-confirmed owned slice and can be repaired
     safely in the same slice,
     repair it first, rerun the required validation, and then continue to commit
   - treat a hook, static-analysis, test, formatting, style, or build failure as a commit-path issue to diagnose before
     refusing the commit; a failing file outside the initial diff is not, by itself, proof that its repair is unsafe
   - when the failure has a concrete, bounded repair with clear ownership, fix it, rerun the failing check and the
     required completion validation, then commit the repair in a separate scoped commit when it is logically distinct
     from the requested slice
   - after every commit, re-check `git status --short` and continue only with remaining explicitly user-confirmed,
     currently owned captured slices; stop after those slices are complete even when other changes remain in the worktree
   - stop and report instead of auto-fixing when a captured change is not explicitly user-confirmed and currently owned,
     ownership becomes ambiguous, the required validation is infeasible, or the necessary repair would widen into a new unsafe slice

## Workflow

1. Inspect `git status --short` and classify ownership using the three AGENTS scenarios:
   - clean working tree before task
   - dirty working tree but owned scope can be reliably separated
   - mixed or ambiguous ownership that cannot be safely separated
   - interpret status columns explicitly:
     - `M ` in the left column means the file is already staged in the Git index
     - ` M` in the right column means the file is modified in the working tree but not staged yet
     - `git diff --cached --name-only` only reports the Git index; if it is empty while `git status --short` shows
       ` M`, the problem is unstaged changes, not a missing diff
   - do not treat IDE changelist checkboxes, selected files, or review UI state as proof that changes are staged;
     confirm staging from Git itself before continuing
2. Define the commit scope before staging:
   - capture the complete initial worktree inventory as an upper boundary, then group only explicitly user-confirmed,
     currently owned changes into independently validated logical slices
   - include only files or hunks belonging to one explicitly user-confirmed, currently owned captured slice; presence
     in the inventory, user authorship, classifiability, or task relevance does not grant commit authority, and files
     that appear after the inventory was captured are never in scope
   - never treat task relevance alone as justification to merge unrelated logical slices into one commit
   - when the confirmed owned scope contains multiple independently validated logical slices, or one safe commit cannot
     cover the confirmed scope cleanly, split it into a batch plan of separate scoped commits
   - do not use batching to bypass mixed ownership, missing validation, broad staging, or an invalid commit message
3. Confirm validation is sufficient for the task class:
   - `server`: prefer `cd server && go run ./cmd/graft validate backend` for completion-state work
   - `web`: prefer `cd web && bun run check` for completion-state work
   - `cross-boundary`: validate both affected sides
   - `docs/automation`: run the strongest honest structural checks available
   - if the commit belongs to a `$graft-pr-review` remediation batch, validation alone is not enough; the review run
     must still preserve exhaustive finding disposition coverage, including `Outside diff range comments`
   - if validation fails on a bounded issue inside the current owned scope, repair that issue and rerun the required
     validation before deciding whether to refuse the commit
4. Stage only the confirmed owned scope:
   - do not use `git add .`, `git add -A`, or `git commit -am` unless the user explicitly asks to commit everything
   - when one file contains mixed ownership, stage only the owned hunks if they can be reliably separated
   - if the intended scope is currently unstaged, stage that exact scope first and then re-check with
     `git diff --cached --name-only` or `git status --short`; do not assume a previous push or IDE selection updated
     the Git index
5. Build the commit message from `AGENTS.md` rules:
   - format: `<type>(<scope>): <summary>`
   - title defaults to English
   - `scope` is required and explicit
   - avoid noise titles such as `wip`, `update`, or `fix typo`
   - ordinary non-merge and non-revert commits must include a real multiline body with at least one `- ` bullet
   - do not create an agent-authored ordinary commit with only a title or literal escaped control text like `\n`
6. Create the scoped commit(s):
   - default to one scoped commit for the current logical slice
   - if a batch plan is required, create each commit sequentially and re-check `git status --short` plus
     `git diff --cached --name-only` before each commit
   - continue through every explicitly user-confirmed, currently owned, validated captured slice without asking for
     another commit confirmation; stop after those slices are complete, or at an ambiguous, unvalidated, or unsafe batch,
     and report the committed batches plus the uncommitted blocker
7. Report:
   - the committed scope
   - the validation command(s) used
   - each final commit title and short SHA
8. If the commit is being made as part of a task handoff, report the exact next-task startup prompt that should be
   used for the next turn.

## Refusal Cases

Do not commit when any of these are true:

* ownership is mixed and cannot be confidently separated
* the commit trigger is valid but the task still lacks the required validation and that validation is still feasible
* the working tree contains unrelated changes that would be staged only by using broad git add patterns
* the proposed commit message would violate the repository Conventional Commit rules
* the commit is being used to close or imply closure for a `$graft-pr-review` run whose latest inventory still leaves
  `Outside diff range comments`, `Nitpick comments`, or other folded latest-review findings unclassified

In these cases, explain the blocker and stop at the smallest safe next step.

## Example Triggers

* `$graft-commit`
* `Use $graft-commit for this slice`
* `提交这次已验证的改动`
* `Commit the current validated scope`
