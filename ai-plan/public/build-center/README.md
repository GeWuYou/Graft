# Docker Build Center

## Current Status Summary

- Topic objective: establish the Build domain authority and deliver the first Docker image build slice.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running new capability requiring design, roadmap, and a recoverable topic.
- Canonical authority:
  - `ai-plan/design/architecture/docker-build-center.md`
  - `server/modules/build/**` and `web/src/modules/build/**`
- Completed so far: topic bootstrap and architecture decision.
- Not started yet: end-to-end persistence, task execution, and UI workflow.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: none
- authority summary: Build owns build jobs and artifacts; Task owns execution state; Container owns Docker execution.

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

- Topic bootstrap is complete; workers are implementing disjoint Phase 0 slices.
- Risk: generated registries and OpenAPI projections must remain derived outputs.
- Next step: accept worker closeouts, integrate authority repairs, and run focused validation.

## Work Intake

- This topic was created through `Work Intake`.
- Full Work Contract is in the tracking file.

## Pending Batch Direction

- Accept server Phase 0 contract/module work.
- Accept web Phase 0 registration/contract work.
- Follow with one integration batch for generated registration and validation.

## Validation Targets

```bash
git diff --check
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-center/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
