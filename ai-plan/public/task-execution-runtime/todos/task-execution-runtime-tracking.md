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
- The Task Runtime still has no worker, dispatcher, retry/cancel coordination, HTTP/realtime route, or consumer executor.
- Next: `task-runtime-worker-and-recovery`.

## Task Checklist

- [x] task-runtime-foundation-authority
- [x] task-module-persistence-state-machine
- [ ] task-runtime-worker-and-recovery
- [ ] task-api-realtime-and-project-adoption
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
    "task-module-persistence-state-machine"
  ],
  "pending_batches": [
    "task-runtime-worker-and-recovery",
    "task-api-realtime-and-project-adoption",
    "task-web-module-and-project-ui",
    "task-final-integration-archive-readiness"
  ],
  "current_batch": "task-module-persistence-state-machine",
  "next_batch": "task-runtime-worker-and-recovery",
  "closeout_status": "completed_no_handoff"
}
```
