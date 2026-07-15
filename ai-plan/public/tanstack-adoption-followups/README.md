# TanStack Adoption Follow-ups

## Current Status Summary

- Topic objective: complete only evidence-backed medium-risk TanStack Query migrations after the P0 rollout.
- Current status: `active-planned`.
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
- Next batch: classify and migrate one P1 standard CRUD group; do not start a broad multi-module rewrite.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full Work Contract in the tracking file, not here.

## Loop Entry

- Preferred entry: `ai-plan/public/tanstack-adoption-followups/startup-prompt.md`.
- Preferred execution mode: `$graft-multi-agent-loop` with one bounded worker batch at a time.
