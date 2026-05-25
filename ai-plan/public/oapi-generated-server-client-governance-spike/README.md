# OAPI Generated Server/Client Governance Spike

## Summary

This topic started with the `monitor/server-status` pilot and now records the guarded generated-contract rollout slices
that followed.

- Topic: `oapi-generated-server-client-governance-spike`
- Task class: `cross-boundary`
- Branch: `feat/oapi-generated-server-client-governance-spike`
- Current recommendation: `archive_topic`

## Current Recovery State

- Auth guarded migration and boundary audit are complete inside this topic.
- Known completed commits:
  - `713a676`
    - Batch 1 migrated:
      - `POST /api/auth/refresh`
      - `POST /api/auth/logout`
- Batch 2 status:
  - completed and committed:
    - `GET /api/auth/sessions`
    - `POST /api/auth/sessions/revoke-all`
    - `POST /api/auth/sessions/revoke-others`
    - `POST /api/auth/sessions/{sessionID}/revoke`
  - commit:
    - `a28ea34`
  - generated/backend/frontend boundaries stay explicit:
    - `server/plugins/auth/**` still owns route registration, validation, service commands, and `httpx` envelopes
    - `web/src/modules/auth/api/auth.ts` still owns module adapters over `request.ts`
- Batch 3 status:
  - completed and committed:
    - `POST /api/auth/change-password`
    - `POST /api/auth/complete-required-password-change`
  - commit:
    - `38a287f`
  - generated/backend/frontend boundaries stay explicit:
    - `server/plugins/auth/**` still owns route registration, validation, service commands, and `httpx` envelopes
    - `web/src/modules/auth/api/auth.ts` still owns module adapters over `request.ts`
- Auth audit closeout:
  - committed:
    - `6fb286a`
      - fixed the single-session revoke path drift in `web/src/modules/auth/contract/paths.ts`
    - `5370cd8`
      - recorded the completed auth boundary audit closeout
  - no remaining auth migration or auth audit follow-up is pending in this topic
  - any future generated freshness-gate expansion should run as a separate docs/automation decision instead of
    reopening auth migration
- Final remaining interface migration:
  - current worktree completes:
    - `GET /healthz`
    - `GET /api/users/{id}/sessions`
    - `POST /api/users/{id}/sessions/revoke-all`
    - `POST /api/users/{id}/sessions/{sessionID}/revoke`
  - repository OpenAPI interface migration status is now:
    - `30 / 30` operations migrated to the guarded generated contract boundary
    - `remaining_interfaces = []`
  - topic recommendation remains:
    - `archive_topic`

## Scope

- Writable scope:
  - `ai-plan/public/oapi-generated-server-client-governance-spike/**`
  - `openapi/**`
  - `scripts/openapi_generated_freshness_check.py`
  - `server/internal/contract/openapi/monitor/**`
  - `server/internal/contract/openapi/rbac/**`
  - `server/internal/contract/openapi/user/**`
  - `server/internal/contract/openapi/auth/**`
  - `server/plugins/monitor/**`
  - `server/plugins/rbac/**`
  - `server/plugins/user/**`
  - `server/plugins/auth/**`
  - `web/src/modules/monitor/**`
  - `web/src/modules/rbac/**`
  - `web/src/modules/user/**`
  - `web/src/modules/auth/**`
- Read-only context:
  - `server/internal/httpx/**`
  - `web/src/utils/request.ts`

## Pilot Rules

- Keep `monitor/server-status` as the only operation in the pilot.
- Keep `httpx` as the backend envelope and localized error owner.
- Keep `request.ts` as the frontend transport/runtime owner.
- Keep generated route/path typing module-owned; do not leave a second handwritten full-path truth when a module
  contract constant can carry the template.
- Do not introduce a second global router or transport truth.

## Current Implementation Shape

- Backend:
  - generated monitor-only server bindings live in `server/internal/contract/openapi/monitor/generated/**`
  - monitor plugin still owns explicit route registration
  - generated layer constrains parameter binding and handler interface only
- Frontend:
  - monitor API uses operation-bound generated typing
  - pages still consume module API only
  - `request.ts` remains the only runtime transport adapter

## Validation Expectation

- `git diff --check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-monitor --mode check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-health --mode check`
- `cd web && bun run openapi:types:check`
- `cd web && bun run check`
- `cd server && go run ./cmd/graft validate backend`
- `scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci`


## Phase 4 Review

### Verdict

The `monitor/server-status` generated governance spike is a `partial success`.

