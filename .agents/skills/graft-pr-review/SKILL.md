---
name: graft-pr-review
description: Repository-specific GitHub PR review workflow for Graft. Use to inventory and verify current-branch PR findings from CodeRabbit, Greptile, Gemini Code Assist, GitHub Advanced Security, failed checks, MegaLinter, or failed tests; default to read-only review and require explicit authorization before local remediation or GitHub writes.
---

# Graft PR Review

Use this skill when the task depends on live GitHub PR state rather than local files alone. Treat root `AGENTS.md` as
the authority for startup, validation, commit, push, and closeout.

## Permission Stages

Use two fail-closed stages:

1. `inventory/review` is the default. Read GitHub and local code, build an exhaustive inventory, verify findings, and
   report dispositions. Do not edit files, commit, push, change PR metadata, reply to threads, or update the ledger.
2. `remediation/write` starts only when the user explicitly requests local fixes or a specific remote write. Permission
   to fix locally does not authorize commit, push, PR metadata changes, replies, ledger writes, or other GitHub writes;
   satisfy each owning skill's authorization gate independently.

## Inventory Workflow

1. Complete startup preflight and resolve the current branch and its PR.
2. Prefer GitHub MCP for focused live discovery. Use `scripts/fetch_current_pr_review.py` as the deterministic fallback
   and JSON normalizer whenever MCP is unavailable or a complete machine-readable inventory is needed.
3. Build one exhaustive inventory before any remediation:
   - unresolved latest-head threads from supported AI reviewers and GitHub Advanced Security
   - failed checks, failed jobs/steps, annotations, MegaLinter, and failed-test signals
   - CodeRabbit pre-merge checks, including every `Warning` and `Inconclusive`
   - all folded latest-review groups, including `Duplicate comments`, `Major comments`, `Minor comments`,
     `Outside diff range comments`, and `Nitpick comments`
4. Reconcile every declared folded-section count with parsed rows. An under-parsed group makes the inventory incomplete;
   inspect the raw review body or a focused helper section until all items are enumerated or an exact blocker is known.
5. Verify every row against the checked-out HEAD and classify it as `actionable-local`, `actionable-large`, `stale`, or
   `noise`. For CI, reproduce the smallest matching local command before diagnosing the failure. `Inconclusive` must be
   resolved; every `Warning` needs an explicit remediate or accept-with-reason decision.
6. In the default read-only stage, stop after reporting the full inventory, evidence, proposed repairs, and the exact
   authorization needed for any next write.

Read [references/finding-lifecycle.md](references/finding-lifecycle.md) when building dispositions, handling folded
sections, or deciding a remote response. Read [references/commands.md](references/commands.md) only when selecting
helper flags, JSON sections, dry runs, replies, or ledger validation.

## Authorized Remediation

When explicit local-remediation authority exists:

1. Fix every verified `actionable-local` finding in the authorized scope. Route an `actionable-large` finding through
   a justified `graft-multi-agent-batch`, `graft-multi-agent-loop`, or report it as `blocked`; never relabel size as
   `stale` or `noise`.
2. Use `graft-validation-runner` for any changed code. Focused checks are supplemental; completion still requires the
   full backend/web entrypoint selected by repository governance.
3. Keep commit and push behind `graft-commit` and `graft-push` authorization. A remediation request alone does not
   grant either operation.
4. Before any authorized PR-thread reply or managed ledger write, run
   `git ls-remote --exit-code origin refs/heads/<current-branch>` and require its SHA to equal `git rev-parse HEAD`.
   If publication is absent or mismatched, leave PR threads and the ledger untouched.
5. Preserve all non-owned PR description content. Only an explicitly authorized metadata update may write the managed
   description region documented in the finding lifecycle reference.

## Completion Contract

Every inventoried finding must end in exactly one disposition: `fixed`, `delegated`, `blocked`, `stale`, or `noise`.
Partial fixes, handled open threads, or a successful push do not close unreconciled folded findings.

Report:

```text
Graft PR review:
- stage: inventory/review | remediation/write
- inventory_source: GitHub MCP | deterministic fallback | mixed
- inventory_complete: yes | no with exact blocker
- findings: <each finding, evidence, classification, disposition>
- coderabbit_handled: <count>
- coderabbit_outside_diff_range: <declared/handled>
- coderabbit_nitpick: <declared/handled>
- coderabbit_pre_merge_checks: <status counts and decisions>
- open_suggestions: <before/after>
- greptile_suggestions: <count and dispositions>
- validation: <commands/results or not-applicable>
- writes: <none or explicitly authorized operations/results>
```

## Boundaries

- Do not treat a narrow `--section` query as a complete review.
- Do not start edits before exhaustive inventory or without explicit local-remediation authority.
- Do not infer GitHub write authority from read access, a token, MCP availability, or local remediation authority.
- Do not wait in the same run for an AI reviewer to answer a reply; classify follow-up in a later inventory.
- Do not create a second validation, commit, push, PR creation, or recovery workflow.
