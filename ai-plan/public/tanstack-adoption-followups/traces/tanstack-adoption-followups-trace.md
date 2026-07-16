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