- `success` would require a closed freshness/drift loop for both backend and frontend generated artifacts.
- `failed` would require boundary takeover or unacceptable glue complexity, which did not happen here.
- The actual result sits in the middle:
  - the boundary shape stayed explicit and locally governable
  - the frontend gained meaningful typed-governance value
  - the backend gained a narrower compile-time constraint than the frontend
  - freshness gating is still incomplete, so the spike is not yet a promotion-ready pattern

### Evidence

#### Backend generated server adapter

- The backend generated layer is intentionally narrow:
  - [`server/internal/contract/openapi/monitor/types.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/internal/contract/openapi/monitor/types.go:1)
  - [`server/internal/contract/openapi/monitor/zz_generated.types.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/internal/contract/openapi/monitor/zz_generated.types.go:1)
- It constrains:
  - the handler method shape `GetMonitorServerStatus(params)`
  - generated enum/query/header parameter values such as `trend_range`, `X-Graft-Locale`, and `X-Request-Id`
- It does not take over:
  - route registration
  - Gin middleware ownership
  - `httpx` response envelope ownership
  - localized error handling
  - plugin lifecycle wiring
- The plugin still owns explicit runtime integration:
  - [`server/plugins/monitor/plugin.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/plugins/monitor/plugin.go:401)
  - [`server/plugins/monitor/plugin.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/plugins/monitor/plugin.go:408)
  - [`server/plugins/monitor/plugin.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/plugins/monitor/plugin.go:425)
  - [`server/plugins/monitor/plugin.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/plugins/monitor/plugin.go:441)
- Phase 4 answer:
  - the backend handler boundary is constrained, but only partially
  - this is compile-time parameter/interface governance, not generated runtime routing governance

#### Backend boundary preservation

- Plugin registry and route ownership stayed in the plugin:
  - `ctx.Router.Group(monitorcontract.MonitorGroup)` remains the entrypoint
- Gin middleware stayed intact:
  - `httpx.RequestIDMiddleware()`
  - `httpx.RequirePermission(...)`
- Existing runtime envelope and error behavior stayed intact:
  - `httpx.AbortLocalizedError(...)`
  - `httpx.WriteSuccess(...)`
