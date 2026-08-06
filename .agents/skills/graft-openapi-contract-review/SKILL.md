---
name: graft-openapi-contract-review
description: Review Graft OpenAPI source changes for contract quality, compatibility, and server/web impact before implementation or regeneration. Use when openapi/**/*.yaml, shared API contracts, or their generated consumers change.
---

# Graft OpenAPI Contract Review

Use this skill for semantic review of Graft's HTTP wire contract. It complements repository validation and does not
create a second OpenAPI authority, generation path, startup flow, or completion gate.

## Authority And Boundaries

- `openapi/**/*.yaml` is the shared wire-contract authority. Confirm the owning module or platform contract before
  editing a path, schema, parameter, response, security scheme, or reusable component.
- `web/src/contracts/openapi/generated/**`, `server/internal/contract/openapi/**`, bundled specs, and other generated
  outputs are derived artifacts. Never repair source drift by hand-editing them; update the source input and regenerate.
- Keep HTTP/OpenAPI DTOs separate from Ent entities and internal persistence details. Stable cross-module semantics belong
  in the documented server/web contract boundaries, with one canonical owner and an explicit derived mapping.
- A change that affects both server and web, or changes route, permission, lifecycle, or shared contract semantics, is
  cross-boundary and must be reviewed and validated as such.

## Review Workflow

1. Establish the startup receipt and read the applicable root, server, web, and contract-governance instructions.
2. Inventory the source diff and classify each change as additive, corrective, behavioral, deprecation, or breaking.
3. Identify the canonical owner, lifecycle (`experimental`, `stable`, `deprecated`, or `removed`), direct consumers, and
   whether generated artifacts are expected to change.
4. Review the contract checklist below. Record concrete findings and unresolved decisions before implementation.
5. Trace affected server handlers/services and web module `api/**`, generated types, stores, composables, and pages. Look
   for duplicate DTOs, lossy mappings, boundary leakage, stale names, and assumptions about optionality or defaults.
6. Run the existing repository OpenAPI checks and the smallest relevant server/web validation. Report failures using the
   repository's normal validation entrypoint; do not invent a skill-specific gate.

## Semantic Checklist

### Breaking And Compatibility

- Removing or renaming paths, operations, parameters, properties, enum values, security requirements, or response codes
  is breaking unless every consumer and compatibility policy explicitly permits it.
- Tightening required fields, narrowing types/formats/patterns, changing nullability, changing defaults, or changing
  pagination/order semantics is treated as potentially breaking and requires consumer evidence.
- Adding a required request field, changing an existing response meaning, or reusing a field for a new lifecycle state is
  not an additive change merely because the document still parses.
- Compatibility bridges are exceptions: record the canonical authority, why direct repair is not possible, affected
  consumers, expiry/cleanup trigger, and validation for both bridge and cleanup. Prefer repairing the authority directly.
- Deprecation must identify the replacement, consumer impact, removal condition, and lifecycle owner.

### Naming And Resource Design

- Use stable domain/resource nouns and consistent pluralization; do not expose implementation technology, database names,
  or UI wording as resource identity.
- Keep path parameters, operation IDs, schema names, property names, and enum members consistent with existing module
  vocabulary and casing. Avoid synonyms that create parallel concepts.
- Model collection, item, action, batch, and nested-resource semantics explicitly. Do not encode an action as a resource
  noun or make an unrelated resource hierarchy merely to match a screen.
- Choose HTTP methods and status codes that describe the state transition. Make idempotency and retry behavior explicit
  for mutating or asynchronous operations.

### Schemas, Errors, Enums, And Pagination

- Request and response shapes must reflect the owning boundary; do not leak Ent edges, internal IDs, persistence flags, or
  server-only implementation fields.
- Required/optional/null semantics, formats, bounds, examples, and defaults must match runtime validation and mapping.
- Errors should use the established envelope/error-code contract, stable machine-readable codes, actionable messages, and
  documented status semantics. Do not make clients parse display text.
- Enums represent a genuinely closed, authority-owned set. Adding a value requires checking tolerant consumers; removing
  or repurposing one is breaking. Document unknown-value behavior where open-ended evolution is expected.
- Pagination, filtering, sorting, totals, cursors, and empty results must be consistent with neighboring endpoints and
  explicit enough for generated clients and UI query state.

### Security And Consumer Impact

- Every protected operation has the correct security scheme, permission/capability owner, actor/resource checks, and
  audit implications. Do not infer authorization from UI visibility.
- Check authentication, authorization failures, sensitive fields, caching, and write replay risks for new or changed data.
- Trace server route/DTO/mapper consumers and web generated `paths`/`components` consumers. Confirm frontend modules use
  the approved request boundary and do not add handwritten API duplicates, generated runtime clients, or extra Axios
  instances.
- Identify migration, rollout, cache, realtime, job, and notification effects when the contract changes lifecycle or
  asynchronous behavior.

## Validation And Output

Use the repository's current commands, normally `just openapi-check`, `cd web && bun run openapi:types:check`,
`cd web && bun run openapi:frontend-governance:check`, and the applicable backend freshness/boundary checks. For a
cross-boundary change, follow `graft-validation-runner` and validate both sides. If a command is unavailable, state the
exact command and limitation; never claim semantic review is replaced by lint or generation success.

Return a concise review containing:

- `authority`: canonical source and lifecycle owner;
- `change_class`: additive, corrective, behavioral, deprecation, or breaking;
- `consumer_impact`: affected server/web modules and generated outputs;
- `findings`: severity (`blocker`, `warning`, `note`), location, evidence, and recommended repair;
- `validation`: exact commands and results;
- `decisions_needed`: unresolved design or compatibility choices;
- `acceptance`: conditions required before merge.

Do not commit or modify generated artifacts as a substitute for source repair. Follow the caller's normal Graft
worktree, validation, comment, and closeout skills for implementation and integration.
