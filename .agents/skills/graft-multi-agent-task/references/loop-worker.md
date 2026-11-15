# Loop Worker Evidence

Read for loop-orchestrated output and checkpoint handling. The [batch worker contract](../../graft-multi-agent-batch/references/worker-contract.md) defines the immutable assigned topology and node-level evidence. A worker does not resume the current batch, dispatch a repaired worker, or choose controller recovery transitions; it returns repair eligibility and required context to its coordinator.

## Output Expectations

When reporting progress or closeout from this wrapper, keep the result brief and include:

1. whether `graft-multi-agent-batch` was used for execution
2. whether `graft-task-closeout` was used for handoff evaluation
3. whether `graft-commit` created a scoped commit for the validated owned scope
4. whether `graft-lessons-learned` was reached through `graft-task-closeout` or explicit lesson delegation
5. the next-session startup prompt, if a handoff is required
6. when the task is loop-orchestrated, a trailing JSON closeout object for the current delegated round with:
   - `round_status`
   - `implementation_result`
   - `changed_scope`
   - `topology_evidence`
   - `commit`
   - `validation_evidence`
   - `risks`
   - `blockers`
   - `required_context` when recovery or retry needs prior evidence
   - optional `recovery_requirements`, containing cause, repair proposal reference, authority evidence, and safe retry
     inputs without directing controller state
   - `model_delegation_evidence`
   - optional `suggested_follow_up`
  - do not include `continue`, `pending_batches`, `next_batch`, `archive_ready`, `topic_complete`, `stop_loop`,
    `suspend_topic`, or `wait_for_user`

The worker JSON above is evidence only. The outer controller must separately emit the canonical controller closeout
record defined by `graft-multi-agent-loop`; a worker must never substitute that record with its own `pending_batches`,
`next_batch`, terminal state, or recovery transition.

When a loop-orchestrated worker answers a checkpoint request instead of a final closeout, keep the response short and
structured. It must include:

1. `current_phase`
2. `changed_files`
3. `last_validation`
4. `next_action`
5. `can_continue`
6. `estimated_remaining_minutes`
7. `eta_confidence`
8. `risks_or_blockers`

Checkpoint responses should also follow these formatting rules:

- begin with `Checkpoint status:`
- do not include `Next-session startup prompt:`
- do not append the final closeout JSON block
- if the worker can continue, leave final completion, validation, and closeout to the same worker round

When this wrapper runs as a `graft-multi-agent-loop` worker in the default `topic-completion-loop` mode:

- worker completion returns round evidence to the outer main agent; it never completes the topic loop
- `suggested_follow_up` may preserve recovery context or describe candidate next work, but it is non-authoritative
- only the outer controller maintains `completed_batches`, `pending_batches`, `current_batch`, and `next_batch`, and
  only it can select recovery transitions, `archive-ready`, or `blocked`
