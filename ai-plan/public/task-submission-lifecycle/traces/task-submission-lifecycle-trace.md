# Task Submission Lifecycle Trace

## 2026-08-05 work-intake-bootstrap

- Classified the work as a long-running cross-boundary refactor with no existing active topic.
- Created design authority, ADR-022, roadmap and active-topic recovery materials.
- Locked independent Submission/Task state machines, local atomic materialization, Task ready worker visibility and claim-as-constraint semantics.
- Rejected a persistent owner-slot projection after confirming PostgreSQL constraints are relation-local; use factual-table local indexes plus a documented PostgreSQL transaction advisory-lock protocol.

## Locked Decisions

- `activated` is a terminal Submission state.
- `submission_version` fences every Submission mutation.
- Legacy `activation_required=true` without Snapshot evidence never becomes a ready Task.

## 2026-08-06 implementation-closeout

- Added `task_submissions` with lease/deadline/version fencing, idempotent terminal transitions and periodic expiry recovery.
- Added generic transaction-scoped prerequisite writing; Build Snapshot, Task/Stage/Event creation and Submission activation commit atomically.
- Added scheduled-to-ready promotion so Worker claim SQL never treats unactivated or scheduled Tasks as worker-entry Tasks.
- Added evidence-gated legacy migration: Build Snapshot rows become activated/ready; rows without Snapshot evidence become expired/cancelled.
- Regenerated OpenAPI, module migration registry and Web types; backend lint, full Go tests, Web checks and migration gates passed.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-bootstrap", "authority-and-contract", "persistence-and-owner-claim", "build-materializer", "legacy-migration-and-projection", "cross-boundary-validation"],
  "pending_batches": [],
  "current_batch": "cross-boundary-validation",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
