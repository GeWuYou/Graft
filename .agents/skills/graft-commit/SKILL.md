---
name: graft-commit
description: Repository-specific scoped commit workflow for Graft. Use when the user explicitly wants the current validated task slice committed, or when `graft-task-closeout` decides a validated owned scope should be committed, and the agent needs to classify ownership, verify scope, choose a compliant Conventional Commit message, and create a safe git commit without bundling unrelated changes.
---

# Graft Commit

Use this skill when the user explicitly asks to commit the current `Graft` worktree, for example with
`$graft-commit`, `commit this slice`, or `提交当前这次改动`, or when `graft-task-closeout` concludes that the current
validated owned scope should be committed before handoff. An explicit `$graft-commit` authorizes continuous, scoped
batch completion for every commit-eligible tracked and untracked entry captured in the initial worktree inventory. The
inventory is an upper boundary: it excludes later-arriving changes, but is complete commit authority for its captured
entries. The workflow must finish with an empty `git status --short` unless a concrete safety, policy, ownership, or
validation blocker is reported.

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
6. When the user explicitly triggers a bare `$graft-commit`, capture the complete initial working-tree inventory as an
   upper boundary, treat its commit-eligible tracked and untracked entries as explicitly user-confirmed and currently
   owned, and finish every resulting logical slice end to end without repeated confirmation:
   - treat a hook, static-analysis, test, formatting, style, or build failure as a commit-path issue to diagnose before
     deciding whether a repair is safe; a failing file outside the initial diff is not, by itself, proof that its repair
     is unsafe
   - before any repair edit, staging, or repair commit, apply the root `AGENTS.md` `Repair Confirmation Interaction
     Contract`; present its `Repair required` proposal through native structured approval when available. When the
     runtime lacks that control, use the root contract's stopped next-turn `1 / 2 / 3 / 4` fallback, including its
     four visible option descriptions; never ask a binary approval question
   - after every commit, re-check `git status --short` and continue until every captured entry is committed and the
     worktree is clean
   - apply that contract when ownership becomes ambiguous, a repair requires an extra commit, or the necessary repair
     widens the slice; report instead when required validation is infeasible or no concrete repair proposal can be formed

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
   - capture the complete initial worktree inventory as an upper boundary, then group every commit-eligible captured
     entry into independently validated logical slices
   - the bare `$graft-commit` inventory grants commit authority to every captured entry; files that appear after the
     inventory was captured are never in scope
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
   - if validation fails, diagnose the concrete issue and use the root `Repair Confirmation Interaction Contract`
     before changing code. Only `execute_repair` permits the declared repair; `show_detailed_diff` shows a patch and
     invokes the same native choice control again, while the other choices forbid the repair
4. Stage only the captured logical slice:
   - a bare `$graft-commit` authorizes `git add .` or `git add -A` only after confirming no files appeared after the
     initial inventory; otherwise stage exact captured paths
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
   - continue through every captured, validated logical slice without another commit confirmation until `git status
     --short` is empty; stop only at an ambiguous, unvalidated, unsafe, or otherwise concrete blocker and report it
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
* the proposed commit message would violate the repository Conventional Commit rules
* the commit is being used to close or imply closure for a `$graft-pr-review` run whose latest inventory still leaves
  `Outside diff range comments`, `Nitpick comments`, or other folded latest-review findings unclassified

In these cases, explain the blocker and stop at the smallest safe next step.

## Example Triggers

* `$graft-commit`
* `Use $graft-commit for this slice`
* `提交这次已验证的改动`
* `Commit the current validated scope`
