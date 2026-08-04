# Build History

## Current Status Summary

- Topic objective: deliver the Phase 2 Build history capabilities that are justified by proven query needs.
- Current status: `active`
- Task class: `cross-boundary`
- Canonical authority: `ai-plan/design/architecture/docker-build-center.md`, `ai-plan/roadmap/build-center.md`, `server/modules/build/**`, and `web/src/modules/build/**`.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: none
- authority summary: Build owns job/artifact history; Task owns execution state and logs; Container owns Docker runtime facts.

## Owned Scope

- `server/modules/build/**`
- `web/src/modules/build/**`
- Build-owned OpenAPI contracts and generated projections
- `ai-plan/public/build-history/**`

## Pending Batch Direction

- The implemented Build-owned filter and pagination slice is awaiting the controller-owned archive-readiness check; do not start Task, Container, Project, saved-view, retention, or durable-projection work without a newly justified batch.