- Request-id and locale semantics stayed aligned with the spec rather than being replaced by a second runtime path:
  - [`openapi/paths/monitor.server-status.yaml`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/openapi/paths/monitor.server-status.yaml:1)
  - [`server/plugins/monitor/plugin.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/plugins/monitor/plugin.go:449)
- Root router takeover did not happen.

#### Frontend typed adapter

- The module API now binds directly to generated operation types while still going through `request.ts`:
  - [`web/src/modules/monitor/api/server-status.ts`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/modules/monitor/api/server-status.ts:1)
- The module response types are now thin aliases from generated schemas:
  - [`web/src/modules/monitor/types/server-status.ts`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/modules/monitor/types/server-status.ts:1)
- The runtime transport owner did not change:
  - [`web/src/utils/request.ts`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/utils/request.ts:1)
- Pages still consume module API and module-owned types instead of importing generated client/runtime code directly:
  - [`web/src/modules/monitor/pages/overview/index.vue`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/modules/monitor/pages/overview/index.vue:379)
  - [`web/src/modules/monitor/shared/server-status-snapshot.ts`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/modules/monitor/shared/server-status-snapshot.ts:4)
- There is explicit regression coverage for the `request.ts` transport truth:
  - [`web/src/modules/monitor/api/server-status.test.ts`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/web/src/modules/monitor/api/server-status.test.ts:15)
- Phase 4 answer:
  - this spike materially reduces handwritten monitor path/query/DTO drift
  - the frontend benefit is real and larger than the backend benefit

#### Glue code cost

- Backend glue cost is moderate:
  - generated package
  - one explicit param binder
  - one compile-time interface assertion
  - generated-file lint exclusions
- Frontend glue cost is low:
  - generated operation typing in one module API
  - thin schema aliases in one module type file
- Net judgment:
  - frontend glue cost is clearly lower than its governance gain
  - backend glue cost is acceptable for a pilot, but not yet strong enough to justify immediate rollout by itself

#### Lint exclusion and backend cache namespace patch

- Generated-file lint exclusions stayed minimal and local to the monitor generated file:
  - [`server/.golangci.yml`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/.golangci.yml:1)
  - [`server/.golangci.test.yml`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/.golangci.test.yml:1)
- The backend validate cache patch is also minimal:
  - [`server/internal/cli/validate.go`](/home/gewuyou/project/go/Graft-wt/feat/oapi-generated-server-client-governance-spike/server/internal/cli/validate.go:623)
- Phase 4 judgment:
  - the lint exclusion is small enough
  - the cache namespace patch is a reasonable governance patch for a worktree-heavy repository
  - it belongs to this topic because it preserves backend validation stability for the new generated-contract slice without redefining validation truth

### Costs And Gaps

- The backend generated adapter does not enforce response-envelope shape at the handler boundary because `httpx` still owns the real success/error envelope.
- The backend adapter still relies on handwritten Gin-to-generated parameter binding.
- The OpenAPI `3.1.x` warning from `oapi-codegen` remains a real rollout risk even though it did not block the pilot.
- The previous largest governance gap was freshness:
  - frontend already had `bun run openapi:types:check`
  - backend now has an equivalent blocking freshness gate for `server/internal/contract/openapi/monitor/zz_generated.types.go`

### Recommendation

Primary recommendation: `3. 在 freshness gate 保持稳定后再考虑下一批渐进迁移`

- Do not expand this pattern to another interface unless both freshness gates stay stable in normal validation.
- Keep the current monitor pilot in place as an accepted bounded experiment.
- Do not switch to generated runtime clients or strict/generated server runtime as part of the next slice.
- If rollout resumes after freshness gating lands, the next candidate should be:
  - another low-risk read-only interface with existing `httpx` envelope stability and low middleware novelty
  - `GET /api/permissions` is the best current candidate because it is read-only, has simpler query semantics than `monitor/server-status`, and avoids auth/session lifecycle coupling
- If rollout does not resume, archive this spike as:
  - accepted partial-success monitor pilot
  - no further expansion without a new dedicated design topic before `auth/user/rbac`

## Next-Session Startup Prompt

```text
使用 $graft-multi-agent-loop。

governance source: root AGENTS.md
task class: docs/automation
recovery source:
  - current repository state
  - ai-plan/public/oapi-generated-server-client-governance-spike/README.md
  - ai-plan/public/oapi-generated-server-client-governance-spike/traces/oapi-generated-server-client-governance-spike-trace.md
  - ai-plan/public/README.md
branch / worktree:
  - feat/oapi-generated-server-client-governance-spike
owned scope:
  - ai-plan/public/oapi-generated-server-client-governance-spike/**
  - ai-plan/public/README.md if topic status changes
  - docs/traces/todos only
  - validation/check tooling directly required for generated freshness gating only if a new topic explicitly broadens scope
forbidden scope:
  - 不修改 server 业务实现
  - 不修改 web 业务实现
  - 不修改 OpenAPI spec 语义
  - 不扩大 generated runtime 覆盖面
objective:
  - 为 monitor generated server/client pilot 设计并收口最小 generated freshness/check gate
  - 明确 backend generated Go artifact 与 frontend generated TS schema 的 blocking/non-blocking gate 位置
  - 判断该 gate 应该落在 docs/automation 还是单独的新治理 topic
validation:
  - git diff --check
  - git status --short
```

## Machine-Readable Closeout

```json
{
  "closeout_status": "partial_success",
  "continue": false,
  "next_prompt": "use_graft_multi_agent_loop_for_generated_freshness_gate_design",
  "stop_reason": "Phase 4 docs-only governance review completed. The monitor pilot kept repository boundaries intact and delivered frontend typed-governance value, but generated artifact freshness gating is still incomplete.",
  "validation": {
    "status": "docs_only",
    "commands": [
      "git diff --check",
      "git status --short"
    ],
    "not_run": [
      "cd server && go run ./cmd/graft validate backend",
      "cd web && bun run openapi:types:check",
      "cd web && bun run check",
      "scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci"
    ],
    "reason": "This review changes only governance docs. No server, web, OpenAPI, or generated artifact files were modified."
  },
  "recommendation": {
    "selected_option": 2,
    "label": "first_add_generated_freshness_gate",
    "pilot_verdict": "partial_success",
    "promote_now": false
  },
  "evidence": {
    "backend_generated_server_adapter": "partial_benefit",
    "frontend_request_ts_compatible_typed_adapter": "clear_benefit",
    "backend_validate_cache_namespace_patch": "reasonable_topic_local_governance_patch",
    "generated_artifact_freshness_gate_needed": true
  },
  "scope_expanded": false,
  "risk_level": "medium"
}
```


## Phase 5 Freshness Gate

### Verdict

The monitor pilot now has a minimal freshness gate on both sides.

- frontend freshness continues to use `bun run openapi:types:check`
- backend freshness is now checked by regenerating the monitor-only Go artifact to a temp file and diffing it against
  `server/internal/contract/openapi/monitor/zz_generated.types.go`
- `graft validate backend` now includes that backend freshness gate through the existing `openapi` stage instead of
  inventing a second backend validation entrypoint

### Review Answers

- `当前 web openapi:types:check 是否已经能证明 frontend generated schema freshness？`
  - Yes. It regenerates the schema into a temp file, formats it, and compares it to the tracked generated file.
- `backend monitor generated contract 是否已有等价 freshness check？`
  - It does now. `python3 scripts/openapi_generated_freshness_check.py --target backend-monitor --mode check` is the
    backend equivalent for the monitor-only generated Go artifact.
- `如果没有，应该新增 server 侧 check，还是统一新增 scripts/openapi generated freshness check？`
  - A repository-owned script is the smallest fit. It keeps generator logic near repo governance and is reused by the
    backend validate flow without changing generated artifacts.
- `这个 freshness gate 应该挂在 cd server && go run ./cmd/graft validate backend 里，还是作为独立命令被 CI/主流程调用？`
  - Both, with one source of truth. The explicit script remains runnable on its own, and `graft validate backend`
    calls it from the existing `openapi` stage.
- `generated file lint exclusion 是否仍然只作用于 generated artifacts？`
  - Yes. This slice does not broaden lint exclusions beyond the existing generated-file scope.
- `是否能在不新增依赖的前提下完成 freshness check？`
  - Yes. The backend check uses existing `python3` and `oapi-codegen`.
- `如果需要新增依赖，是否必须停止并升级为单独决策？`
  - No new dependency was needed. If a future broader gate requires one, that should be a separate decision.

### Progressive Migration Preconditions

- keep the backend and frontend freshness gates green in local and CI validation
- keep generated files pure outputs; only specs or generator flags may change them
- keep runtime ownership explicit: no generated Gin router takeover and no `request.ts` replacement
- keep rollout limited to low-risk read-only interfaces until another slice proves the cost remains proportionate

### Recommended Next Batches

1. `GET /api/permissions`
2. `GET /api/users`
3. `GET /api/roles`

Do not start auth/session lifecycle routes or write-heavy endpoints before those read-only slices prove stable.

## Phase 6 Guarded Progressive Migration Batch 2

### Verdict

The guarded RBAC read migration now covers the next low-risk batch after `GET /api/permissions`.

- backend expands the narrow RBAC generated contract package to the read-only batch:
  - `getPermissions`
  - `getRoles`
  - `getRolePermissions`
- route registration, middleware ownership, and `httpx` envelope ownership remain in the RBAC plugin
- frontend keeps consuming the module API through `request.ts`, and the RBAC read helpers now bind to their generated
  OpenAPI response envelope data types
- freshness gating stays explicit under the broader `backend-rbac-read` target for the RBAC generated artifact

### Shape

- Backend:
  - `server/internal/contract/openapi/rbac/**` owns the generated RBAC read handler-shape/header contract only
  - `server/plugins/rbac/**` still owns explicit route registration for:
    - `/api/permissions`
    - `/api/roles`
    - `/api/roles/{id}/permissions`
  - `server/plugins/rbac/**` still owns auth middleware, `httpx` success/error envelopes, and read-service invocation
- Frontend:
  - `web/src/modules/rbac/api/rbac.ts` still exports module-owned helpers
  - `request.ts` remains the only runtime transport owner
  - the RBAC read helpers now bind to:
    - `paths['/api/permissions']['get']`
    - `paths['/api/roles']['get']`
    - `paths['/api/roles/{id}/permissions']['get']`

### Validation Expectation For This Batch

- `git diff --check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-monitor --mode check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-rbac-read --mode check`
- `cd web && bun run openapi:types:check`
- `cd web && bun run test:run -- --runInBand src/modules/rbac/api/rbac.test.ts`
- `cd server && go test ./internal/contract/openapi/rbac ./plugins/rbac`

## Phase 6 Guarded Progressive Migration Batch 4

### Verdict

The guarded RBAC management artifact now covers `POST /api/roles/{id}/permissions/assign` as the next write-only batch.

- backend keeps the generated layer narrow:
  - generated contract now constrains the assign-permissions path/header/request-body shape
  - RBAC plugin still owns explicit route registration, permission middleware, service invocation, and `httpx`
    envelopes
- frontend keeps module API ownership:
  - `web/src/modules/rbac/api/rbac.ts` still calls `request.ts`
  - `assignRolePermissions()` now binds to the generated OpenAPI request-body type for the same operation
- backend freshness gating remains unified under `backend-rbac-management`; no second generated RBAC artifact was
  introduced

### Validation Expectation For This Batch

- `git diff --check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-monitor --mode check`
- `python3 scripts/openapi_generated_freshness_check.py --target backend-rbac-management --mode check`
- `cd web && bun run openapi:types:check`
- `cd web && bun run test:run src/modules/rbac/api/rbac.test.ts`
- `cd server && go test ./internal/contract/openapi/rbac ./plugins/rbac`
