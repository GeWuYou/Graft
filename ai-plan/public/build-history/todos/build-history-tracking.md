# Build History Tracking

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Build owns history projections and artifact evidence; Task, Container, and Project retain their established authority boundaries.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/public/build-history
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- `phase-2-history-discovery` selected Build-owned filtering and pagination as the smallest justified implementation batch.

## Task Checklist

- [x] Establish the first justified Build history batch.
- [ ] Add Build-owned history filtering and pagination.

## Acceptance Conditions

- Build remains the authority for job and artifact history.
- No duplicate Task execution/log/realtime authority is introduced.
- Any persistence, OpenAPI, and web changes pass the applicable completion validation.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-2-history-discovery"],
  "pending_batches": ["phase-2-history-filtering-and-pagination"],
  "current_batch": "phase-2-history-filtering-and-pagination",
  "next_batch": null,
  "closeout_status": "active",
  "stop_reason": null,
  "recovery": {"status": "none", "resume_target": null, "repair_authority": null, "repair_eligible": false}
}
```
