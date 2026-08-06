---
name: graft-domain-model-review
description: Default semantic review of Graft domain language, aggregate boundaries, ownership, lifecycle transitions, and scenario invariants. Use during design and review of domain behavior, state machines, persistence facts, or cross-module workflows.
---

# Graft Domain Model Review

Use this skill as Graft's default semantic review layer for domain behavior. It complements the owning module,
`ai-plan/design/domains/**`, architecture documents, API contracts, and repository validation; it does not create a
second domain authority, issue tracker, `CONTEXT.md`, or design/closeout workflow.

## Read First

1. Complete the root `AGENTS.md` startup preflight and record the startup receipt.
2. Read `server/AGENTS.md` for backend domain or persistence changes and `web/AGENTS.md` for frontend workflow
   changes; read both for cross-boundary behavior.
3. Read the owning domain README and canonical design documents under `ai-plan/design/domains/<domain>/`.
4. Read the relevant architecture, contract, security, database, notification, or task-lifecycle governance documents
   named by the change.
5. Inspect the owning module descriptor, public module API, persistence schema, OpenAPI source, and concrete callers.

## When To Use

Run before implementation during design and again during review when a change:

- introduces or renames a business concept, resource, command, event, or module boundary;
- adds or changes an aggregate, entity relationship, ownership rule, invariant, or persistence fact;
- adds, removes, or reorders lifecycle states, transitions, retries, leases, cancellation, expiration, or recovery; or
- changes a cross-module workflow such as task submission, build execution, scheduler ownership, audit, notification,
  or runtime reconciliation.

Boot and design workflows should actively match this skill for these triggers. It runs by default for matching work,
not only when a user explicitly asks for a domain review.

## Workflow

### 1. Establish Language And Authority

List the terms used by the change and their precise meanings. Identify the domain/module owner, the aggregate root (if
one exists), the canonical persisted or contract fact, and the authoritative design document. Search for existing
names before introducing a synonym. If the mismatch is caused by an upstream authority, report and repair that owner
first when feasible; do not hide it with a mapper, alias, or compatibility model.

### 2. Test The Boundary And Invariants

For every proposed aggregate or service, state:

- what it owns and what it deliberately does not own;
- which invariants must hold atomically and which can be eventual;
- which commands or public interfaces are allowed to mutate it;
- which data is a reference to another aggregate rather than a copied fact; and
- whether the interface is deep enough to protect callers from persistence or runtime details.

Check that module dependencies use documented public interfaces (`server/internal/moduleapi` where appropriate), not
another module's internal implementation. A database table, API DTO, generated type, UI model, or event is not an
aggregate merely because it has a name.

### 3. Walk Scenarios And State Machines

Write the smallest representative scenarios before accepting a model: create/submit, duplicate request, normal
progression, retry, concurrent mutation, cancellation, timeout/expiration, failure, recovery, and terminal cleanup.
For each lifecycle state, record valid incoming transitions, actor/owner, side effects, idempotency key or fencing
version, emitted events, and the observable terminal meaning. Reject states that only describe an implementation step
without a caller or recovery semantics. Check that invalid transitions fail at the owner boundary rather than being
silently normalized by consumers.

### 4. Check Cross-Boundary Representations

Trace the domain fact through its representations:

`domain/module -> persistence -> OpenAPI or event contract -> generated TypeScript -> query/store/composable -> view`.

Confirm each layer preserves the domain meaning, names, requiredness, enum/state semantics, ownership, and error
behavior. Prefer a canonical fact plus narrow projections; flag duplicate DTOs, copied state machines, presentation
models that can mutate domain state, or adapters that conceal authority drift. Invoke the API, event, permission,
cross-boundary, query-key, or TypeScript DX semantic skill when its specialized review is triggered.

### 5. Align Design And Evidence

Compare the result with the canonical domain document and ADR/design decisions. Update the owning design authority in
the same slice when the model or lifecycle changes; do not record a competing summary in a skill, module README, or
ad-hoc note. Classify evolution as additive, behavior-changing, deprecated, breaking, or exception-with-expiry; the
last classification must record the compatibility bridge's cleanup trigger. State migration, compatibility, cleanup,
and observability implications. Produce evidence from focused tests and the repository's
normal validation entrypoint; missing evidence is a finding, not proof of safety.

## Findings And Decision Rules

Report findings as `blocking`, `high`, `medium`, or `note`, each with the violated invariant or term, concrete evidence,
and one recommended correction. A finding is blocking when ownership is ambiguous, an invariant can be bypassed, a
state transition has no recovery/terminal meaning, a module boundary leaks internals, or the change creates a second
source of truth. Prefer deleting a redundant model or state over adding a translation layer. Compatibility requires the
canonical authority, reason direct repair is not possible now, affected consumers, expiry/cleanup trigger, and
validation of both bridge and cleanup path.

## Review Output

```text
Domain model review:
- domain_and_authority: <domain/module and canonical ai-plan design path>
- vocabulary: <terms, definitions, renamed/synonym decisions>
- aggregate_and_ownership: <roots, boundaries, actors, invariants>
- lifecycle: <states, valid transitions, retries, recovery, terminal cleanup>
- scenarios: <normal, duplicate, concurrent, failure, cancellation, expiration>
- representations: <persistence/contract/event/TypeScript/UI alignment>
- evolution: additive | behavior-changing | deprecated | breaking | exception-with-expiry
- findings: <severity, evidence, recommendation>
- validation: <commands and results>
```

## Guardrails

- `ai-plan/design/domains/**` and the owning module contract remain the domain design authority.
- Do not introduce DDD ceremony, repositories, value objects, aggregates, CQRS, or event sourcing without a concrete
  invariant or boundary that benefits from it.
- Do not use branded TypeScript IDs or duplicate lifecycle unions merely to make a model look more typed; preserve the
  canonical contract and optimize for caller clarity.
- Do not expose Ent entities, storage fields, lease tokens, internal causes, or runtime implementation details as
  business concepts without an explicit public contract decision.
- Do not claim a domain design is sound from lint alone; semantic review needs scenarios, invariant evidence, and
  focused tests.
