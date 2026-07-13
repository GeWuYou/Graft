# Saved Query Views

## Current Status Summary

- Topic objective: Reuse private saved views across advanced query list pages and move the controls into the query builder.
- Current status: `validated-awaiting-archive-decision`
- Task class: `cross-boundary`
- Intake summary: Long-running feature with one shared web capability and four owner-validated list surfaces.
- Canonical authority:
  - `server/modules/saved-view/**`
  - owner routes and OpenAPI contracts for project, audit, access-log, and app-log
  - `web/src/shared/components/query-list/**`
- Completed: shared controls, all four consumer integrations, owner routes, OpenAPI generation, and cross-boundary validation.
- Pending: archive or extend the topic only when a new saved-view consumer is approved.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: saved-view storage is generic; each list owner retains route authorization and query-state validation.

## Owned Scope

- `server/modules/saved-view/**`, relevant owner routes, and `openapi/**`
- `web/src/shared/components/query-list/**` and the project, audit, access-log, and app-log list surfaces
- topic-local tracking and trace materials

Out of scope:

- shared or role-visible saved views
- new database tables, migrations, or browser-local persistence as the source of truth

## Locked Decisions

1. Saved views are private to the authenticated user and persist filters, sorters/time ranges, page size, and visible columns, never the current page.
2. The saved-view module remains storage-only; every list route retains its own permission and payload validation boundary.

## Current Recovery Point

- The initial rollout is complete and validated.
- Next step: archive this topic or create a bounded extension batch for a newly approved consumer.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full Work Contract in the tracking file, not here.

## Validation Targets

```bash
git diff --check
graft validate backend
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/saved-query-views/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
