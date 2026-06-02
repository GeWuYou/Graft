# Monitor Server Status Tracking

## Topic

- Topic: `monitor-server-status`
- Parent topic: `multi-worktree-governance` (archived)
- Branch: `feat/wt-monitor-server-status`
- Worktree: `/mnt/f/gewuyou/Project/Go/Graft-WorkTree/Graft-wt-monitor-server-status`
- Scope: first minimal implementation slice under `server/modules/monitor/**` and `web/src/modules/monitor/**`

## Goal

- Keep the first capability focused on `server-status` only.
- Deliver the first minimal cross-boundary implementation slice without expanding beyond owned scope.
- Preserve the existing repository boundaries while wiring menu, route, permission, API, and page ownership inside the plugin/module.

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

- This topic was split out of `multi-worktree-governance` as a standalone active topic.
- The first implementation slice now exists in `server/modules/monitor/**` and `web/src/modules/monitor/**`.
- Backend plugin registration required one explicit shared-hotspot update in `server/internal/pluginregistry/generated.go`.
- The first minimal cross-boundary `monitor/server-status` slice now passes the backend and frontend completion entrypoints in this worktree.
- A server-only follow-up round now switches monitor plugin summaries from local dependency placeholders to runtime ordered plugin descriptors.
- That follow-up required explicit shared-hotspot updates in `server/internal/app/runtime.go`, `server/internal/plugin/plugin.go`, and `server/internal/plugin/runtime_metadata.go` to inject an observation-only runtime metadata snapshot into plugin context.
- The follow-up round passes direct package tests for `internal/plugin` and `plugins/monitor`, and also passes `cd server && GIT_DIR=... GIT_WORK_TREE=... go run ./cmd/graft validate backend`.
- The current cross-boundary dashboard round expands the `server-status` payload inside `server/modules/monitor/**` only:
  - adds runtime memory / host summary
  - adds dependency detail plus latency samples
  - adds plugin dependency lists
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

## Shared Hotspots

- `ai-plan/public/README.md`
- `server/internal/app/runtime.go`
- `server/internal/plugin/**`
- `server/internal/pluginregistry/generated.go`
- `server/internal/pluginapi/**`
- `server/internal/contract/**`
- `web/src/router/**`
- `web/src/layouts/**`
- `web/src/locales/**`

## Ownership Boundary

- Standing ownership does not include the shared hotspots above.
- This slice used only the explicit shared-hotspot exception for `server/internal/pluginregistry/generated.go`.
- The follow-up round additionally used explicit shared-hotspot exceptions for `server/internal/app/runtime.go` and `server/internal/plugin/**` only to expose runtime metadata snapshots to plugins.
- No web scope or other plugin scope expansion was required.

## Active Risks

- Server version currently uses the explicit fallback value `dev`; there is still no stronger canonical runtime version source in the current repository surface.
- The dependency snapshot is intentionally shallow and based on existing runtime resources only; deeper health semantics would require a new scoped slice.
- The trend window is process-local and intentionally ephemeral; it does not survive restart and should not be presented as historical observability.
- Backend validation in this WSL worktree still depends on the explicit `GIT_DIR` and `GIT_WORK_TREE` override because plain `git` resolution remains misconfigured here.
- Frontend completion still depends on the repository bun run check entrypoint; if that command cannot run, the round must report the exact command gap rather than claiming full completion.

## Immediate Next Step

- No immediate implementation gap remains for the current IA-alignment slice after validation.
- If a later round is requested, keep it inside `server-status` depth improvements such as:
  - richer plugin runtime-status semantics
  - splitting `概览` / `性能趋势` into separate real pages only when data density justifies it
  - adding `健康检查日志` or `配置诊断` as new explicit monitor capabilities rather than placeholder contracts
