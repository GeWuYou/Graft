---
name: graft-commit
description: "Commit authorized Graft changes in validated logical slices while preserving exact ownership and staging boundaries."
---

# Graft Commit

Use [root governance](../../../AGENTS.md) and [delivery details](../../../ai-plan/design/governance/ai/仓库协作与交付细则.md) for authorization, ownership, staging, branch safety, and message rules. Reuse the current startup receipt and validation evidence.

## Scope and Repair

An explicit commit request or a validated closeout decision supplies the commit trigger. A bare `$graft-commit` captures the full initial tracked/untracked inventory as user-confirmed commit scope; a scoped request retains its named goal. Separate unrelated logical changes into independently understandable commits.

Diagnose validation or hook failures and continuously repair those necessary to the authorized goal, including newly discovered related files and regression tests with clear ownership. The initial inventory protects against unrelated later-arriving changes; it does not prohibit necessary task-owned repairs. Revalidate affected behavior. Ask only when the repair changes the goal, touches inseparable user/unknown changes, or needs a new external action. An extra repair commit is not itself a permission boundary.

## Workflow

1. Inspect `git status --short`, the actual diff, and the index. Left-column `M ` is staged; right-column ` M` is unstaged. IDE selections are not Git staging evidence.
2. Classify ownership and group the authorized inventory and necessary repairs into logical slices. Preserve user and concurrent worker changes; stage exact paths or separable hunks.
3. Confirm task-class completion validation. Reuse passing evidence for unchanged code; hooks and CI still run normally. If PR remediation is active, retain the exhaustive finding inventory including folded outside-diff and nitpick rows.
4. Inspect `git diff --cached` before committing. Use a Conventional Commit title `<type>(<scope>): <summary>` and real multiline body with meaningful `- ` bullets, following delivery details. No escaped newline literals, broad formatting, or title-only ordinary commits.
5. Commit each validated logical slice, inspect the remaining status, and continue through the authorized inventory. A bare invocation finishes clean unless unrelated later arrivals or a concrete ownership, safety, or validation blocker remain; identify those explicitly.
6. Report each commit SHA/title, scope, and validation result. For a real future-turn handoff, include the root startup context.

Do not commit ambiguous ownership, bypass a failing feasible validation, or imply PR-review closure while findings remain unclassified. A commit grants no push, merge, or other external-write authority.
