# Machine-readable Closeout

Read only when the caller requests machine-readable closeout. Root governance and the loop controller remain authoritative.

### JSON Closeout Contract

When a caller such as `graft-multi-agent-loop` requests machine-readable closeout, use these rules:

- keep the human-readable closeout first
- if a future turn is required, include exactly one line beginning with `Next-session startup prompt:`
- append exactly one fenced ` ```json ` block as the last structured artifact
- for ordinary closeout outside `graft-multi-agent-loop`, `continue=true` requires a non-empty `next_prompt`
- for `graft-multi-agent-loop` in the default `topic-completion-loop` mode:
  - `continue=true` requires `loop_mode=topic-completion-loop`
  - `continue=true` requires updated `completed_batches` and `pending_batches`
  - `continue=true` requires `next_batch` and `next_batch_prompt` when batches remain
  - `pending_batches=[]` requires a final archive-readiness check before the loop may stop
  - `continue=true` requires `next_prompt=null`
  - a terminal handoff is limited to `blocked`, `cancelled`, `unsafe`, `archive-ready`, or explicit stop; it must
    include a terminal reason resolved by the outer controller
  - a `recovery_handoff` is non-terminal and may emit `Next-session startup prompt:` only when `recovery.status` is
    `required`, `current_batch` and `pending_batches` are preserved, `recovery.resume_target` is
    `RESUME_CURRENT_BATCH`, and the failed batch is unsettled
  - after startup preflight restores the receipt and recovery becomes eligible, the outer controller must transition
    `RECOVERY_COMPLETE -> RESUME_CURRENT_BATCH -> DISPATCH`; it must not wait for a user re-dispatch
- `continue=false` requires `next_prompt=null` unless the closeout is a terminal or recoverable handoff to a future
  turn
- `validation.status` should be one of `passed`, `failed`, or `not_run`
- `risk_level` should be one of `low`, `medium`, or `high`
- `consumed_budget` must describe only the current slice or delegated round
- `remaining_budget` must reflect the budget left after the current slice or delegated round

Recommended JSON shape:

```json
{
  "closeout_status": "completed_no_handoff | committed_and_handed_off | handoff_only | recovery_handoff | blocked | cancelled | unsafe",
  "continue": true,
  "loop_mode": "topic-completion-loop | checkpoint-loop | null",
  "current_batch": "string or null",
  "completed_batches": ["..."],
  "pending_batches": ["..."],
  "next_batch": "string or null",
  "next_batch_prompt": "string or null",
  "next_prompt": "string or null",
  "stop_reason": "string or null",
  "validation": {
    "status": "passed | failed | not_run",
    "commands": ["..."],
    "note": "string or null"
  },
  "verification": {
    "classification": "agent-only | human-acceptance-required | browser-automation-required | mixed",
    "browser_status": "not-needed | authorized-local | ci-contract",
    "human_acceptance": "not-required | required | awaiting_human_acceptance"
  },
  "commit": {
    "created": true,
    "sha": "string or null",
    "title": "string or null"
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 3,
    "commits": 0,
    "runtime_minutes": 10
  },
  "remaining_budget": {
    "rounds": 4,
    "files_changed": 27,
    "commits": 2,
    "runtime_minutes": 80
  },
  "scope_expanded": false,
  "risk_level": "low",
  "recovery": {
    "status": "none | required | context_restored | complete",
    "resume_target": "RESUME_CURRENT_BATCH | null",
    "current_batch_preserved": true,
    "pending_batches_preserved": true,
    "failed_batch_settled": false,
    "retry_exhausted": false,
    "repair_authority": "string or null",
    "repair_eligible": false,
    "required_context": {
      "failed_round": "object",
      "evidence": "object"
    }
  }
}
```

When this closeout is consumed by `graft-multi-agent-loop`, the `recovery` object is evidence input and the outer
controller owns the canonical decision record. Its controller record must preserve `current_batch` and
`pending_batches` during recovery and include `controller_state`, `terminal_reason`, and the complete recovery fields
shown in the loop skill. `retry_exhausted=true` records wave evidence only; it does not make `closeout_status`
terminal and does not authorize a worker or closeout helper to choose `BLOCKED`, `ARCHIVE_READY`, or a recovery
transition.

For `graft-multi-agent-loop` in the default `topic-completion-loop` mode, prefer this interpretation:

- `continue=true` with `next_prompt=null` means same-session continuation
- `next_batch` and `next_batch_prompt` drive the next delegated worker round
- `Next-session startup prompt:` is emitted for terminal handoff or a structured `recovery_handoff`; recovery is not a
  terminal state and must resume the preserved current batch after its eligibility conditions are met
