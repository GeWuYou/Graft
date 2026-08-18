# Batch Worker Contract

Read this reference when dispatching a worker, requesting a checkpoint, validating closeout, or preparing one retry.

## Dispatch package

Pass governance source, task class, recovery source, objective, owned scope, forbidden scope, validation, output
format, commit authority, and verified parent/worker model relation. Never dispatch a write-capable worker with only an
objective or file target.

## Final closeout

Require `round_status`, `implementation_result`, `changed_scope`, `commit`, `validation_evidence`, `risks`, `blockers`,
model-relation evidence, and `required_context` when retry or recovery needs prior facts. `suggested_follow_up` is
optional advisory metadata. A worker never emits controller state such as `continue`, `pending_batches`, `next_batch`,
`archive_ready`, `topic_complete`, `stop_loop`, `suspend_topic`, or `wait_for_user`.

## Checkpoint

A checkpoint begins with `Checkpoint status:` and contains `current_phase`, `changed_files`, `last_validation`,
`next_action`, `can_continue`, `estimated_remaining_minutes`, `eta_confidence`, and `risks_or_blockers`. It is a health
report, not closeout or permission for local takeover.

Use the default wait ladder only when the caller has not set a stronger bound: wait 15 minutes, send one bounded health
checkpoint only when no final closeout and no new evidence exist, then wait at least 30 minutes after
`can_continue=true`. Continue with credible ETA or doubled windows. End only on final closeout, explicit blocked state,
owned-scope conflict, unsafe worktree, or actual retry-policy activation.

Retry the same bounded slice once with a fresh worker when no usable closeout can be obtained. Pass prior failure,
partial owned-scope diff, and validation evidence. A second failure returns retry-exhaustion wave evidence and
`required_context` to the outer controller; it never terminates the topic loop.
