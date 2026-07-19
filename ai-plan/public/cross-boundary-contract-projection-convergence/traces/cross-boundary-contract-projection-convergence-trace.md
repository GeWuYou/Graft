# Cross-Boundary Contract Projection Convergence Trace

## 2026-07-19 Work Intake And Inventory Start

- Created a new active topic because the archived platform/container pilot is complete and cannot own the broader convergence work.
- Reaffirmed the authority split: OpenAPI owns HTTP contracts; existing Go server contracts own non-HTTP values; generated web artifacts are derived consumers.
- Started the complete unmigrated inventory with operationId-to-runtime-path projection as the first implementation batch.
- Added scripts/openapi-runtime-paths.mjs, which reads the deterministic bundled OpenAPI artifact and emits the operationId-indexed web runtime path artifact.
- Wired generation and freshness checks through just generate, just openapi-check, PR validation, and pre-push without changing the existing Go descriptor authority for non-HTTP values.
- Validation passed: git diff --check, the ai-plan, AI governance, and shared-asset structure guards, just generate, just openapi-check, the backend completion entrypoint, and the web completion entrypoint.

## Locked Decisions

- Web UI route, component, storage, query, and view-model contracts remain web-private and are not projection inputs.
- Generated values cannot grant permissions or capability access; server bootstrap and runtime responses remain the effective authority.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "inventory-and-openapi-runtime-path-projection"
  ],
  "pending_batches": [
    "notification-project-runtime-target-task-migration",
    "rbac-user-audit-monitor-scheduler-system-config-migration",
    "security-announcement-app-log-access-log-migration",
    "drift-gate-expansion",
    "final-convergence-and-archive-readiness"
  ],
  "current_batch": "inventory-and-openapi-runtime-path-projection",
  "next_batch": "notification-project-runtime-target-task-migration",
  "closeout_status": "completed"
}
```
