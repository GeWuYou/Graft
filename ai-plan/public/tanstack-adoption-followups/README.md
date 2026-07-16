# TanStack Adoption Follow-ups

## Current Status Summary

- Topic objective: complete only evidence-backed medium-risk TanStack Query migrations after the P0 rollout.
- Current status: `active` (reopened after evidence audit identified additional bounded list snapshots).
- Task class: `web`.
- Intake summary: long-running frontend architecture evolution with shared guidance and bounded module batches.
- Canonical authority:
  - `ai-plan/design/architecture/前端架构设计.md`
  - `web/AGENTS.md`
  - module `api/**`, module query keys, and `web/src/shared/query/**`

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `web`
- recovery source: `parent topic`
- authority summary: Query owns cached server state; each module owns its API input, query keys, mutation invalidation, and UI-local state.

## Owned Scope

- P1 web modules selected by the tracking batches.
- Query architecture guidance and topic-local recovery materials.

Out of scope:

- TanStack Router migration.
- Replacing TDesign tables/forms, existing LogViewer virtualization, Monaco, xterm, or WebSocket streams without new evidence.
- Server, OpenAPI, route/menu, or permission semantic changes unless authority discovery proves they are required.

## Current Recovery Point

- P0 QueryClient, announcement, monitor, access-log, app-log, and audit migrations landed in commit `5acd17d8`.
- P1 standard CRUD and resource-detail migrations landed in commits `344f9b15` and `7b38485a`.
- The final `non-query-go-no-go` batch remains valid: Table, Virtual, Form, and Router are deliberately unadopted.
- The reopened `notification-list-query-migration` and `rbac-query-migration` batches are complete. RBAC role lists,
  role-editor permission catalogs, and filtered permission lists are module-owned Query state; filters, pagination,
  drawers, drafts, selections, and detail sessions remain local. The system-config collection is now a module-owned
  Query snapshot; group search and selection, editor drafts, visibility, and mutation flags remain local. The only
  pending batch is `remaining-query-no-go-review`.

## Non-Query Decision

- Do not add TanStack Table, Virtual, Form, or Router for framework symmetry. The current authorities are TDesign
  Table/Form, the existing LogViewer realtime/virtualized path, and Vue Router; no measured performance or
  maintenance deficiency was found in this topic.
- Reconsider Table, Virtual, or Form only with a reproduced deficiency, a before/after acceptance metric that names
  the affected interaction, and a rollback that removes the new dependency and restores the existing authority.
- Reconsider Router only after the frontend architecture authority changes. Any proposal must show why Vue Router can
  no longer own route registration and navigation semantics, define route/navigation acceptance metrics, and retain a
  reversible migration path.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full Work Contract in the tracking file, not here.

## Loop Entry

- Preferred entry: `ai-plan/public/tanstack-adoption-followups/startup-prompt.md`.
- Preferred execution mode: `$graft-multi-agent-loop` with one bounded worker batch at a time.
