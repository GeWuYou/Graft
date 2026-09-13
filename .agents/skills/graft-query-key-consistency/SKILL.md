---
name: graft-query-key-consistency
description: "Review Graft query identity, normalization, invalidation, and realtime refresh when client cache behavior changes."
---

# Graft Query Key Consistency

Apply [root governance](../../../AGENTS.md) and reuse this task’s startup and validation evidence.

Use this skill as Graft's default semantic review layer for client cache identity. It complements the owning web
module contract, API/OpenAPI authority, TanStack Query conventions, and `bun run check`; it does not create a second
cache library, query registry, invalidation framework, or validation path.

## Relevant Evidence

Reuse startup and [web guidance](../../../web/AGENTS.md). Inspect the affected key factory, API contract, consumers, and mutation/invalidation paths. Read realtime, notification, or polling design only when that source supplies freshness to the changed query. Use relevant architecture/contract rules when identity or ownership changes, and compare neighboring factories before introducing a new shape. Keep the evidence in the integrated review.

## When To Use

Apply the relevant checks during design; review the resulting diff and new evidence without repeating unchanged work when a change:

- adds or changes `useQuery`, `useInfiniteQuery`, `useMutation`, `queryClient` calls, polling, or realtime refresh;
- adds a list, detail, saved-view, summary, count, permission, or resource query;
- changes filters, pagination, sorting, selected IDs, tenant/user scope, or normalization of query inputs; or
- changes a mutation, event, notification, websocket, or task transition that should refresh cached data.

Use the changed semantics to select checks; a file extension or incidental keyword alone does not require a full review.

## Workflow

### 1. Establish The Cache Authority

Identify the module and resource owner, API contract source, query-key factory (or the missing owner), and every known
consumer. Treat a query key as a public module identity: one resource has one stable namespace, with deliberate list,
detail, summary, and scoped variants. If a mismatch comes from an upstream contract or module owner, repair that owner
first; do not add aliases or duplicate keys to conceal drift.

### 2. Review Identity And Normalization

Check that:

- key segments use the canonical module/resource vocabulary and do not mix singular/plural or page-local synonyms;
- factories are colocated with the module's shared query API and return stable readonly tuples where inference helps;
- list filters are normalized once, with deterministic omission/default handling; compare normalized values and preserve
  array order where it carries identity rather than treating object reference churn as a query-key difference;
- detail keys contain the canonical identifier and never use an ambiguous empty or `undefined` ID as a real resource;
- user, tenant, permission, locale, or feature scope is included exactly when it changes the response; and
- list/detail/collection prefixes support intentional partial invalidation without invalidating unrelated resources.

Prefer `satisfies` or narrow literal inference for registries and key factories where it improves callers. Do not add
branded IDs, generic key builders, or a new cache abstraction without a concrete collision or caller-DX problem.

### 3. Review Reads, Mutations, And Invalidation

Trace each query from API call to view and each mutation to its affected cache entries. Verify create/update/delete,
bulk actions, task transitions, and permission changes invalidate or update every affected list, detail, count, summary,
and saved-view query. Prefer the owning factory in `invalidateQueries`, `setQueryData`, and `fetchQuery`; raw arrays are
findings unless the key is intentionally local and documented by the module boundary. Check optimistic updates,
rollback, stale responses, duplicate submission, and error paths for cache consistency.

### 4. Review Polling And Realtime Semantics

For polling, subscriptions, websocket events, notifications, or task progress, identify the event/resource owner and
the freshness guarantee. Confirm event payloads map to canonical keys, updates are idempotent, reconnects do not create
duplicate listeners, and terminal task states stop polling while refreshing the final detail/list projection. Do not
make every consumer invent event-to-key mappings; put the mapping at the owning module boundary.

### 5. Produce Evidence

Record the authority owner, key namespace/factory, normalized inputs, consumer inventory, mutation-to-invalidation map,
realtime/polling behavior, findings, and focused test/check results. Missing consumer or freshness evidence is a finding,
not proof of safety.

## Findings And Decision Rules

Report `blocking`, `high`, `medium`, or `note`. A finding is blocking when two keys identify the same resource
differently, a mutation leaves stale authoritative data, scope can leak across users/tenants, or realtime updates can
silently disappear. Prefer deleting a duplicate key or moving ownership to the shared module boundary over adding a
compatibility alias.

## Review Output

Include findings, authority/caller evidence, material decisions, and validation gaps in the task’s integrated review.
Use domain-specific fields only when they clarify a decision; do not repeat a separate full report for each skill.

## Guardrails

- The API/module contract and owning query module remain authoritative; query keys are consumers of resource identity.
- Do not use `invalidateQueries` as a substitute for understanding ownership or silently broaden invalidation to the
  whole cache.
- Do not expose transport-only fields or implementation-specific event names as UI cache identity.
- Use existing repository validation; `bun run check` remains the frontend completion entrypoint.
