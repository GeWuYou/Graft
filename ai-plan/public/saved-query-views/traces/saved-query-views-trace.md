# Saved Query Views Trace

## 2026-07-13 Work Intake and Batch 1

- Created a long-running cross-boundary topic for the shared saved-query-view capability.
- Reused the existing `saved-view` module and rejected a generic public route because list owners must retain permission and payload validation authority.
- Locked the first rollout to project, audit, access-log, and app-log; views are private and exclude current pagination position.

## 2026-07-13 Rollout Completion

- Added consumer-owned CRUD routes and OpenAPI contracts for audit, access-log, and app-log; the shared storage module remains route-free.
- Added shared query-list controls and presentation helpers, then migrated Application Management and all three log explorers.
- Saved state includes filters, sorter/time state, page size, and visible columns; applying a view always starts at page one.
- Kept saved-view controls in the query builder and removed the obsolete Application Management table-toolbar control.
- Validated with `go run ./cmd/graft validate backend`, frontend lint and hygiene checks, `bun run test:run` (1304 tests), and `bun run build`.
- Browser evidence confirms the audit page places saved-view selection and actions in the query bar while the table toolbar retains only refresh and column settings.

## Loop Batch State

```json
{
  "completed_batches": ["saved-query-views-foundation", "saved-query-views-consumer-rollout", "saved-query-views-announcement-scheduler"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "validated-awaiting-archive-decision"
}
```

## 2026-08-05 Announcement and Scheduler Extension

- Added owner-scoped saved-view CRUD routes and OpenAPI contracts for Announcement Management and Scheduled Task Management.
- Migrated both pages to the shared advanced query builder and saved-view control; saved state restores filters, page size,
  visible columns, and starts from page one.
- Removed Announcement Management's local column-preference persistence in favor of saved-view presentation state.
- Validated with `go run ./cmd/graft validate backend`, focused module tests, focused page tests, and frontend hygiene,
  formatting, type, and contract checks.
