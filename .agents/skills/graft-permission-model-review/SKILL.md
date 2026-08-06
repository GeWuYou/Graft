---
name: graft-permission-model-review
description: Default semantic review of Graft permission, protected route, menu/bootstrap visibility, capability, and audit alignment. Use when adding or changing permissions, roles, routes, menus, authorization checks, or security-sensitive UI actions.
---

# Graft Permission Model Review

Use this skill as Graft's default semantic review layer for authorization semantics. It complements the canonical
permission registry/module contract, backend security governance, navigation governance, bootstrap aggregation, audit
design, and normal validation; it does not create a second policy engine or client-side security authority.

## Read First

1. Complete the root `AGENTS.md` startup preflight and record the startup receipt.
2. Read `server/AGENTS.md` and `web/AGENTS.md` for cross-boundary permission or route work.
3. Read `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md`, the navigation/route governance document,
   and notification/audit design documents when affected.
4. Inspect the owning module descriptor and contract permission constants, server permission registry, protected route
   guards, menu metadata, user bootstrap filtering, generated/API contract, and concrete web capability checks.

## When To Use

Run before implementation during design and again during review when a change:

- adds, removes, renames, or changes a permission, role, resource, action, risk level, or ownership rule;
- adds or changes a protected HTTP route, menu item, route metadata, bootstrap capability, or object action;
- changes frontend visibility, disabled-state, authorization directives, or API error handling for forbidden actions; or
- changes an auditable operation, dangerous action, impersonation, credential, task control, or permission assignment.

Boot and design workflows should actively match this skill for these triggers. It runs by default for matching work,
not only when a user explicitly asks for a permission review.

## Workflow

### 1. Establish Authority And Trust Boundaries

Identify the module/lifecycle owner and canonical permission code. Trace one capability through:

`module descriptor/contract -> permission registry -> route guard/service authorization -> menu/bootstrap -> web route/page/action -> audit event`.

The backend authorization boundary is authoritative. UI hiding, route metadata, or a bootstrap permission list is an
ergonomic projection, never a security check. If drift is upstream, repair the authority first; do not add aliases or
fallback permission mappings without a recorded exception.

### 2. Review Permission Semantics

Check that:

- codes are stable, module-owned, and named by resource plus action rather than UI labels or implementation details;
- read, create, update, destructive, task-control, and administrative actions have deliberately separated scopes;
- permission metadata has the correct module, resource, action, risk level/category, display key, and description key;
- role assignment and inheritance behavior are explicit, and deny-by-default remains intact;
- object ownership, tenant scope, actor identity, and elevated/system operations are enforced at the server boundary;
- route guards protect every method/path, including detail, mutation, bulk, async, and retry/cancel operations; and
- authentication failures and authorization failures preserve the canonical public error semantics without leaking causes.

Do not create one broad permission merely to simplify wiring, or client-only checks that imply authorization.

### 3. Review Menu, Route, And Bootstrap Alignment

Compare menu `Code`, `ParentCode`, `Path`, `Permission`, and module ownership with the canonical navigation classification
and route contract. Confirm bootstrap filtering removes unauthorized entries without changing the server policy, direct
URL/API access remains denied, and route names/URLs are stable object boundaries rather than runtime implementation
paths. Object capabilities belong on the owning detail surface; they should not become unrelated first-level menus.

For each visible action, verify the web capability source, loading/forbidden/disabled states, mutation error handling,
and query invalidation. Do not duplicate permission catalogs or hard-code permission strings in unrelated pages.

### 4. Review Audit And Dangerous Operations

For writes, destructive actions, credential/role changes, task controls, and impersonation, identify the audit owner,
actor/resource fields, outcome, correlation/request ID, and failure behavior. Confirm audit emission occurs at the
trusted server operation boundary, is not fabricated by the browser, and cannot be bypassed by alternate routes or jobs.
Check confirmation, re-authentication, idempotency, concurrency, and recovery requirements for dangerous operations.

### 5. Produce Evidence

Record the authority owner, permission matrix, protected route inventory, menu/bootstrap/web projections, audit mapping,
threat/trust-boundary assumptions, findings, and focused validation results. Missing route or audit evidence is a finding.

## Findings And Decision Rules

Report `blocking`, `high`, `medium`, or `note`. A finding is blocking when a protected operation is reachable without a
server check, a permission code has ambiguous ownership, UI projection diverges from the contract, actor/resource scope
can be bypassed, or a dangerous operation lacks trusted audit evidence. Prefer repairing the module contract and shared
registries over compatibility maps.

## Review Output

```text
Permission model review:
- authority_owner: <module contract/permission registry>
- permission_matrix: <resource/action/risk/owner summary>
- protected_routes: <methods and paths with guards>
- navigation_projection: <menu/route/bootstrap/web capability alignment>
- trust_boundary: <server checks, actor/resource scope, deny-by-default>
- audit_and_dangerous_ops: <events, fields, confirmations, recovery>
- findings: <severity, evidence, recommendation>
- validation: <graft validate backend, bun run check, or focused results>
```

## Guardrails

- Permission contracts, server authorization, and audit policy remain authoritative; web visibility is not security.
- Do not add a second policy engine, client-side allowlist, or permission alias to mask drift.
- Do not expose internal authorization causes or sensitive resource details in public errors.
- Cross-boundary changes require both server and web validation through the repository entrypoints.
