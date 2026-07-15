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
- No P1 migration batch has started.

## Pending Batches

1. `standard-crud-query-migration`
   - Evaluate and migrate one bounded group from user, RBAC, notification, system-config, scheduled-task, or runtime-target.
2. `resource-detail-query-migration`
   - Evaluate project/container list and static detail resources; preserve realtime stream authority and editor drafts.
3. `non-query-go-no-go`
   - Record evidence for Table, Virtual, and Form only when a concrete performance or maintenance problem exists; Router is rejected unless architecture authority changes.

## Acceptance Conditions

- New server-data pages assess Query before adding manual request state.
- Every migrated query uses module API wrappers, normalized keys, mutation invalidation, and logout-safe cache behavior.
- No duplicate Pinia/page-local server snapshot remains after a migrated batch.
- Any non-Query TanStack adoption has written evidence, acceptance metrics, and an explicit rollback path.

## Loop Batch State

```json
{
  "completed_batches": ["p0-query-foundation-and-high-yield-consumers"],
  "pending_batches": [
    "standard-crud-query-migration",
    "resource-detail-query-migration",
    "non-query-go-no-go"
  ],
  "current_batch": null,
  "next_batch": "standard-crud-query-migration",
  "closeout_status": "active-planned"
}
```
