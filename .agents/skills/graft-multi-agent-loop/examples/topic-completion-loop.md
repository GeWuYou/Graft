# Topic Completion Loop Example

This example shows the default `topic-completion-loop` lifecycle.

## Initial State

```yaml
loop_mode: topic-completion-loop
completed_batches: []
pending_batches:
  - batch1-baseline-audit
  - batch2-route-alignment
  - batch3-closeout-hardening
current_batch: batch1-baseline-audit
next_batch: null
```

## Batch 1 Closeout

Human-readable closeout:

```text
Batch 1 baseline audit completed within owned scope. Validation passed. No blockers require user intervention.
```

Machine-readable closeout:

```json
{
  "closeout_status": "completed_no_handoff",
  "continue": true,
  "loop_mode": "topic-completion-loop",
  "current_batch": "batch1-baseline-audit",
  "completed_batches": ["batch1-baseline-audit"],
  "pending_batches": ["batch2-route-alignment", "batch3-closeout-hardening"],
  "next_batch": "batch2-route-alignment",
  "next_batch_prompt": "Execute Batch 2 within owned scope. Align canonical routes and keep validation focused on the changed batch.",
  "next_prompt": null,
  "stop_reason": null,
  "validation": {
    "status": "passed",
    "commands": ["git diff --check"],
    "note": null
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
    "runtime_minutes": 8
  },
  "remaining_budget": {
    "rounds": 4,
    "files_changed": 20,
    "commits": 2,
    "runtime_minutes": 72
  },
  "scope_expanded": false,
  "risk_level": "low"
}
```

Outer main agent action:

1. Accept Batch 1 closeout.
2. Update `completed_batches`.
3. Update `pending_batches`.
4. Choose `batch2-route-alignment`.
5. Dispatch the next worker.

## Batch 2 Closeout

```json
{
  "closeout_status": "completed_no_handoff",
  "continue": true,
  "loop_mode": "topic-completion-loop",
  "current_batch": "batch2-route-alignment",
  "completed_batches": ["batch1-baseline-audit", "batch2-route-alignment"],
  "pending_batches": ["batch3-closeout-hardening"],
  "next_batch": "batch3-closeout-hardening",
  "next_batch_prompt": "Execute Batch 3 within owned scope. Finish closeout hardening and final validation for the topic.",
  "next_prompt": null,
  "stop_reason": null,
  "validation": {
    "status": "passed",
    "commands": ["git diff --check"],
    "note": null
  },
  "commit": {
    "created": true,
    "sha": "abc1234",
    "title": "docs(graft-loop): define topic completion lifecycle"
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 3,
    "commits": 1,
    "runtime_minutes": 14
  },
  "remaining_budget": {
    "rounds": 3,
    "files_changed": 17,
    "commits": 1,
    "runtime_minutes": 58
  },
  "scope_expanded": false,
  "risk_level": "low"
}
```

Outer main agent action:

1. Accept Batch 2 closeout.
2. Update trace and todo.
3. Update `completed_batches`.
4. Update `pending_batches`.
5. Dispatch Batch 3.

## Batch 3 Closeout

Human-readable closeout:

```text
Batch 3 completed within owned scope. `pending_batches` is now empty, so the outer main agent must run the final archive-readiness check before stopping.
```

Machine-readable closeout:

```json
{
  "closeout_status": "completed_no_handoff",
  "continue": true,
  "loop_mode": "topic-completion-loop",
  "current_batch": "batch3-closeout-hardening",
  "completed_batches": [
    "batch1-baseline-audit",
    "batch2-route-alignment",
    "batch3-closeout-hardening"
  ],
  "pending_batches": [],
  "next_batch": null,
  "next_batch_prompt": null,
  "next_prompt": null,
  "stop_reason": null,
  "validation": {
    "status": "passed",
    "commands": ["git diff --check"],
    "note": "Final archive-readiness check is still required because pending_batches is now empty."
  },
  "commit": {
    "created": false,
    "sha": null,
    "title": null
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 4,
    "commits": 0,
    "runtime_minutes": 16
  },
  "remaining_budget": {
    "rounds": 2,
    "files_changed": 13,
    "commits": 1,
    "runtime_minutes": 42
  },
  "scope_expanded": false,
  "risk_level": "low"
}
```

Outer main agent action:

1. Accept Batch 3 closeout.
2. Observe that `pending_batches=[]`.
3. Run the final archive-readiness check instead of stopping immediately.

## Final Archive-Readiness Check

```text
Final archive-readiness check passed. Topic reached archive-ready state and the owned closeout docs were committed.
Next-session startup prompt: Governance source: root AGENTS.md. Task class: docs/automation. Recovery source: none. Owned scope: .agents/skills/graft-multi-agent-loop/** plus bounded shared governance docs. First rerun startup preflight from AGENTS.md 4.1 only if follow-up governance work is later reopened.
```

```json
{
  "closeout_status": "completed_no_handoff",
  "continue": false,
  "loop_mode": "topic-completion-loop",
  "current_batch": "final-archive-readiness-check",
  "completed_batches": [
    "batch1-baseline-audit",
    "batch2-route-alignment",
    "batch3-closeout-hardening",
    "final-archive-readiness-check"
  ],
  "pending_batches": [],
  "next_batch": null,
  "next_batch_prompt": null,
  "next_prompt": "Governance source: root AGENTS.md. Task class: docs/automation. Recovery source: none. Owned scope: .agents/skills/graft-multi-agent-loop/** plus bounded shared governance docs. First rerun startup preflight from AGENTS.md 4.1 only if follow-up governance work is later reopened.",
  "stop_reason": "archive-ready",
  "validation": {
    "status": "passed",
    "commands": ["git diff --check", "rg -n \"topic-completion-loop|completed_batches|pending_batches\" .agents/skills/graft-multi-agent-loop"],
    "note": null
  },
  "commit": {
    "created": true,
    "sha": "def5678",
    "title": "docs(graft-loop): clarify checkpoint semantics"
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 4,
    "commits": 1,
    "runtime_minutes": 16
  },
  "remaining_budget": {
    "rounds": 2,
    "files_changed": 13,
    "commits": 0,
    "runtime_minutes": 42
  },
  "scope_expanded": false,
  "risk_level": "low"
}
```

Terminal result:

- success -> continue after Batch 1
- success -> continue after Batch 2
- success -> final archive-readiness check after Batch 3
- archive-ready only after the final archive-readiness check passes
