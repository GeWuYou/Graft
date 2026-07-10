# ADR-004: Task Runtime State Machine

- Status: accepted
- Date: 2026-07-10
- Scope: `server/modules/task`, `server/internal/moduleapi/task.go`, `openapi/**`, Task consumers

## Context

Project lifecycle actions run external Docker Compose commands for a potentially long time. A one-function job queue cannot preserve a useful stage timeline, logs, cancellation semantics, retry policy or crash ambiguity. Temporal supplies useful persistence and history ideas, but its workflow/activity terminology and distributed runtime are not appropriate for Graft's current modular monolith.

## Decision

1. Add `server/modules/task` as a platform module-owned Task Runtime using `Task`, `TaskPlan`, `Stage`, and `StageExecutor` names.
2. Task Runtime owns state machines, persistence, workers, logs, limited event history, realtime, generic query APIs and history. Consumer modules own plans, executors and business authorization semantics.
3. PostgreSQL is the sole source of truth. Redis is optional auxiliary infrastructure only; neither a queue nor state is stored only in Redis.
4. Stage lifecycle is authoritative in `task_stages`; `task_events` only records non-derivable task lifecycle, retry, cancellation and recovery facts.
5. A process crash turns a running non-resumable Stage into `unknown` and the parent Task into `needs_attention`; Docker shell actions default to manual reconciliation, never automatic replay.
6. `scheduler` may create Tasks on schedule but does not run Stage executors.

## Consequences

- Project will migrate lifecycle actions from synchronous HTTP completion to `202 Accepted` Task receipts.
- Consumers cannot create task tables, runners, log flows or WebSocket topics.
- MVP remains serial and in-process; DAG and distributed execution need a future ADR after evidence of need.
