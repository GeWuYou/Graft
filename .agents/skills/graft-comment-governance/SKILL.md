---
name: graft-comment-governance
description: Govern high-value Chinese comments in Graft handwritten Go, TypeScript, and Vue code. Use when adding, updating, reviewing, or removing code comments, package documentation, Vue component documentation, TSDoc, or when closing any task that changes .go, .ts, or .vue files.
---

# Graft Comment Governance

Use the repository comment standard as the semantic authority. Keep comments in Chinese, preserve standard Go doc-comment form, and record information that implementation and types cannot express.

## Preflight

1. Read `AGENTS.md`, the applicable `server/AGENTS.md` or `web/AGENTS.md`, and `ai-plan/design/governance/ai/代码注释与模块文档规范.md`.
2. Classify files as handwritten source, generated code, third-party code, migration artifact, or build output. Only handwritten source is in scope unless the task explicitly expands it.
3. Inspect the implementation, callers, tests, and existing comments before changing a comment. Treat stale comments as defects; do not preserve them because they are old.
4. For every code-task closeout, review the changed Go, TypeScript, and Vue files with this skill before reporting completion.

## Value Gate

Before adding a comment, answer both questions:

1. Can a maintainer infer this information from the code and types alone? If yes, do not comment it.
2. Does it explain a design reason, constraint, business rule, algorithm, or external-system behavior? If no, do not comment it.

Remove or rewrite comments that merely translate names, types, templates, or framework lifecycle calls. Do not introduce a comment-count or documentation-coverage target.

## Language Rules

### Go

* Write Chinese Go doc comments for every handwritten exported package, type, interface, function, method, const, and var; start each comment with its identifier.
* Document public responsibility, error semantics, lifecycle, concurrency, registration order, compatibility, or caller-visible constraints. Use independent lines for non-obvious locking, retry, cache, transaction, and shutdown decisions.
* Do not turn every parameter or field into a prose copy of its type. Comment fields only when their lifecycle, nullability, ownership, or sharing semantics are non-obvious.

### TypeScript

* Use `/** ... */` TSDoc/JSDoc for public interfaces, exported APIs, generic contracts, and asynchronous behavior only when types do not convey their business meaning or constraints.
* Describe validation preconditions, fallback or failure behavior, ownership, and stable semantics. Do not document obvious primitive fields, return types, or local variable names.

### Vue

* Give non-trivial SFCs a component-responsibility comment in `<script setup>` that identifies its data boundary or integration behavior.
* Comment Props only for UI/business meaning that types cannot express. Comment reactive state, watchers, subscriptions, and lifecycle hooks only for the reason, ownership, cleanup obligation, or external behavior.
* Never annotate template elements, translate `onMounted`, or explain self-evident refs and computed values.

## Delegation Gate

For a comment-only implementation or review batch, the main agent must verify that the orchestration layer selected and exposed the requested worker model before delegation. A model name written in a task prompt is not verification.

The target worker configuration is `model=gpt-5.6-luna` with `reasoning_effort=medium`. The orchestration layer must expose and verify both fields before delegation. If that configuration cannot be selected or verified, pause delegation and ask the user to choose one of: use the main agent's current model, provide a model configuration that the orchestrator can verify, or have the main agent take over. Do not dispatch a worker until this is resolved.

When the orchestration API supports full-history forking, do not combine `fork_context=true` with explicit model or
reasoning-effort overrides. Use an independent context when selecting the verified worker configuration, or inherit the
parent configuration without overrides.

Use non-overlapping roles for a large comment-governance batch:

* `comment-audit-agent`: read-only inventory and value classification.
* `governance-skill-agent`: `.agents/skills/**` and governance documentation only.
* `backend-comment-agent`: assigned handwritten `server/**` packages only.
* `frontend-comment-agent`: assigned handwritten `web/**` modules only.
* `review-agent`: read-only comparison of comments against final implementation.

Respect the available concurrency limit. The main agent owns batch ordering, integration, validation, and final acceptance.

## Commit Gate

Comment governance never grants commit authority by itself. A write-capable comment worker's dispatch contract must
state whether the worker may commit, but a scoped comment commit is permitted only when all of these preconditions hold:

* the current turn has the `AGENTS.md` startup receipt, including the applicable task class and authority summary;
* ownership is classified under the root Git workflow, and the exact files or hunks are separable from user, unknown,
  unrelated, and other-worker changes;
* the current worktree and branch state have been inspected, and the scope can be staged without broad staging or
  crossing the worktree's ownership boundary; and
* there is a legal repository commit trigger: the user explicitly requested a commit, or `graft-task-closeout`
  decided that the validated owned scope should be committed.

A comment review, worker dispatch, existing dirty worktree, or this skill's `Commit Gate` is not a commit trigger. When
the preconditions hold, the worker must run `$graft-commit` after its comment review and required validation pass; a
comment-only implementation is not complete merely because the source diff exists. `$graft-commit` remains subject to
the root `AGENTS.md` ownership, validation, staging, and mixed-change refusal rules.

Choose the commit scope using the normal `graft-commit` ownership rules:

* If the worker can prove exact file or hunk ownership, create the scoped comment commit.
* If the comment edits belong to the same owned task but cannot be separated from its implementation hunks, include them
  in that task's scoped commit through `$graft-commit`.
* If ownership or hunk separation is ambiguous, do not stage broadly or claim completion. Return a blocked commit result
  for the main agent to resolve.

When the worker cannot perform the commit because the orchestration layer did not grant authority, the main agent must
invoke `$graft-commit` after accepting the worker result if ownership and validation are sufficient.

## Closeout

`commit_status: not-needed` is valid only when no accepted handwritten Go, TypeScript, or Vue comment change exists in
the reported scope, or when the scope contains only generated, third-party, migration, or build paths explicitly
exempted by this skill. It must not be used to conceal a missing commit trigger, missing startup receipt, unresolved
ownership/worktree ambiguity, missing validation, or a commit that the worker was expected to make. If an accepted
comment change cannot be committed, use `commit_status: blocked` and include this structured field with both values:

```text
commit_blocked_reason:
- reason: <concrete blocker>
- missing_result: <specific commit, validation, ownership, or authorization result still required>
```

The `reason` and `missing_result` fields are mandatory for `blocked`; use `none` only for statuses where no blocker
exists.

Report:

```text
comment_governance:
- changed_files: <paths>
- decisions: added | updated | removed | none-needed
- value_categories: why | constraint | business-rule | algorithm | external-behavior
- exemptions: <generated/third-party/migration/build paths or none>
- delegation_model: <verified model, main-agent takeover, or not-applicable>
- commit_status: created | not-needed | blocked | pending-main-agent
- commit_blocked_reason:
  - reason: <concrete blocker or none>
  - missing_result: <specific missing result or none>
- commit_scope: <paths or hunks, or none>
- commit_title: <title or none>
- commit_sha: <short SHA or none>
- validation: <commands and results>
```

Do not claim that formatting, lint, or a comment-presence check proves comment quality. Review the changed comments
against the final implementation. A changed comment scope must not be reported as complete while its required commit
result is missing or unexplained.
