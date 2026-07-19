# Cross-Boundary Contract Projection Convergence Tracking

## Topic

Cross-Boundary Contract Projection Convergence

## Scope

Complete the remaining projection of server-owned cross-boundary values and OpenAPI-owned runtime API paths into generated web artifacts, then block new hand-written authority mirrors.

## Repository Truth

- `AGENTS.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/public/archive/cross-boundary-contract-projection/README.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: OpenAPI owns HTTP paths, operations, wire schemas, and public wire enums; server Go contracts own non-HTTP values; web consumes generated artifacts only.
requires:
  design: false
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-loop
bootstrap:
  targets:
    - ai-plan/public/cross-boundary-contract-projection-convergence
    - ai-plan/public/cross-boundary-contract-projection-convergence/roadmap
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- The prior platform/container pilot is archived and remains evidence only.
- All web runtime HTTP paths now consume the OpenAPI operationId projection. The auth refresh adapter uses `postAuthRefresh`; container retains only web-private route paths.
- All identified web-visible non-HTTP server values now consume module-scoped generated descriptors. The task realtime topic and event values were promoted from inline runtime strings into `server/modules/task/contract` before projection.
- The final archive-readiness checks are recorded in the trace. No pending authority mirror remains in the scoped inventory.

## Non-HTTP Authority Disposition

| Candidate group | Canonical authority | Disposition |
| --- | --- | --- |
| access-log and app-log permissions | `server/internal/httpx`, `server/internal/logger` | Generated module permission artifacts |
| announcement, audit, monitor, notification, RBAC, scheduler, security, system-config, and user permissions | respective `server/modules/*/contract/permission.go` owners | Generated module permission artifacts |
| project inspection error and realtime topics | `server/modules/project/contract` | Generated project error-code and realtime-topic artifact |
| runtime-target summary topic | `server/modules/runtime-target/contract` | Generated runtime-target realtime-topic artifact |
| task topic and realtime events | `server/modules/task/contract/realtime.go` | Generated task topic/event artifact after authority promotion |
| OpenAPI wire enums | `openapi/**` and generated schema | Continue consuming generated schema types; no descriptor duplication |
| routes, navigation, display maps, filters, and view models | `web` module contracts | Explicitly retained as web-private contracts |

## Task Checklist

- [x] inventory-and-openapi-runtime-path-projection
- [x] notification-project-runtime-target-task-migration
- [x] rbac-user-api-path-migration
- [x] audit-monitor-api-path-migration
- [x] scheduled-task-system-config-api-path-migration
- [x] security-announcement-api-path-migration
- [x] app-log-access-log-api-path-migration
- [x] drift-gate-expansion
- [x] final-convergence-and-archive-readiness
- [x] non-http-descriptor-coverage-and-archive-readiness

## Acceptance Conditions

- Every HTTP runtime API path used by web is derived from OpenAPI operation identity, with no module `contract/paths.ts` server API mirror remaining.
- Every web-visible non-HTTP server value references an existing Go constant through a descriptor; descriptors contain no copied literal authority.
- Runtime permissions, capabilities, menu visibility, and unknown API code/message values continue to be server-decided.
- CI rejects freshness drift, new hand-written mirrors, missing required descriptors, visibility leaks, duplicate ownership, and new deprecated references.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "inventory-and-openapi-runtime-path-projection",
    "notification-project-runtime-target-task-migration",
    "rbac-user-api-path-migration",
    "audit-monitor-api-path-migration",
    "scheduled-task-system-config-api-path-migration",
    "security-announcement-api-path-migration",
    "app-log-access-log-api-path-migration",
    "drift-gate-expansion",
    "final-convergence-and-archive-readiness",
    "auth-dashboard-api-path-migration",
    "container-api-path-migration"
  ],
  "pending_batches": [],
  "current_batch": "non-http-descriptor-coverage-and-archive-readiness",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
