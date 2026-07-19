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
- Full unmigrated inventory is recorded in `inventory.md`: 17 module API mirrors (193 values) plus all candidate non-HTTP module contract groups.
- Notification, project, runtime-target, and task now consume generated operationId runtime paths; their retained path contracts are web-private routes only.
- Current next step: migrate RBAC, user, audit, monitor, scheduler, and system-config API path consumers without adding a second HTTP authority.

## Task Checklist

- [x] inventory-and-openapi-runtime-path-projection
- [x] notification-project-runtime-target-task-migration
- [ ] rbac-user-audit-monitor-scheduler-system-config-migration
- [ ] security-announcement-app-log-access-log-migration
- [ ] drift-gate-expansion
- [ ] final-convergence-and-archive-readiness

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
    "notification-project-runtime-target-task-migration"
  ],
  "pending_batches": [
    "rbac-user-audit-monitor-scheduler-system-config-migration",
    "security-announcement-app-log-access-log-migration",
    "drift-gate-expansion",
    "final-convergence-and-archive-readiness"
  ],
  "current_batch": "notification-project-runtime-target-task-migration",
  "next_batch": "rbac-user-audit-monitor-scheduler-system-config-migration",
  "closeout_status": "completed"
}
```
