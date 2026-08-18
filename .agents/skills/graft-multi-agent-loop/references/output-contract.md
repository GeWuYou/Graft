# Multi-Agent Loop Output Contract

Read this reference when validating worker evidence, recording a controller transition, or emitting a machine-readable
outer-controller closeout.

## Worker round evidence

Worker closeout contains `round_status`, `implementation_result`, `changed_scope`, `commit`, `validation_evidence`,
`risks`, `blockers`, model-relation evidence, and `required_context` when retry or recovery depends on prior facts.
`suggested_follow_up` is optional advisory metadata. A worker never emits controller fields such as `continue`,
`pending_batches`, `next_batch`, `archive_ready`, `topic_complete`, `stop_loop`, `suspend_topic`, or `wait_for_user`.

## Controller decision

After `VERIFY`, only the outer controller records `closeout_status`, `current_batch`, `completed_batches`,
`pending_batches`, `next_batch`, budgets, recovery, and stop reason. `retry_exhausted` remains wave evidence until the
outer controller verifies and settles it. An unsettled failure remains a recovery handoff; a terminal `blocked`,
`cancelled`, or `unsafe` decision requires `failed_batch_settled: true`.

Required top-level closeout fields:

```json
{
  "closeout_status": "completed_no_handoff | committed_and_handed_off | handoff_only | recovery_handoff | blocked | cancelled | unsafe",
  "continue": true,
  "loop_mode": "topic-completion-loop | checkpoint-loop | null",
  "current_batch": "string or null",
  "completed_batches": ["string"],
  "pending_batches": ["string"],
  "next_batch": "string or null",
  "next_batch_prompt": "string or null",
  "next_prompt": "string or null",
  "stop_reason": "string or null",
  "validation": {"status": "passed | failed | not_run", "commands": ["string"], "note": "string or null"},
  "commit": {"created": true, "sha": "string or null", "title": "string or null"},
  "consumed_budget": {"rounds": 1, "files_changed": 0, "commits": 0, "runtime_minutes": 0},
  "remaining_budget": {"rounds": 0, "files_changed": 0, "commits": 0, "runtime_minutes": 0},
  "scope_expanded": false,
  "risk_level": "low | medium | high",
  "recovery": {
    "status": "none | required | context_restored | complete",
    "resume_target": "RESUME_CURRENT_BATCH | null",
    "current_batch_preserved": true,
    "pending_batches_preserved": true,
    "failed_batch_settled": false,
    "retry_exhausted": false,
    "repair_authority": "string or null",
    "repair_eligible": false,
    "required_context": {"failed_round": "object", "evidence": "object"}
  }
}
```

When recovery is required, preserve the failed round and evidence in `required_context`, keep current and pending
batches unchanged, and resume through `RECOVERY_COMPLETE -> RESUME_CURRENT_BATCH -> DISPATCH`.
