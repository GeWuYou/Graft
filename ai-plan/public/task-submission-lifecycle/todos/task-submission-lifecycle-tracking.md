# Task Submission Lifecycle Tracking

## Topic

Task Submission Lifecycle

## Scope

Replace Task-level Reservation activation with a first-class Submission lifecycle, a generic local materialization contract, owner advisory-lock coordination, legacy recovery, and `TaskStatusReady` cross-boundary projection.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/domains/task/任务提交生命周期设计.md`
- `ai-plan/design/decisions/ADR-022-task-submission-materialization.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Task Runtime owns Submission and Task state; Build owns its prerequisite writer; OpenAPI owns the shared Task status wire contract.
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/task-submission-lifecycle/README.md
    - ai-plan/public/task-submission-lifecycle/startup-prompt.md
    - ai-plan/public/task-submission-lifecycle/todos/task-submission-lifecycle-tracking.md
    - ai-plan/public/task-submission-lifecycle/traces/task-submission-lifecycle-trace.md
    - ai-plan/design/domains/task/任务提交生命周期设计.md
    - ai-plan/design/decisions/ADR-022-task-submission-materialization.md
    - ai-plan/roadmap/Task提交生命周期实施计划.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: cross-boundary-validation.
- Completed: Work Intake, design, ADR, roadmap and topic bootstrap.
- Completed: Submission persistence, Build materializer, legacy cutover migration, generated projections and validation.

## Task Checklist

- [x] Work Intake, topic, design, ADR and roadmap bootstrap.
- [x] Establish `TaskStatusReady` and Submission materialization contracts.
- [x] Implement Submission persistence, owner advisory-lock protocol and expiry recovery.
- [x] Implement Build writer and local atomic materialization.
- [x] Migrate legacy Reservation rows and update OpenAPI/Web projections.
- [x] Run cross-boundary validation and closeout.

## Acceptance Conditions

- No Submission remains permanently reserved.
- Worker only claims ready Tasks.
- Local Snapshot, Task, Submission and owner claim write atomically.
- Legacy activation flag cannot create a ready Task without durable Snapshot evidence.

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
