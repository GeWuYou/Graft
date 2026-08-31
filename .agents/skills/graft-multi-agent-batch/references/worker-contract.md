# Batch Worker Contract

Read this reference when dispatching a worker, requesting a checkpoint, validating closeout, or preparing one retry.

## Dispatch package

Pass governance source, task class, recovery source, topology revision, node id, dependency list, objective, owned
scope, forbidden scope, authority owner, execution context, validation, acceptance gate, output format, commit
authority, and verified parent/worker model relation. Never dispatch a write-capable worker with only an objective or
file target. The worker receives an immutable topology slice and cannot change its dependencies or frontier.

## Final closeout

Require `round_status`, `implementation_result`, `changed_scope`, `commit`, `validation_evidence`, `risks`, `blockers`,
`topology_evidence`, model-relation evidence, and `required_context` when retry or recovery needs prior facts.
`topology_evidence` identifies the assigned `topology_revision` and `node_id`, records dependency and acceptance-gate
observations, and reports `completed`, `blocked`, or `retry-needed`; it is evidence only. `suggested_follow_up` is
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

## Topology Evidence

The worker may report facts about its assigned node, but it must not select or mutate the next frontier. A minimal
node-level evidence shape is:

```text
topology_evidence:
- topology_revision: <integer>
- node_id: <assigned-node>
- dependency_observed: passed | failed | not-applicable
- node_status: completed | blocked | retry-needed
- gate_evidence: <validation or acceptance evidence>
```

The batch coordinator may additionally report settled nodes, blocked descendants, ready-node candidates, and a bounded
replan delta to the outer controller. These remain advisory wave evidence; only the outer controller updates
`pending_batches`, `next_batch`, recovery transitions, or terminal state.
