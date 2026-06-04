# Monitor Server Status Tracking

## Topic

- Topic: `monitor-server-status`
- Parent topic: `multi-worktree-governance` (archived)
- Status: `archived`
- Original branch: `feat/wt-monitor-server-status`
- Final closeout branch: `fix/module-runtime-ui-closeout`
- Scope: first minimal implementation slice under `server/modules/monitor/**` and `web/src/modules/monitor/**`

## Goal

- Keep the first capability focused on `server-status` only.
- Deliver the first minimal cross-boundary implementation slice without expanding beyond owned scope.
- Preserve the existing repository boundaries while wiring menu, route, permission, API, and page ownership inside the module boundary.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/模块与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Current Recovery Point

- This topic has been archived under `ai-plan/public/archive/monitor-server-status/**`.
- This topic was split out of `multi-worktree-governance` as a standalone active topic.
- The first implementation slice now exists in `server/modules/monitor/**` and `web/src/modules/monitor/**`.
- Backend module registration required one explicit shared-hotspot update in `server/internal/moduleregistry/generated.go`.
- The first minimal cross-boundary `monitor/server-status` slice now passes the backend and frontend completion entrypoints in this worktree.
- A server-only follow-up round now switches monitor module summaries from local dependency placeholders to runtime ordered module descriptors.
- That follow-up required explicit shared-hotspot updates in `server/internal/app/runtime.go`, `server/internal/module/module.go`, and `server/internal/module/runtime_metadata.go` to inject an observation-only runtime metadata snapshot into module context.
- The follow-up round passes direct package tests for `internal/module` and `modules/monitor`, and also passes `cd server && GIT_DIR=... GIT_WORK_TREE=... go run ./cmd/graft validate backend`.
- The current cross-boundary dashboard round expands the `server-status` payload inside `server/modules/monitor/**` only:
  - adds runtime memory / host summary
  - adds dependency detail plus latency samples
  - adds module dependency lists
  - adds an in-memory short retention trend window
  - adds summary counters for dashboard cards
- The corresponding `web/src/modules/monitor/**` page is now a dashboard-style monitor view with theme-aware cards and ECharts, and this slice updates `web/AGENTS.md` to freeze the relevant theme-token and color-mode rules.
- The current IA-alignment slice keeps the same backend payload and real page scope, but changes runtime navigation semantics to `服务器管理 / 服务器状态` through one explicit backend parent menu plus frontend tree assembly.
- The same slice explicitly keeps future monitor IA placeholders in `ai-plan` only and does not add runtime menus, routes, or permissions for them.
- The current monitor page now owns:
  - 5-second default auto refresh with manual / preset / custom interval options
  - page-hidden pause plus visible-immediate refresh
  - retry backoff after failed refreshes
  - friendly trend empty state when fewer than 2 samples exist
  - icon-assisted overview cards and grouped runtime sections
- The final Module Runtime UI closeout completed under `fix/module-runtime-ui-closeout`:
  - canonical UI name is `模块运行时` / `Module Runtime`
  - web route meta, breadcrumb, tab title, page title, empty copy, and drawer title now align with the canonical name
  - the table kept existing read-only fields and only adjusted column width allocation
  - the high-visibility table note was downgraded to auxiliary copy
  - the drawer now groups basic information, dependencies, migration, Schema, config, and diagnostics
  - no new menu, API, config, module write action, or dynamic plugin-platform behavior was added

## Shared Hotspots

- `ai-plan/public/README.md`
- `server/internal/app/runtime.go`
- `server/internal/module/**`
- `server/internal/moduleregistry/generated.go`
- `server/internal/moduleapi/**`
- `server/internal/contract/**`
- `web/src/router/**`
- `web/src/layouts/**`
- `web/src/locales/**`

## Ownership Boundary

- Standing ownership does not include the shared hotspots above.
- This slice used only the explicit shared-hotspot exception for `server/internal/moduleregistry/generated.go`.
- The follow-up round additionally used explicit shared-hotspot exceptions for `server/internal/app/runtime.go` and `server/internal/module/**` only to expose runtime metadata snapshots to modules.
- No web scope or other module scope expansion was required.

## Archived Risks

- Server version currently uses the explicit fallback value `dev`; there is still no stronger canonical runtime version source in the current repository surface.
- The dependency snapshot is intentionally shallow and based on existing runtime resources only; deeper health semantics would require a new scoped slice.
- The trend window is process-local and intentionally ephemeral; it does not survive restart and should not be presented as historical observability.
- Future monitor or module-runtime depth work should open a new bounded topic instead of reopening this archived line.

## Final Validation

- `cd web && bun run test:run src/modules/monitor/pages/modules/index.test.ts`
- `cd web && bun run test:run src/utils/route/bootstrap.test.ts src/modules/monitor/pages/modules/index.test.ts`
- `cd web && bun run format:check`
- `cd web && bun run typecheck`
- `cd web && bun run stylelint "src/modules/monitor/pages/modules/index.vue"`
- `cd web && bun run check`

## Immediate Next Step

- None. The topic is archived.
