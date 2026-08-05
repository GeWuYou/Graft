# Saved Query Views Tracking

## Topic

Saved Query Views

## Scope

Private saved query views for Application Management, Audit Log Explorer, Access Log Explorer, App Log Explorer,
Announcement Management, and Scheduled Task Management.

## Repository Truth

- `AGENTS.md`
- `server/modules/saved-view/**`
- `web/src/shared/components/query-list/**`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Generic saved-view storage with owner-validated OpenAPI routes and shared query-list controls.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: direct-specialized-skill
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/saved-query-views
closeout:
  archive: false
  lessons_review: true
```

## Current Recovery Point

- The private saved-query-view rollout is complete for Application Management, Audit Log Explorer, Access Log Explorer,
  App Log Explorer, Announcement Management, and Scheduled Task Management.
- No database migration is required; the existing `saved_views` table remains authoritative.

## Task Checklist

- [x] Add saved-view APIs and validation to Announcement Management and Scheduled Task Management.
- [x] Add shared query-list controls and migrate Application Management.
- [x] Migrate audit, access-log, and app-log consumers.
- [x] Validate the cross-boundary feature slice.

## Acceptance Conditions

- Every supported page can save, apply, update, and delete its own private query view.
- Applying a view restores supported state and returns to page one.
- Saved-view controls appear in the query builder, not the table toolbar.

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
