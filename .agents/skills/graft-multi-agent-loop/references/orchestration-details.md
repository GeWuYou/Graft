# Loop Orchestration Details

Read this reference when preparing budgets, wait/checkpoint handling, repair recovery, retry, or a handoff.

## Budget

Record maximum rounds, files changed, commits, runtime minutes, risk ceiling, allowed scope, and forbidden scope before
dispatch. Stop dispatching new work when a limit is reached; preserve controller state and emit a governed handoff.
Do not reinterpret a budget stop as topic completion.

## Wait and checkpoint

Use one active worker per serial round by default. A quiet wait, no visible diff, or no final closeout is observation,
not proof of stall. Use the default 15-minute initial wait and one bounded checkpoint only when no final closeout and
no new evidence are visible. After `can_continue=true`, resume the same worker and wait at least 30 minutes; continue
with credible ETA or doubled windows. A checkpoint is a health report, not closeout or takeover permission.

Stop or retry only on final closeout, explicit blocked state, `can_continue=false`, owned-scope conflict, unsafe
worktree, or actual retry-policy activation. Do not interrupt merely because a worker is slow while evidence grows.

## Verification and repair

Validate worker scope, behavior, comments, required repository entrypoints, and commit evidence. Do not accept a
successful worker result with uncommitted owned changes when the round contract requires commit. If validation or
integration reveals a bounded worker-owned repair, continue the same worker when safe; otherwise enter recovery.

For a failed or malformed closeout, request one bounded recovery attempt, then retry the same batch once with a fresh
worker carrying prior failure, partial diff, validation evidence, and `required_context`. Retry exhaustion is wave
evidence only. Preserve current and pending batches until the outer controller settles the failure.

## Recovery and handoff

`RECOVERY_REQUIRED` records the failed round, evidence, repair authority, repair eligibility, and
`RESUME_CURRENT_BATCH` target. A resumed session reruns startup preflight, restores the receipt as
`RECOVERY_CONTEXT_RESTORED`, confirms repair authority, then transitions through
`RECOVERY_COMPLETE -> RESUME_CURRENT_BATCH -> DISPATCH`.

Use `Next-session startup prompt:` only for terminal handoff or recoverable recovery handoff. Ordinary successful
topic-completion rounds advance automatically without a next-session prompt.
