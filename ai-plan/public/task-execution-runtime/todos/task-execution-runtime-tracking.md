# Task Execution Runtime Tracking

## Topic

Task Execution Runtime

## Scope

Provide a platform-level state-machine Task Runtime that consumers use for multi-stage operations while preserving module boundaries and PostgreSQL authority.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/任务执行运行时设计.md`
- `ai-plan/design/decisions/ADR-004-task-runtime-state-machine.md`
- `openapi/**`
- `server/internal/moduleapi/task.go`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Task Runtime design, Task state-machine ADR, canonical OpenAPI, and moduleapi form the platform contract; project is the first consumer.
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - topic
    - design
    - roadmap
    - adr
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Persistence and state-machine foundation completed: `task` is registered as a compile-time module with module-owned
  `tasks`、`task_stages`、`task_events` 和 `task_logs` migrations, SQL persistence and tested state transitions.
- The Task Runtime has a PostgreSQL-backed serial dispatcher, fixed in-process workers, executor registry, Task submission,
  cancellation, retry coordination, and conservative crash recovery. It still has no HTTP/realtime route or Project consumer.
- Task API, realtime and the first Project adoption are complete in the current batch. Next: `task-web-module-and-project-ui`.

## Task Checklist

- [x] task-runtime-foundation-authority
- [x] task-module-persistence-state-machine
- [x] task-runtime-worker-and-recovery
- [x] task-api-realtime-and-project-adoption
- [ ] task-web-module-and-project-ui
- [ ] task-final-integration-archive-readiness

## Acceptance Conditions

- Task Runtime remains module-owned and consumer-agnostic.
- PostgreSQL persists task, stage, log and non-derivable event facts.
- A non-resumable crashed Stage becomes `unknown`; its Task becomes `needs_attention`.
- Generic Task API uses owner authorization and explicit capabilities.
- Project can become the first consumer without Task importing Project.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "task-runtime-foundation-authority",
    "task-module-persistence-state-machine",
    "task-runtime-worker-and-recovery",
    "task-api-realtime-and-project-adoption"
  ],
  "pending_batches": [
    "task-web-module-and-project-ui",
    "task-final-integration-archive-readiness"
  ],
  "current_batch": "task-api-realtime-and-project-adoption",
  "next_batch": "task-web-module-and-project-ui",
  "closeout_status": "completed_no_handoff"
}
```
