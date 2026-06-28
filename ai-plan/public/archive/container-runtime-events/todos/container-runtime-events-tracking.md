# Container Runtime Events Tracking

## Topic

- Topic: `container-runtime-events`
- Status: `archived`
- Goal: 为容器模块建立 runtime-agnostic Runtime Events 设计、实施计划、分批恢复入口与最终治理收口。

## Scope

- 本次 final closeout owned scope：
  - `ai-plan/public/container-runtime-events/**`
  - `ai-plan/public/archive/container-runtime-events/**`
  - `ai-plan/public/README.md`
- 主题级长期 owned scope：
  - `server/modules/container/**`
  - `openapi/**`
  - `server/internal/contract/openapi/**`
  - `web/src/modules/container/**`

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器运行时事件能力设计.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`

## Current Recovery Point

- Phase 1 live at `561a9021`：runtime event foundation 已完成。
- Phase 2 live at `1ce24b39`：container events UX 已完成。
- Phase 3 live at `076a3576`：provider extensibility and hardening 已完成。
- Final closeout 结论：
  - 已完成 archive-readiness audit
  - 未发现需要继续回到实现层的真实 drift
  - 主题已移入 archive，并从 active topic index 清理

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
- [x] Phase 3：provider extensibility hardening
- [x] Final：archive readiness and governance sync

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "batch-0-design-authority-and-topic-bootstrap",
    "phase-1-runtime-event-foundation",
    "phase-2-container-events-ux",
    "phase-3-provider-extensibility-and-hardening",
    "final-archive-readiness-and-governance-sync"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
