# Monitor Server Status Tracking

## Topic

- Topic: `monitor-server-status`
- Parent topic: `multi-worktree-governance` (archived)
- Branch: none yet
- Worktree: none yet
- Scope: design-only topic; future implementation is expected under `server/plugins/monitor/**` and `web/src/modules/monitor/**`

## Goal

- Freeze the recovery entry for the new standalone monitor topic.
- Keep the first capability focused on `server-status` only.
- Preserve the existing repository boundaries before any implementation slice starts.

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
- The repository root is back on `main`.
- This topic currently stops at design materials only; no `server/**` or `web/**` source code is modified.

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
- The owned scope for the next implementation slice should stay inside `server/plugins/monitor/**` and `web/src/modules/monitor/**`.

## Active Risks

- If future work pushes `monitor` business truth into shell-owned or shared-hotspot paths, the topic will stop being a clean standalone recovery entry.
- If the first implementation slice expands beyond `server-status`, the topic will lose the intended minimal MVP boundary.

## Immediate Next Step

- Keep this topic design-only until the corresponding implementation slice is explicitly started.
- When implementation begins, keep menu, route, permission, API, and locale ownership inside the plugin/module boundaries above.
