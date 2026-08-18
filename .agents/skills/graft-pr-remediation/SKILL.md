---
name: graft-pr-remediation
description: Repository-specific closed-loop workflow for Graft PR remediation. Use only when the user explicitly invokes `$graft-pr-remediation` or explicitly asks to fix verified PR findings, validate and publish the repairs, reply to eligible still-open AI review threads, resolve those threads, and append the verified run to the managed review ledger.
---

# Graft PR Remediation

Use this skill for one explicitly authorized end-to-end PR repair run. Treat root `AGENTS.md` as the startup,
validation, commit, push, and closeout authority. Reuse `graft-pr-review` for inventory and dispositions;
`graft-validation-runner`, `graft-commit`, and `graft-push` remain the owners of their respective stages.

## Authorization Contract

An explicit `$graft-pr-remediation` invocation authorizes this bounded workflow for the current branch PR:

- repair every verified `actionable-local` finding in the owned scope
- run the required validation
- create scoped commits and push the current task branch through `graft-commit` and `graft-push`
- reply to and resolve eligible still-open threads created by supported AI reviewers
- append and verify one final run in the `graft-pr-review` managed ledger

It does not authorize force-push, merge, PR title/body changes, arbitrary issue comments, human-authored thread
resolution, or unbounded `actionable-large` work. Keep those operations behind their existing explicit authorization
gates. Managed-ledger authority and payload validation remain owned by `graft-pr-review`.

## Closed-Loop Workflow

1. Complete the root startup preflight and inspect `git status --short` before any write.
2. Run the complete `graft-pr-review` inventory. Reconcile every open thread, folded group, pre-merge warning or
   inconclusive check, failed workflow, security signal, and test report before editing.
3. Assign every finding exactly one disposition: `fixed`, `delegated`, `blocked`, `stale`, or `noise`.
4. Repair every `actionable-local` finding in the authorized scope. Route `actionable-large` work through the
   repository's governed delegation path or leave it `blocked`; do not disguise it as `stale` or `noise`.
5. Use `graft-validation-runner` for the smallest correct completion entrypoint. Preserve the exhaustive finding
   inventory through validation, commit, and push.
6. Use `graft-commit` to commit every validated owned repair and `graft-push` to publish it. An explicit invocation of
   this skill is the explicit commit-and-push request for this bounded workflow.
7. Before any PR reply or resolution, run
   `git ls-remote --exit-code origin refs/heads/<current-branch>` and require its SHA to equal `git rev-parse HEAD`.
   Also require the PR head SHA to equal that same value and the fresh inventory's `thread_resolution_source` to be
   `github-graphql`; fail closed if authoritative thread state is unavailable.
8. Rebuild the full PR inventory after publication. A thread is eligible for reply and resolution only when all of
   these are true:
   - it is still open and was created by a supported AI reviewer
   - its finding is `fixed`, `stale`, or `noise` with current-HEAD evidence
   - it is not `contested`, `delegated`, or `blocked`
   - no human-authored review judgment remains unresolved in the thread
9. For each eligible thread:
   - write a concise reply that names the disposition and cites the fixing commit, current path/test evidence, or
     canonical authority that proves `noise`
   - preview the reply with the `graft-pr-review` helper's `--reply-dry-run`, then send the exact reviewed payload
   - rebuild the inventory; if the reviewer has contested the reply, stop and leave the thread open for human review
   - preview resolution with
     `--resolve-comment-id <root-comment-id> --resolve-expected-head <full-sha> --resolve-dry-run`, then rerun without
     `--resolve-dry-run`
10. Rebuild the complete inventory once more. Confirm every eligible thread is no longer open and every remaining open
    finding is explicitly `delegated`, `blocked`, or awaiting human judgment.
11. Build one final managed-ledger entry from that inventory. Include all required `graft-pr-review` fields:
    `coderabbit_handled`, `coderabbit_outside_diff_range`, `coderabbit_nitpick`,
    `coderabbit_pre_merge_checks`, `open_suggestions`, and `greptile_suggestions`. Also include disposition counts,
    validation, publication proof, and AI-thread replied/resolved/remaining counts.
12. Validate the entry offline with `--ledger-validate-body-file`. Reconfirm local, remote branch, and PR head SHAs;
    preview the exact append with `--ledger-body-file ... --ledger-expected-head <full-sha> --ledger-dry-run`, then
    rerun without `--ledger-dry-run`. Rebuild the complete inventory and require the managed ledger's latest run to
    record the current PR head and the exact validated entry.
13. Finish only after the eligible-thread count is zero and the final managed-ledger append is visible in the refreshed
    inventory. A failed or unverifiable ledger append leaves the remediation run `blocked`, not complete.

Read `graft-pr-review`'s [finding lifecycle](../graft-pr-review/references/finding-lifecycle.md) before drafting replies
and its [helper commands](../graft-pr-review/references/commands.md) before performing remote writes.

## Reply Rules

- For `fixed`, name the published commit and the validated repair location.
- For `stale`, state the current-HEAD evidence that makes the old finding no longer applicable.
- For `noise`, state the canonical authority and why the reviewer interpretation is false.
- Never resolve a thread merely because its diff line is outdated or a bot printed an “addressed” marker.
- Never reply to or resolve human-authored threads through this skill.
- Do not wait synchronously for an AI reviewer. A later response is a new inventory event and may become `contested`.

## Closeout

```text
Graft PR remediation:
- inventory_complete: yes | no with blocker
- findings: fixed / delegated / blocked / stale / noise counts
- validation: commands and results
- commits: SHAs and titles
- publication_proof: local / remote branch / PR head SHAs
- ai_threads: eligible / replied / resolved / remaining counts
- managed_ledger: validated / dry-run / appended / verified
- remaining_open: explicit delegated, blocked, or human-review rows
- writes: files, commits, push, replies, resolutions, managed-ledger append
```

Do not claim closed-loop completion while an eligible verified AI thread remains open or the final managed-ledger run
has not been verified.
