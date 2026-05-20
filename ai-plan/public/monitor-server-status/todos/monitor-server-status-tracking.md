# Monitor Server Status Tracking

## Topic

- Topic: `monitor-server-status`
- Parent topic: `multi-worktree-governance` (archived)
- Branch: `feat/wt-monitor-server-status`
- Worktree: `/mnt/f/gewuyou/Project/Go/Graft-WorkTree/Graft-wt-monitor-server-status`
- Scope: first minimal implementation slice under `server/plugins/monitor/**` and `web/src/modules/monitor/**`

## Goal

- Keep the first capability focused on `server-status` only.
- Deliver the first minimal cross-boundary implementation slice without expanding beyond owned scope.
- Preserve the existing repository boundaries while wiring menu, route, permission, API, and page ownership inside the plugin/module.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Current Recovery Point

- This topic was split out of `multi-worktree-governance` as a standalone active topic.
- The first implementation slice now exists in `server/plugins/monitor/**` and `web/src/modules/monitor/**`.
- Backend plugin registration required one explicit shared-hotspot update in `server/internal/pluginregistry/generated.go`.
- The first minimal cross-boundary `monitor/server-status` slice now passes the backend and frontend completion entrypoints in this worktree.

## Shared Hotspots

- `ai-plan/public/README.md`
- `server/internal/pluginregistry/generated.go`
- `server/internal/pluginapi/**`
- `server/internal/contract/**`
- `web/src/router/**`
- `web/src/layouts/**`
- `web/src/locales/**`

## Ownership Boundary

- Standing ownership does not include the shared hotspots above.
- This slice used only the explicit shared-hotspot exception for `server/internal/pluginregistry/generated.go`.
- No additional shared-hotspot expansion was required.

## Active Risks

- Server version currently uses the explicit fallback value `dev`; there is still no stronger canonical runtime version source in the current repository surface.
- The dependency snapshot is intentionally shallow and based on existing runtime resources only; deeper health semantics would require a new scoped slice.
- Backend validation in this WSL worktree still depends on the explicit `GIT_DIR` and `GIT_WORK_TREE` override because plain `git` resolution remains misconfigured here.

## Immediate Next Step

- No additional implementation is required for the first minimal slice unless a follow-up round is requested.
- If a follow-up round is needed, keep it inside `server-status` depth improvements rather than broadening to new monitor capabilities.
