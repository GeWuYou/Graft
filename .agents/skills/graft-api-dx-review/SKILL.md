---
name: graft-api-dx-review
description: Default semantic review of Graft API caller experience across OpenAPI, Gin handlers, generated TypeScript, and Axios consumers. Use when adding or changing an HTTP API, shared request/response contract, pagination, errors, permissions, or asynchronous task receipt.
---

# Graft API DX Review

Use this skill as the default semantic review layer for API design work. It complements, and does not replace, the
canonical API boundary and OpenAPI governance, repository startup, or existing validation entrypoints. Do not create a
second API authority, compatibility workflow, CI gate, or closeout format.

## Read First

1. Complete the root `AGENTS.md` startup preflight and record the startup receipt.
2. Read `server/AGENTS.md` and `web/AGENTS.md` for cross-boundary API work.
3. Read `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`.
4. Read `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` when the change affects shared contract semantics.
5. Read the owning module descriptor, OpenAPI source, generated contract consumer, and the relevant API client/handler.

## When To Use

Run before implementation during design and again during review when a change:

- adds, removes, renames, or semantically changes an HTTP operation or field;
- changes shared pagination, filtering, sorting, enum, error, auth, or permission semantics;
- introduces or changes an asynchronous task, submission, polling, or receipt flow; or
- claims that an API change is compatible or has no downstream consumer impact.

The boot and design workflows should actively match this skill for these triggers. It is not limited to explicit user
requests for an API review.

## Workflow

### 1. Establish The Caller And Authority

Identify the primary caller (web module, shell, CLI, or external integration), the lifecycle/resource owner, and the
canonical OpenAPI or module-contract source. Trace web callers through:

`OpenAPI -> generated TypeScript -> Axios adapter -> query/store/composable -> view or task workflow`.

For shell, CLI, or external integrations, trace request construction, authentication, retry/timeout behavior, error
handling, and output or callback semantics through their actual adapter. Run that caller's existing validation command;
when none exists, record the limitation as missing evidence rather than assuming the web path covers it.

If a mismatch is caused by an upstream authority, report and repair that owner first when feasible. Do not add a
downstream mapping, alias, fallback, or compatibility DTO merely to hide authority drift.

### 2. Review The Public Shape

Check that:

- paths and operation names describe stable user-managed resources, not storage or runtime implementation details;
- request fields contain only caller-controlled input, with explicit requiredness, defaults, validation, and safe
  limits;
- responses express caller-facing meaning and do not expose Ent entities, internal fields, SQL, dependency output, or
  accidental transport details;
- list endpoints have consistent pagination, filtering, sorting, empty-result, and total/count semantics;
- enums, timestamps, nullable fields, and identifiers have one documented meaning across server and generated TS;
- errors use the stable code/messageKey/data envelope, safe public detail, and actionable status semantics; causes stay
  in server logs;
- permission failures and authentication requirements are explicit and do not rely on UI visibility alone.

### 3. Review Workflow Ergonomics

For mutations, verify idempotency and retry behavior, duplicate-submit handling, conflict/precondition semantics, and
whether the caller can safely refresh or recover after a timeout. For asynchronous work, verify that the response gives a
stable task/submission receipt, status lookup or subscription path, progress/error semantics, and a clear terminal
result. The API must not force each consumer to invent its own polling or error interpretation.

### 4. Review Consumer DX

Inspect generated TypeScript and the real Axios/TanStack Query consumer. Prefer inference from the generated contract;
avoid hand-written duplicate DTOs, broad `any`, unnecessary generic wrappers, and casts that make invalid calls easy.
Check that optionality, literal unions, error narrowing, pagination helpers, and mutation results produce useful
autocomplete and actionable compiler errors. Confirm query invalidation and task refresh behavior are discoverable at
the module boundary.

### 5. Review Evolution And Evidence

Classify each change as additive, behavior-changing, deprecated, or breaking. For non-additive changes, record why direct
authority repair or a coordinated mono-repo update is insufficient, affected consumers, and an expiry/cleanup trigger
before allowing compatibility. Produce a concise evidence bundle:

- authority owner and caller list;
- request/response and error-shape summary;
- pagination/async/auth/idempotency decisions;
- generated TypeScript and consumer impact;
- OpenAPI or contract diff plus focused validation results.

## Findings And Decision Rules

Report findings by severity: `blocking`, `high`, `medium`, or `note`. A finding is blocking when it creates ambiguous
caller behavior, leaks internal data, breaks a stable consumer, weakens permission/error guarantees, or introduces a
second source of truth. Prefer one concrete design correction over a compatibility bridge. If evidence is unavailable,
state the missing evidence instead of inferring safety.

Use existing repository validation only. For server changes use `graft validate backend`; for web changes use `bun run
check`; for shared contract changes validate both sides. Focused checks may supplement these entrypoints but must not
become a second completion path.

## Review Output

```text
API DX review:
- authority_owner: <OpenAPI/module contract path>
- primary_callers: <modules or integrations>
- operation_scope: <paths/operations>
- shape_decisions: <request/response/error/pagination summary>
- workflow_decisions: <auth/permission/idempotency/async summary>
- consumer_impact: <generated TS and concrete consumers>
- compatibility: additive | behavior-changing | deprecated | breaking | exception-with-expiry
- findings: <severity, evidence, recommendation>
- validation: <commands and results>
```

## Guardrails

- OpenAPI and the documented module contract remain the authority; generated TypeScript and UI code are consumers.
- Do not expose Entity models or internal error causes through the API.
- Do not invent a generic API adapter, polling framework, error taxonomy, or compatibility layer for one endpoint.
- Do not optimize for theoretical type sophistication; optimize for callers needing fewer annotations and casts.
- When the review identifies a domain, permission, event, query-key, or cross-boundary concern, invoke the corresponding
  Graft semantic review skill rather than duplicating its rules here.
