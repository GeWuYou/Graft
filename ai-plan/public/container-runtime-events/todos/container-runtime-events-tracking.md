# Container Runtime Events Tracking

## Topic

- Topic: `container-runtime-events`
- Status: `phase-1 implemented`
- Goal: 为容器模块建立 runtime-agnostic Runtime Events 设计、实施计划与分批恢复入口，随后按 `graft-multi-agent-loop`
  推进 bounded implementation batches。

## Scope

- 当前批次 owned scope：
  - `ai-plan/design/容器运行时事件能力设计.md`
  - `ai-plan/public/container-runtime-events/**`
  - `ai-plan/public/README.md`
- 主题级长期 owned scope：
  - `server/modules/container/**`
  - `openapi/**`
  - `server/internal/contract/openapi/**`
  - `web/src/modules/container/**`
- shared hotspot 仅在明确 bounded batch 下允许触达：
  - `server/internal/realtime/**`
  - `web/src/shared/realtime/**`
  - `web/src/shared/observability/**`

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器运行时事件能力设计.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/design/服务端API边界与兼容治理规范.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Current Recovery Point

- Phase 1 已完成，当前 live recovery point：
  - backend 已新增 canonical `RuntimeEvent` domain、source-agnostic `RuntimeEventManager`、bounded in-memory per-container history、`seq` replay record 和 history HTTP seed endpoint
  - Docker runtime 已作为首个 `RuntimeEventSource` adapter 接入，上游不泄漏 Docker-specific event model，severity 仍由 canonical mapping 决定
  - `ops.container.events` permission contract 与 `container.events:{id}` realtime topic 已落地
  - frontend 已新增 module-owned `events-manager`，detail page `Events` tab 已消费 `history seed + live append`
  - reconnect backfill 已落在 frontend manager，通过 `seq` merge / dedupe 避免重复和补洞
- 当前仍待后续批次完成：
  - Event Type / Severity filter
  - Search / Copy JSON / Jump to Logs
  - Timeline / timestamp UX hardening
  - provider extensibility hardening

## Task Checklist

- [x] Batch 0：建立 design authority
- [x] Batch 0：建立 active topic recovery
- [x] Batch 0：登记 phased rollout 与 compatibility analysis
- [x] Phase 1：backend `RuntimeEventSource` / `RuntimeEventManager` foundation
- [x] Phase 1：`ops.container.events` permission contract
- [x] Phase 1：Docker source adapter
- [x] Phase 1：bounded history + history API + realtime topic
- [x] Phase 1：reconnect backfill + `seq` merge strategy
- [x] Phase 1：frontend detail `Events` tab minimum slice
- [x] Phase 2：Event Type Filter
- [x] Phase 2：Severity Filter
- [x] Phase 2：Search / Copy JSON / Jump to Logs
- [x] Phase 2：Timeline Collapse / Relative-Absolute Time Toggle
- [ ] Phase 3：provider extensibility hardening
- [ ] Final：archive readiness and governance sync

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "batch-0-design-authority-and-topic-bootstrap",
    "phase-1-runtime-event-foundation",
    "phase-2-container-events-ux"
  ],
  "pending_batches": [
    "phase-3-provider-extensibility-and-hardening",
    "final-archive-readiness-and-governance-sync"
  ],
  "current_batch": "phase-2-container-events-ux",
  "next_batch": "phase-3-provider-extensibility-and-hardening",
  "closeout_status": "phase-2-complete"
}
```
