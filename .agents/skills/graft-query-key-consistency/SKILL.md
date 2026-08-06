---
name: graft-query-key-consistency
description: Default semantic review of Graft TanStack Query key ownership, normalization, invalidation, and realtime refresh behavior. Use when adding or changing web queries, mutations, cache updates, polling, subscriptions, or shared resource adapters.
---

# Graft Query Key Consistency

Use this skill as Graft's default semantic review layer for client cache identity. It complements the owning web
module contract, API/OpenAPI authority, TanStack Query conventions, and `bun run check`; it does not create a second
cache library, query registry, invalidation framework, or validation path.

## Read First

1. Complete the root `AGENTS.md` startup preflight and record the startup receipt.
2. Read `web/AGENTS.md` and the relevant frontend architecture and contract governance documents.
3. Read the owning module's API/contract files, existing shared query helpers, and any event, notification, polling, or
   realtime design that supplies fresh data.
4. Inspect neighboring modules for established query-key factories before introducing a new shape.

## When To Use

Run before implementation during design and again during review when a change:

- adds or changes `useQuery`, `useInfiniteQuery`, `useMutation`, `queryClient` calls, polling, or realtime refresh;
- adds a list, detail, saved-view, summary, count, permission, or resource query;
- changes filters, pagination, sorting, selected IDs, tenant/user scope, or normalization of query inputs; or
- changes a mutation, event, notification, websocket, or task transition that should refresh cached data.

Boot and design workflows should actively match this skill for these triggers. It runs by default for matching work,
not only when a user explicitly asks for a query review.

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
- list filters are normalized once, with deterministic omission/default handling and no accidental object identity churn;
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

```text
Query key consistency review:
- authority_owner: <module contract/query owner>
- key_namespace: <canonical prefixes and factories>
- normalization: <filter/scope/ID rules>
- consumers: <queries, mutations, views, realtime sources>
- invalidation_map: <mutation/event -> affected keys>
- freshness: <polling/subscription/reconnect/terminal behavior>
- findings: <severity, evidence, recommendation>
- validation: <bun run check or focused tests and results>
```

## Guardrails

- The API/module contract and owning query module remain authoritative; query keys are consumers of resource identity.
- Do not use `invalidateQueries` as a substitute for understanding ownership or silently broaden invalidation to the
  whole cache.
- Do not expose transport-only fields or implementation-specific event names as UI cache identity.
- Use existing repository validation; `bun run check` remains the frontend completion entrypoint.
