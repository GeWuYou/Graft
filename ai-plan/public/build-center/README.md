# Docker Build Center

## Current Status Summary

- Topic objective: establish the Build domain authority and deliver the first Docker image build slice.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running new capability requiring design, roadmap, and a recoverable topic.
- Canonical authority:
  - `ai-plan/design/architecture/docker-build-center.md`
  - `server/modules/build/**` and `web/src/modules/build/**`
- Completed so far: topic bootstrap, Phase 0 contracts, Build module registration, and the Docker execution foundation with persisted frozen snapshots and artifact settlement.
- Next bounded batch: Build read API and the web Build Jobs workflow.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `build-center`
- authority summary: Build owns jobs, artifacts, and frozen snapshots; Task owns execution state/logs/cancel/retry; Container owns Docker execution; Project owns request-time authorization and workspace authority.

## Owned Scope

- `ai-plan/design/architecture/docker-build-center.md`
- `server/modules/build/**`
- `server/internal/moduleapi/build*.go`
- `web/src/modules/build/**`
- `openapi/**` Build contract inputs and generated projections

Out of scope:

- changing Application, Runtime Target, or Compose deployment lifecycle semantics
- registry push, Git sources, buildx, multi-platform, secrets, SBOM, signatures, or automatic deployment
- unrelated self-update changes already present in the worktree

## Locked Decisions

1. Canonical navigation is `Build > Build Jobs` at `/build/jobs`.
2. Task Runtime remains the only execution status, log, retry, cancel, and realtime authority.
3. Container exposes Docker build capability; Build owns job and artifact business records.

## Phase Plan

- Phase 0: authority, module contracts, and task transaction/query adapters.
- Phase 1: local Dockerfile build, persistence, API, and web workflow.
- Phase 2+: history projections and advanced sources/executors.

## Current Recovery Point

- Docker execution foundation is committed in `e6d0f5c4` after backend, web, OpenAPI, migration, and diff validation.
- Risk: generated registries and OpenAPI projections remain derived outputs; web workflow must consume Build-owned read contracts rather than Task or Docker internals.
- Next step: implement the minimal Build read API and standalone Build Jobs web workflow.

## Work Intake

- This topic was created through `Work Intake`.
- Full Work Contract is in the tracking file.

## Pending Batch Direction

- Add only the Build read API contracts required for the Build Jobs workflow.
- Build the standalone list/create/detail workflow under `web/src/modules/build/**` using those canonical contracts.

## Validation Targets

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
just openapi-check
python3 scripts/validate_sql_migrations.py
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-center/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
