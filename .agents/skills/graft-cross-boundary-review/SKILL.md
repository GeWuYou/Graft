---
name: graft-cross-boundary-review
description: Review Graft changes that cross server, OpenAPI, generated TypeScript, TanStack Query, composables, stores, and views for authority drift, duplicated transformations, and leaked layer responsibilities.
---

# Graft Cross-Boundary Review

Use this skill when a change crosses `server` and `web`, changes an OpenAPI or shared contract, updates generated
artifacts, or moves data through an API client, query/composable, store, or view. It is a semantic review aid, not a
replacement for `graft-boot`, domain-specific governance, code generation, or repository validation.

## Review Authority

Establish the canonical owner before reviewing implementation details:

* HTTP paths, operations, request/response schemas, and public wire enums: `openapi/**` source.
* Non-HTTP stable values and module semantics: the owning Go module contract, descriptor, or
  `server/internal/moduleapi/**` boundary.
* Generated Go bindings, bundled OpenAPI, generated web schemas, and TypeScript projections: derived artifacts.
* Query keys, cache policy, composables, stores, and view models: the owning web module, constrained by the canonical
  wire contract.

If the symptom is downstream from an authority drift, report and repair the highest broken authority in the task
scope. Do not add an alias, fallback, adapter, or compatibility DTO merely to hide a mismatch.

## Workflow

1. Map the changed value or behavior from source authority to HTTP/OpenAPI, generated outputs, API client, query or
   composable, store, and view. Record each transformation and its owner.
2. Check contract closure: names, optionality, nullability, enum values, error shape, pagination, async/task receipt,
   permissions, and lifecycle states must remain equivalent across the chain.
3. Check boundary ownership:
   * server code does not expose Ent entities, database edges, or internal implementation details;
   * generated files are not hand-edited or treated as authority;
   * web code does not recreate API payload types or server-only semantics;
   * page-local presentation mapping is not promoted into a second shared contract.
4. Check transformation economy. Remove pass-through DTOs, duplicate mappers, redundant stores, and one-off query
   wrappers when they add no policy, caching, validation, or presentation value. Keep a mapping only when it protects a
   stable boundary or expresses a real view concern.
5. Check query and reactive behavior: query keys are module-owned and consistent, invalidation/refetch follows the
   mutation or realtime contract, loading/error/empty states preserve the API semantics, and no stale local copy can
   become a competing source of truth.
6. Check caller ergonomics: generated TypeScript inference should flow through API helpers and composables; avoid
   broad `any`, unnecessary casts, generic wrappers that erase inference, and literal widening in registries or maps.
7. Review failure and compatibility paths: distinguish server errors from transport errors, preserve actionable error
   codes/messages, and document any unavoidable compatibility bridge with owner, reason, consumers, and cleanup trigger.
8. Run the required server/web or cross-boundary validation entrypoints after the semantic review. Regenerate and check
   generated artifacts only when their canonical source or a related generated projection is in the changed chain;
   never validate only a manually edited projection. For query, composable, store, or view-only changes with no such
   projection, run the affected server/web validation without creating unrelated generated diffs.

## Findings Format

Report findings first, ordered by severity: `blocking`, `high`, `medium`, or `note`. Each finding includes:

* boundary and affected value/operation;
* canonical authority and observed projection path;
* concrete drift, duplicated responsibility, or leaked abstraction;
* user/maintainer impact;
* smallest authority-first repair;
* validation evidence or missing validation.

Conclude with `authority_summary`, `projection_map`, `accepted_mappings`, `compatibility_bridges`, and `validation_gap`.
If no findings exist, say so explicitly and list residual test or generation risk.

## Boundaries

* Do not introduce a second source of truth, second generation path, or local compatibility layer to make a review pass.
* Do not redesign APIs, query libraries, stores, or module boundaries solely because a pattern looks different; require
  a concrete semantic, maintenance, or caller-experience benefit.
* Keep this skill compatible with existing Graft startup, worktree, authority-escalation, comment, and closeout rules.
