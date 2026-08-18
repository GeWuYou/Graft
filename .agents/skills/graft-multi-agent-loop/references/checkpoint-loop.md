# Checkpoint Loop Example

This example shows the explicit non-default `checkpoint-loop` mode.

Use this mode only when the caller explicitly sets `loop_mode=checkpoint-loop`.

## Initial State

```yaml
loop_mode: checkpoint-loop
completed_batches:
  - batch1-audit
pending_batches:
  - batch2-risk-evaluation
current_batch: batch2-risk-evaluation
```

## Checkpoint Response

```text
Checkpoint status:
- current_phase: waiting for architecture decision
- changed_files: []
- last_validation: not_run
- next_action: pause until menu ownership decision is made
- can_continue: false
- estimated_remaining_minutes: 0
- eta_confidence: high
- risks_or_blockers: user architecture decision required
```

Rules illustrated here:

- checkpoint is a health report, not a final closeout
- `checkpoint-loop` is explicit, not implicit
- the outer main agent may stop and hand off only because a terminal blocker exists

## Terminal Closeout

```text
Architecture decision is required before Batch 2 can continue safely.
Next-session startup prompt: Governance source: root AGENTS.md. Task class: docs/automation. Recovery source: parent topic. Owned scope: .agents/skills/graft-multi-agent-loop/** plus bounded shared governance docs. First rerun startup preflight from AGENTS.md 4.1, then resume with the architecture decision for Batch 2.
```

```json
{
  "closeout_status": "blocked",
  "continue": false,
  "loop_mode": "checkpoint-loop",
  "current_batch": "batch2-risk-evaluation",
  "completed_batches": ["batch1-audit"],
  "pending_batches": ["batch2-risk-evaluation"],
  "next_batch": null,
  "next_batch_prompt": null,
  "next_prompt": "Governance source: root AGENTS.md. Task class: docs/automation. Recovery source: parent topic. Owned scope: .agents/skills/graft-multi-agent-loop/** plus bounded shared governance docs. First rerun startup preflight from AGENTS.md 4.1, then resume with the architecture decision for Batch 2.",
  "stop_reason": "architecture decision required",
  "validation": {
    "status": "not_run",
    "commands": [],
    "note": "Loop blocked before validation because the architecture decision is still unresolved."
  },
  "commit": {
    "created": false,
    "sha": null,
    "title": null
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 0,
    "commits": 0,
    "runtime_minutes": 6
  },
  "remaining_budget": {
    "rounds": 2,
    "files_changed": 12,
    "commits": 1,
    "runtime_minutes": 34
  },
  "scope_expanded": false,
  "risk_level": "medium"
}
```
