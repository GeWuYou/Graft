---
name: graft-event-contract-review
description: Default semantic review of Graft fact-event contracts, delivery guarantees, transaction boundaries, idempotent consumers, retries, and operational observability. Use when adding or changing server/internal/event, eventbus, event handlers, durable delivery, or event-driven module behavior.
---

# Graft Event Contract Review

Use this skill as Graft's default semantic review layer for event-driven behavior. It complements the owning module,
`server/internal/event`, the Task Runtime, contract governance, and existing validation entrypoints; it does not create
a second event authority, queue, worker, retry policy, delivery store, CI gate, or closeout format.

## Read First

1. Complete the root `AGENTS.md` startup preflight and record the startup receipt.
2. Read `server/AGENTS.md`, `ai-plan/design/architecture/项目设计.md`, and
   `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`.
3. Read `ai-plan/design/governance/backend/队列采用与任务异步化治理规范.md` and the owning domain design. Read
   the task lifecycle design when the event is emitted by or affects a Task.
4. Inspect the canonical event type/owner, publisher transaction, handler registration, durable-delivery store, and
   concrete consumers. Read `web/AGENTS.md` when a consumer changes realtime, query cache, or caller-visible state.

## When To Use

Run before implementation during design and again during review when a change:

- adds, removes, renames, versions, publishes, subscribes to, or changes the payload/meaning of an event;
- changes `server/internal/event`, `server/internal/eventbus`, an Outbox/durable-delivery path, or consumer handler;
- makes a business fact asynchronously observable to audit, notification, logging, realtime, or another module; or
- adds or changes retry, idempotency, delivery receipt, external side effect, failure recovery, or event metrics.

Boot and design workflows should actively match this skill for these triggers. It runs by default for matching work,
not only when a user explicitly asks for an event review.

## Workflow

### 1. Choose The Correct Boundary And Owner

State the fact that occurred, its domain/module owner, and its canonical event contract. Decide whether it is:

- a `server/internal/event` fact event for synchronous receive, best-effort notification, or durable Outbox delivery;
- a synchronous, bounded `server/internal/eventbus` notification with no persistence guarantee; or
- a `server/modules/task` execution workflow that requires progress, cancellation, stages, queryable state, retry, or
  human recovery.

An event is not a command, a second read model, or a substitute Task. `eventbus` is never a durable queue. If the
event represents a stable cross-module fact, use the documented module or platform contract owner; do not duplicate a
string event name or payload model in a consumer.

### 2. Review Contract And Transaction Semantics

Verify that the envelope has a stable type, non-zero version, source, event identity, correlation/causation where
needed, and a deliberately minimal, desensitized payload. The payload conveys the fact, not an audit/notification
domain model, persistence entity, credential, unrestricted command, or mutable execution plan.

Determine when the fact becomes true. If loss across restart or a split between committed business data and consumer
observation would violate correctness, publish `DeliveryDurable` in the same SQL transaction through
`TransactionalPublisher.PublishTx`. A receipt means accepted for delivery, never that any consumer has completed.
Best-effort delivery is allowed only when loss does not change business correctness; record the loss tolerance.

### 3. Review Delivery, Idempotency, And Retry

For each consumer, identify its stable handler ID, the deduplication authority, and the behavior of a repeated event.
Handlers must be idempotent using a publisher-reused `Event.ID` or a business idempotency key backed by a constrained
persisted fact or the external provider's equivalent, not an in-memory map. When the same business fact can be
republished as a new envelope, require that durable business key; `Event.ID` alone cannot deduplicate it. Check delivery
at least once, lease loss, crash recovery, out-of-order arrivals, duplicate publish, and partial consumer failure.

Make retry bounded and owned by the existing delivery/runtime mechanism. Automatic retry requires evidence that the
handler and every external effect are safe to replay. Do not add a module-local goroutine, worker, DLQ, retry table,
or queue. For a long-running, cancellable, staged, or manually recoverable external effect, submit/notify a Task
instead and preserve Task Runtime ownership.

### 4. Review Consumer Boundaries And Observability

Confirm that consumers use only public module contracts and do not write another module's internal facts or Task
terminal state. A notification, audit record, realtime message, or cache invalidation is a consumer result and cannot
silently become the business authority. Require authz and audit review where a consumer performs sensitive action.

Trace the operational path: accepted, pending, claimed/running, completed, retrying, terminal failure, lease expiry,
backpressure, handler error, and manual recovery. Confirm correlation IDs connect structured logs, audit evidence, and
the caller-visible Task/realtime path without putting sensitive payload in them. Realtime is an incremental signal;
HTTP/persisted facts remain recovery authority.

### 5. Review Evolution And Evidence

Classify the event change as additive, behavior-changing, deprecated, breaking, or exception-with-expiry. Version a stable event rather than
silently changing its meaning. Before adding compatibility, identify the canonical owner, affected consumers, why a
coordinated mono-repo repair is not possible, expiry/cleanup trigger, and validation for both paths. Produce concise
evidence from handler-focused tests and the normal repository validation entrypoint; missing delivery or replay
evidence is a finding, not proof of safety.

## Findings And Decision Rules

Report findings as `blocking`, `high`, `medium`, or `note`, with the event/consumer, evidence, and one recommended
correction. A finding is blocking when an event has no authority owner, durable facts can be lost after commit,
handlers are not replay-safe, a consumer bypasses Task ownership or module boundaries, sensitive data is emitted, or a
second queue/worker/state machine is introduced. Prefer repairing the canonical fact/contract or deleting a redundant
event over adding a translation layer.

Use existing repository validation only. For server changes use `graft validate backend`; for shared contract or web
consumer changes validate both `graft validate backend` and `bun run check`. Focused event/handler tests supplement
these commands but do not form a second completion path.

## Review Output

```text
Event contract review:
- event_and_authority: <event type/version, domain owner, canonical contract path>
- boundary_choice: internal/event | eventbus | task, with reason
- fact_and_transaction: <when true, publisher, durable/best-effort, transaction evidence>
- contract: <source, correlation, payload classification, sensitive-data decision>
- consumers: <handler IDs, ownership, deduplication and side effects>
- delivery_and_recovery: <receipt, retry, lease, failure, manual recovery semantics>
- observability: <logs, metrics, audit, realtime/recovery authority>
- evolution: additive | behavior-changing | deprecated | breaking | exception-with-expiry
- findings: <severity, evidence, recommendation>
- validation: <commands and results>
```

## Guardrails

- `server/internal/event` owns fact-event delivery mechanics; `server/modules/task` owns managed execution lifecycle;
  neither grants a business module a new generic queue or worker.
- An event envelope and a consumer projection are derived representations, not competing authority for domain facts.
- Do not turn a receipt, realtime signal, notification, or successful publish into evidence that a consumer completed.
- Do not put credentials, unrestricted inputs, Ent entities, mutable plans, or unbounded external work in an event.
- Do not infer idempotency from a retry loop. Demonstrate the persisted or provider-backed deduplication boundary.
- When API, domain, permission, query cache, or cross-boundary representation changes too, invoke the corresponding
  Graft semantic review skill rather than duplicating its rules here.
