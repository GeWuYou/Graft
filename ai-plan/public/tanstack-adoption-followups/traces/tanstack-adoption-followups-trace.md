# TanStack Adoption Follow-ups Trace

## 2026-07-15 Work Intake And Baseline

- Work Intake classified remaining TanStack adoption as long-running frontend architecture evolution: design and an active topic are required; roadmap and ADR are not.
- Added AI design and review guidance that makes Query the first evaluation for server-state duplication, manual request state, refresh logic, polling, and realtime-triggered HTTP refresh.
- Recorded P0 baseline commit `5acd17d8`: shared QueryClient plus announcement, monitor, access-log, app-log, and audit migrations.
- Deferred Table, Virtual, Router, and Form migration by default; they require separately recorded evidence rather than framework symmetry.

## 2026-07-16 P1 Standard CRUD Query Migration

- Migrated `web/src/modules/user/pages/index.vue` user-list server state from a page-local request ref to the
  module-owned `userQueryKeys.list()` Query cache.
- User create, edit, status, delete, and single/batch role assignment now update the affected cached list snapshot;
  reset-password does not alter the list snapshot and intentionally has no cache mutation.
- Kept filters, pagination, row selection, drawers, form drafts, and the lazy role catalog outside the list cache.
  The role catalog remains an interaction-local fetch because this batch did not establish a cross-page snapshot need.
- Reference: TanStack Query documents direct cache updates from mutation responses and stable query-key ownership at
  https://tanstack.com/query/latest/docs/framework/vue/guides/updates-from-mutation-responses and
  https://tanstack.com/query/latest/docs/framework/vue/guides/query-keys.

## 2026-07-16 P1 Resource Detail Query Migration

- Migrated the standalone container resources page's images, networks, volumes, and system HTTP snapshots to
  module-owned `containerResourceQueryKeys` and `useDockerResourceQueries`. The active tab remains local UI state,
  while each static snapshot is fetched only after its tab becomes active and is held solely in the Query cache.
- Did not migrate project/container list and detail surfaces: they coordinate realtime subscriptions, polling, log
  streams, command state, and editor drafts. Those concerns require a separate authority and lifecycle design; moving
  them into Query merely for consistency would duplicate or obscure their existing state boundaries.
- Reference: TanStack Query requires query keys to identify cached data independently and documents dependent/lazy
  query enabling at https://tanstack.com/query/latest/docs/framework/vue/guides/query-keys and
  https://tanstack.com/query/latest/docs/framework/vue/guides/dependent-queries.

## 2026-07-16 Non-Query Go/No-Go And Archive Readiness

- Decided no-go for TanStack Table, Virtual, Form, and Router. `web/package.json` and `web/bun.lock` do not contain
  the four packages, while the current authorities remain TDesign Table/Form, Vue Router, and the existing LogViewer
  realtime/virtualized behavior. No concrete measured performance or maintenance deficiency was found.
- This is not a permanent prohibition. Table, Virtual, or Form may be reconsidered only after an owner records a
  reproducible baseline, target interaction/data shape, before/after acceptance metrics, and a rollback that removes
  the dependency and restores the current authority. Useful upstream references are the [Table overview](https://tanstack.com/table/latest/docs/overview),
  [Virtual introduction](https://tanstack.com/virtual/latest/docs/introduction), and [Form Vue quick start](https://tanstack.com/form/latest/docs/framework/vue/quick-start).
- Router remains no-go unless frontend architecture authority first changes. A future proposal must prove a Vue Router
  route-registry or navigation deficiency, define route/navigation metrics, and preserve a reversible migration path;
  see the [TanStack Router Vue quick start](https://tanstack.com/router/latest/docs/framework/vue/quick-start).
- Final archive-readiness check passed: all three P1 batches are complete; Query acceptance conditions are recorded as
  met, and any non-Query adoption is now guarded by evidence, metrics, and rollback. The Work Contract has
  `closeout.archive: false`, so the topic remains in place and is marked `archive-ready` without changing the public
  topic index.

## 2026-07-16 Reopened Evidence Audit

- Reopened the topic after finding a remaining page-local notification list HTTP snapshot. The bounded
  `notification-list-query-migration` moves only that paginated list to a module-owned key; the header bell, route
  filters, detail drawer, selection, and mutation flags remain outside Query cache.
- Pending follow-up batches are `rbac-query-migration`, `system-config-query-migration`, and
  `remaining-query-no-go-review`. The previously recorded Table, Virtual, Form, and Router no-go decision remains in
  force unless a new evidence-backed proposal meets its metrics and rollback requirements.

## 2026-07-16 Notification List Query Migration

- Migrated `web/src/modules/notification/pages/list/index.vue` from page-local rows, total, loading, error, and
  refresh state to `notificationQueryKeys.list(normalizedQuery)`. Its query function calls only the existing
  `getNotifications` module API wrapper.
- Route filters, pagination controls, detail drawer state, selected record, mutation flags, and header refresh event
  remain outside the Query cache. A single-read response updates cached rows directly; unread-filtered pages and
  bulk/delete changes invalidate module list variants so pagination stays server-authoritative.
- Reference: [TanStack Query keys](https://tanstack.com/query/latest/docs/framework/vue/guides/query-keys),
  [updates from mutation responses](https://tanstack.com/query/latest/docs/framework/vue/guides/updates-from-mutation-responses),
  and [query invalidation](https://tanstack.com/query/latest/docs/framework/vue/guides/query-invalidation).

## 2026-07-16 RBAC Query Migration

- Migrated RBAC role lists, the role-editor permission catalog, and filtered permission lists to module-owned Query
  keys. Permission-list keys normalize empty filters while API requests keep empty parameters omitted; clearing a
  filter therefore restores the matching cached snapshot instead of issuing a duplicate request.
- Role create, update, status, and delete responses precisely update the cached role list. Role-permission mutations
  invalidate only that list because the mutation endpoint does not return the authoritative role summary. Permission
  detail, role detail, and role-permission binding calls remain drawer-session requests so their loading and draft
  lifecycle cannot overwrite the active editor session.
- Reference: [Query keys](https://tanstack.com/query/latest/docs/framework/vue/guides/query-keys),
  [updates from mutation responses](https://tanstack.com/query/latest/docs/framework/vue/guides/updates-from-mutation-responses),
  and [query invalidation](https://tanstack.com/query/latest/docs/framework/vue/guides/query-invalidation).

## 2026-07-16 System Config Query Migration

- Migrated the system-config collection from page-local `items`, `loading`, and error state to the module-owned
  `systemConfigQueryKeys.list()` Query snapshot. Its query function calls only the existing `getSystemConfigs` module
  API wrapper.
- Group search and selection, expanded tree nodes, editor visibility and drafts, schema validation, and saving/reset
  flags remain page-local. Update and reset API responses precisely replace the matching cached item, rather than
  retaining a second mutable collection in the page.
- Reference: [TanStack Query keys](https://tanstack.com/query/latest/docs/framework/vue/guides/query-keys) and
  [updates from mutation responses](https://tanstack.com/query/latest/docs/framework/vue/guides/updates-from-mutation-responses).
