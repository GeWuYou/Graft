# Observability Development Governance Trace

## 2026-05-29 Phase A completed logging development standard

- Re-ran startup preflight from root `AGENTS.md`.
- Read:
  - `.ai/environment/tools.ai.yaml`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/public/README.md`
  - `ai-plan/design/项目设计.md`
  - `ai-plan/design/插件与依赖注入设计.md`
  - `ai-plan/design/前端架构设计.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
  - archived `request-correlation-access-logging`
  - archived `logging-unification-rollout`
  - archived `plugin-audit-correlation-governance`
- Reconfirmed canonical authority chain:
  - `server/internal/logger/**` owns backend app/error logging baseline
  - `server/internal/httpx/**` owns request correlation, structured access logging, and HTTP security-event bridge
  - `server/internal/audit/**` plus `server/plugins/audit/**` own audit persistence and metadata normalization
- Produced `ai-plan/design/日志治理开发规范.md`.
- Marked Phase A done in the topic tracking docs.
- Explicitly deferred:
  - code inventory for Phase B
  - runtime code changes
  - frontend audit-console integration work

## 2026-05-29 Phase B started inventory-only compliance scan

- Renamed branch to `feat/observability-development-governance` to match the active topic.
- Scanned:
  - `server/internal/**`
  - `server/plugins/**`
  - `server/cmd/**`
- Confirmed `server/pkg/**` does not exist in this repository state.
- Inventory conclusions so far:
  - real fix-now items are concentrated in manual request-id helper propagation, non-canonical log field names, one remaining runtime panic path, and one global logger direct-use site
  - generated `ent/**` `panic(err)` and `log.Println` defaults exist, but most hits are generated artifacts rather than current bounded production authority
  - `fmt.Println` in key-generation CLIs remains intentional stdout, not an app-log bypass

## 2026-05-29 Phase B completed bounded compliance rollout

- Repaired the highest in-scope authority surfaces first instead of adding new compatibility layers:
  - `server/internal/i18n/service.go`
  - `server/internal/app/runtime.go`
  - `server/plugins/user/route_user_handlers.go`
  - `server/plugins/user/plugin.go`
  - `server/plugins/rbac/route_write_handlers.go`
  - `server/plugins/rbac/write_service.go`
  - `server/plugins/auth/route_errors.go`
  - `server/plugins/monitor/plugin.go`
  - `server/plugins/user/storeent/runtime.go`
- Kept compatibility-only consumers unchanged where the external contract still requires them:
  - audit query/read aliases such as `request_id`
  - generated OpenAPI and generated Ent artifacts outside the bounded production path
- Validation passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./plugins/user/... ./plugins/rbac/... ./plugins/audit/...`

## 2026-05-29 Phase C completed audit-console governance UX rollout

- Kept authority-first scope bounded to frontend consumption surfaces:
  - `web/src/modules/audit/**`
  - `web/src/modules/rbac/**`
  - `web/src/modules/user/**`
  - `web/src/shared/correlation.ts`
- Added route-query-driven audit filter hydration and in-place URL sync so governance views are shareable without inventing a second frontend-only audit model.
- Added `requestId` / `traceId` visibility, copy actions, and bounded query mapping to the canonical backend `request_id` filter.
- Added `source` and `reason` presentation so operators can distinguish security events from broader audit events without inventing fake backend fields.
- Added RBAC role-list, RBAC permission-list, and user-list navigation into `/audit/logs` with bounded related-audit queries.
- Added correlation-aware success/error hints on user and RBAC write operations using the latest canonical frontend request correlation snapshot.
- Preserved intentionally:
  - existing backend audit query contract
  - current shell/page structure
  - no full audit-console redesign
- Removed the runtime use of static overview risk-watch copy because it presented fake governance signals; P2 analytics remain future work.
- Validation passed:
  - `cd web && bun run check`

## 2026-05-29 Topic reached archive-ready closeout

- Phase A, Phase B, and Phase C all completed in the required order.
- Current branch remains `feat/observability-development-governance`.
- This topic now serves as archive-ready evidence for future observability or audit UX follow-up.
