# Cross-Boundary Contract Projection Convergence

## Current Status Summary

- Topic objective: complete the authority-preserving migration from server-owned cross-boundary contracts and OpenAPI HTTP contracts to generated web artifacts.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: a new long-running convergence topic continues the archived platform and container projection pilot without reopening its completed history.
- Canonical authority:
  - `openapi/**` for HTTP paths, operations, wire schemas, and public wire enums
  - `server/internal/contract/**`, `server/modules/*/contract/**`, descriptors, and `server/internal/moduleapi/**` for non-HTTP cross-boundary values
- Completed so far: platform and container non-HTTP contract projection, freshness generation, and CI integration.
- Not started yet: operationId-to-runtime-path projection and the remaining module migrations.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic
- authority summary: OpenAPI remains the HTTP authority; server-owned Go contracts remain the non-HTTP authority; web consumes generated artifacts and keeps only private UI contracts.

## Owned Scope

- `ai-plan/public/cross-boundary-contract-projection-convergence/**`
- `ai-plan/public/README.md`
- `openapi/**` and deterministic OpenAPI-derived runtime path artifacts
- `server/internal/contract/**`, `server/modules/*/contract/**`, descriptors, `server/internal/moduleapi/**`, and associated projection/drift automation
- `web/src/contracts/generated/**` and module consumers required to remove server-owned mirrors

Out of scope:

- web-owned copies of server API paths, permissions, message keys, runtime values, or public wire enums
- changes to web-private UI routes, component contracts, storage keys, or query/view-model state unless required to remove a server mirror

## Locked Decisions

1. OpenAPI `operationId` is the identifier for generated web runtime API path lookup; path strings remain derived from canonical OpenAPI source.
2. API `code` and `messageKey` remain open strings with server fallback; generated permission values never grant runtime authority.
3. Generated descriptors reference existing Go constants for non-HTTP values and must not duplicate their literals.

## Phase Plan

- Batch 0: inventory, topic bootstrap, and operationId path projection foundation.
- Batch 1: migrate notification, project, runtime-target, and task API path consumers and server-owned values.
- Batch 2: migrate RBAC, user, audit, monitor, scheduler, system-config, security, announcement, app-log, and access-log.
- Batch 3: extend drift gates for hand-written mirrors, missing descriptors, visibility leaks, duplicate owners, and deprecated references.
- Batch 4: final convergence inventory, generated-artifact freshness, and archive-readiness review.

## Current Recovery Point

- The archived `cross-boundary-contract-projection` topic is historical evidence for the completed pilot only.
- Current batch: `notification-project-runtime-target-task-migration` completed.
- Next step: migrate the `rbac-user-audit-monitor-scheduler-system-config-migration` API path consumers to the generated operationId runtime-path artifact.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in `todos/cross-boundary-contract-projection-convergence-tracking.md`.

## Pending Batch Direction

- Complete the full unmigrated inventory before accepting each module migration batch.
- Keep module migration batches disjoint and validate both sides for every shared contract change.

## Validation Targets

```bash
git diff --check
just openapi-check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/cross-boundary-contract-projection-convergence/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
