# TanStack Adoption Follow-ups Tracking

## Topic

TanStack Adoption Follow-ups

## Work Contract

```yaml
version: 1
kind: architecture-evolution
scope: long-running
authority_summary: Query owns cached server state through one shared client while each web module owns API inputs, keys, invalidation, and local UI state.
requires:
  design: true
  topic: true
  roadmap: false
  adr: false
execution:
  engine: topic-completion-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - ai-plan/design/architecture/前端架构设计.md
    - ai-plan/public/tanstack-adoption-followups
closeout:
  archive: false
  lessons_review: true
```

## Current Recovery Point

- P0 baseline is committed as `5acd17d8`.
- P1 `standard-crud-query-migration` migrated the user-management list to a module-owned Query key and direct cache
  updates after list-affecting mutations. Filters, pagination, selection, form drafts, and lazy role-catalog behavior
  remain page-local UI state.
- The topic reopened on 2026-07-16 after an evidence audit found that the notification list still maintained a
  page-local server snapshot. `notification-list-query-migration` is complete: its normalized module key owns the
  paginated snapshot, single-read responses update known rows directly, and bulk/delete operations invalidate list
  variants rather than maintaining a page-local copy.

## Final Non-Query Decision

- `non-query-go-no-go` is complete. Do not add `@tanstack/vue-table`, `@tanstack/vue-virtual`,
  `@tanstack/vue-form`, or `@tanstack/vue-router` in this topic.
- Evidence: none of those packages is present in `web/package.json` or `web/bun.lock`; the application already uses
  TDesign tables/forms, Vue Router, and its existing LogViewer realtime/virtualized implementation. This review found
  no concrete measured performance or maintenance deficiency that would justify a parallel authority.
- Future Table, Virtual, or Form proposal: record a reproducible baseline, an affected interaction and data shape,
  target acceptance metrics (for example p95 interaction latency, dropped-frame rate, retained selection/focus, or
  maintenance reduction), and a rollback that removes the package and restores the current TDesign or LogViewer path.
- Future Router proposal: first revise the frontend architecture authority; then prove a navigation or route-registry
  deficiency in Vue Router, define route/navigation acceptance metrics, and retain a reversible migration path.
- References for a future evidence-backed proposal: [TanStack Table overview](https://tanstack.com/table/latest/docs/overview),
  [TanStack Virtual overview](https://tanstack.com/virtual/latest/docs/introduction),
  [TanStack Form Vue guide](https://tanstack.com/form/latest/docs/framework/vue/quick-start), and
  [TanStack Router Vue guide](https://tanstack.com/router/latest/docs/framework/vue/quick-start).

## Acceptance Conditions

- New server-data pages assess Query before adding manual request state.
- Every migrated query uses module API wrappers, normalized keys, mutation invalidation, and logout-safe cache behavior.
- No duplicate Pinia/page-local server snapshot remains after a migrated batch.
- Any non-Query TanStack adoption has written evidence, acceptance metrics, and an explicit rollback path.

## Archive-Readiness Check

- Passed on 2026-07-16: Query batches use module API wrappers, normalized module keys, direct cache updates or
  invalidation, and the shared logout-safe QueryClient lifecycle.
- Passed on 2026-07-16: migrated pages retain no duplicate Pinia or page-local server snapshot; filters, selection,
  drafts, streams, and other UI-local concerns remain outside Query cache.
- Passed on 2026-07-16: non-Query tools have an explicit no-go decision. Any future adoption is gated by written
  evidence, measurable acceptance criteria, and rollback.
- The Work Contract sets `closeout.archive: false`; retain this topic in place as `archive-ready` and do not change
  `ai-plan/public/README.md` or move the topic directory.

## Loop Batch State

```json
{
  "completed_batches": [
    "standard-crud-query-migration",
    "resource-detail-query-migration",
    "non-query-go-no-go",
    "notification-list-query-migration"
  ],
  "pending_batches": [
    "rbac-query-migration",
    "system-config-query-migration",
    "remaining-query-no-go-review"
  ],
  "current_batch": null,
  "next_batch": "rbac-query-migration",
  "closeout_status": "active"
}
```
