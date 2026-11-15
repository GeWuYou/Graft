---
name: graft-typescript-dx-audit
description: "Review inference, literal types, generics, and caller safety when changing Graft public TypeScript contracts or shared APIs."
---

# Graft TypeScript DX Audit

Apply [root governance](../../../AGENTS.md) and reuse this task’s startup and validation evidence.

Use this skill as Graft's default semantic review layer for TypeScript developer experience. It complements generated
OpenAPI contracts, frontend module ownership, contract governance, and existing validation; it does not create a
second type authority, utility-type library, frontend architecture, type-test framework, or completion gate.

## Relevant Evidence

Reuse startup, web guidance, and prior contract evidence. Inspect the changed public type and actual callers first.

- Wire types or shared semantic changes: inspect the canonical OpenAPI/module contract and the affected [contract governance](../../../ai-plan/design/governance/platform/契约治理与魔法值治理规范.md).
- Module ownership or layering changes: consult [frontend architecture](../../../ai-plan/design/architecture/前端架构设计.md).
- Query, permission, lifecycle, or API concerns: cover the changed concern in the integrated review and load its specialized skill only for missing detail.

An internal TypeScript edit without public-contract or caller impact does not require this complete workflow.

## When To Use

Apply the relevant checks during design; review the resulting diff and new evidence without repeating unchanged work when a change:

- adds or changes a handwritten TypeScript public API, module contract, composable, store, query helper, or API adapter;
- adds casts, `any`, `unknown` narrowing, generic wrappers, utility types, literal objects, discriminated unions, or
  branded types;
- duplicates, maps, or projects generated OpenAPI types across a module boundary; or
- makes callers supply explicit type arguments, repeat annotations, or work around unclear autocomplete/errors.

Use the changed semantics to select checks; a file extension or incidental keyword alone does not require a full review.

## Workflow

### 1. Start With The Caller And Canonical Type

Identify the concrete callers and the canonical owner of each value/shape. Generated OpenAPI contracts remain the
wire-contract authority; module `contract/**` owns stable module-facing semantics; `types/**` is normally private.
Trace the type from its authority through Axios, query/store/composable, and view. If a local type exists only because
an upstream authority drifted, repair the authority or use a narrow presentation projection rather than creating a
parallel DTO.

Ask the caller-first questions: can an ordinary caller invoke this without a type argument, cast, or duplicate type
annotation; does autocomplete present the valid choices; and does an invalid call fail where the caller made it? A
clever type that obscures this is a defect, not an improvement.

### 2. Preserve Useful Literals Without Over-Modeling

Use `as const` for a module-owned, canonical closed set whose literal values downstream callers need, then derive its
union from that single value. Use `satisfies` to validate registries, route/menu metadata, configuration maps, and
contract tables while preserving their inferred literals. Avoid annotation forms that immediately widen useful values
to `string`, but do not freeze incidental arrays, mutable UI state, or framework payloads merely to obtain a narrow
type.

Prefer a literal union or generated enum for a real closed state set. Use a discriminated union only when variants
carry different required data or cause genuinely different control flow, such as a caller-visible lifecycle or mutation
state. Align its discriminant with the canonical server/OpenAPI contract; do not copy a domain state machine into a
view model without a concrete presentation need.

### 3. Design Inference-First Interfaces

Let TypeScript infer from function arguments, callback parameters, return values, and TanStack Query/Axios contract
types. Order generic parameters so inference uses common required inputs first, give safe defaults only where they do
not hide a mistake, and keep public generic counts small. Do not wrap `useQuery`, `useMutation`, Axios, refs, or
generated clients in generic helpers that erase their error/data/key inference merely for a shorter call site.

Use utility types as a local projection from an authoritative model (`Pick`, `Omit`, `Partial`, `Readonly`, mapped
types) when their resulting name and error messages remain clearer than a dedicated type. Do not build multi-stage
conditional/mapped-type pipelines, universal `Result` wrappers, or generic factories unless repeated real callers
demonstrably need the abstraction.

### 4. Restrict Boundary Types And Escape Hatches

Use `unknown` at untrusted or opaque runtime boundaries and narrow it immediately with a local guard or schema; do not
let it flow into views. `any`, broad assertions, double casts, and non-null assertions require a localized reason and
must not become an adapter boundary. Prefer structural types for ordinary module data.

Do not introduce branded IDs globally. A brand is permitted only at a high-risk infrastructure boundary where two
same-shaped values have repeatedly been confused, construction/validation is centralized, runtime representation stays
unchanged, and ordinary callers do not need scattered `as Brand` casts. Route/storage/request infrastructure may meet
that threshold; business IDs usually do not. Reject brands that merely make code look more typed.

### 5. Validate DX And Evolution

Inspect actual caller inference in the IDE/compiler and ensure errors identify the invalid argument or missing case.
For a stable public helper, add a focused compile-time regression test only when inference itself is a meaningful
contract; do not introduce a type-test harness for incidental local implementation. Confirm runtime validation still
exists for untrusted values: static types never replace server validation, OpenAPI, or permission checks.

Classify public type changes as additive, behavior-changing, deprecated, or breaking. Coordinate canonical changes
across server and web instead of preserving old types through aliases by default. Use `bun run check` for web changes;
validate both web and server when shared contracts change. Focused tests supplement, not replace, the repository
completion entrypoint.

## Findings And Decision Rules

Report findings as `blocking`, `high`, `medium`, or `note`, including caller evidence and one simpler correction. A
finding is blocking when a hand-written type competes with canonical OpenAPI/module contract authority, an assertion
allows an invalid request or lifecycle state to cross a boundary, a public helper erases essential inference, or a new
type layer conceals cross-module ownership drift. Prefer deleting a duplicate type, cast, or wrapper over adding a more
advanced type construction.

## Review Output

Include findings, authority/caller evidence, material decisions, and validation gaps in the task’s integrated review.
Use domain-specific fields only when they clarify a decision; do not repeat a separate full report for each skill.

## Guardrails

- Optimize for callers needing fewer annotations, casts, and source-code archaeology, not for maximal type cleverness.
- Generated OpenAPI types and documented module contracts remain authoritative; never patch generated types or recreate
  their DTOs by hand to make a local type easier to use.
- `satisfies` checks conformance without a reason to widen values; `as const` preserves a canonical closed set, not an
  excuse to make ordinary state immutable.
- Do not use branded IDs, discriminated unions, generic abstractions, or type-level tests unless they prevent a proven
  boundary mistake or materially improve repeated caller ergonomics.
- Static typing does not authorize trust in frontend input, replace server validation, or weaken permission checks.
- When the audit finds a contract, query key, domain, API, permission, event, or cross-boundary concern, cover that concern in the integrated review; load its specialized skill only for needed detail.
