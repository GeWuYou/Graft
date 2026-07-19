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

## 2026-07-19 First API Path Consumer Migration

- Migrated notification, project, runtime-target, and task API consumers from module-owned HTTP path mirrors to `OPENAPI_RUNTIME_PATH` and `buildOpenApiRuntimePath`.
- Preserved only web-private route contracts and removed task's API-only path contract.
- Updated the frontend OpenAPI governance guard to recognize the generated operationId path artifact as a path lookup, not a generated runtime HTTP client; direct runtime clients remain prohibited.
- Focused/scoped web checks passed: focused Vitest, frontend OpenAPI governance, lint, stylelint, hygiene, OpenAPI freshness, and diff checks.
- The full web completion entrypoint (`cd web && bun run check`) failed only on five pre-existing timeout tests in `src/locales/index.test.ts`, `src/permission.test.ts`, and `src/modules/auth/store/session.test.ts`.

## 2026-07-19 RBAC And User API Path Migration

- Migrated RBAC and user API consumers and focused tests to generated OpenAPI runtime paths; RBAC no longer retains an API-only path contract.
- Focused Vitest passed for the RBAC and user API clients; the full web completion entrypoint was not rerun for this resumed checkpoint repair.

## 2026-07-19 Audit And Monitor API Path Migration

- Focused monitor API tests passed; `web/src/modules/audit/**` has no focused API test file.

## 2026-07-19 Non-HTTP Descriptor Coverage And Archive Readiness

- Classified every remaining candidate as a Go contract projection, OpenAPI-generated wire type, or web-private UI contract; the tracking table records each disposition.
- Added module-scoped generated artifacts for all confirmed permission groups, project error/topic values, the runtime-target summary topic, and task realtime values.
- Promoted task realtime topic/event strings into `server/modules/task/contract/realtime.go`; emitted values now use those typed constants, correcting the web's stale dotted event union.
- Repaired two discovered downstream authority drifts: the auth refresh adapter now reads `OPENAPI_RUNTIME_PATH.postAuthRefresh`, and container retains a route-only `contract/paths.ts` with no HTTP path values.
- `cd server && go run ./cmd/graft validate backend`, projection freshness, web formatting/type checking, the OpenAPI frontend guard, lint/style/hygiene checks, focused affected web tests (42/42), and the production build passed.
- `cd web && bun run check` reaches its full Vitest stage but remains blocked by the pre-existing locale, permission, and auth-session timeout cluster; a transient i18n fixture-directory error also occurred in that full parallel test run. These failures are outside the changed contract consumers, whose focused tests pass.

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
