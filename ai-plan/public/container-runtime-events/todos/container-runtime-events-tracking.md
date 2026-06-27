# Container Runtime Events Tracking

## Topic

- Topic: `container-runtime-events`
- Status: `active recovery entry`
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

- 规划 authority 已落定，尚未开始产品代码实现。
- 当前可复用基础设施结论：
  - Logs、Stats、Dashboard summary 已使用统一 realtime topic pipeline。
  - Shell 使用 module-private websocket，不可作为 Runtime Events transport authority。
  - `LogRingBuffer<T>` 虽然命名偏日志，但实现已是 generic，可直接复用为 event replay buffer。
  - `stream-viewport-state` 可直接复用为 Events page visual state primitive。
- 当前 domain model 决议：
  - `RuntimeEvent` 只承载 event fact
  - `runtime` 属于 stream context，不属于 event fact
  - `severity` 由 canonical mapping 决定
  - 不维护 `category`
  - 不维护 `message`
- 当前 backend owner 决议：
  - `RuntimeEventSource` 是新的 runtime extension interface
  - `RuntimeEventManager` 不得直接知道 Docker
  - reconnect 后必须重新拉取 bounded history，并按 `seq` merge / dedupe
  - phase 1 默认新增 `ops.container.events`
  - web 不得新建平行 realtime topic contract owner

## Task Checklist

- [x] Batch 0：建立 design authority
- [x] Batch 0：建立 active topic recovery
- [x] Batch 0：登记 phased rollout 与 compatibility analysis
- [ ] Phase 1：backend `RuntimeEventSource` / `RuntimeEventManager` foundation
- [ ] Phase 1：`ops.container.events` permission contract
- [ ] Phase 1：Docker source adapter
- [ ] Phase 1：bounded history + history API + realtime topic
- [ ] Phase 1：reconnect backfill + `seq` merge strategy
- [ ] Phase 1：frontend detail `Events` tab minimum slice
- [ ] Phase 2：Event Type Filter
- [ ] Phase 2：Severity Filter
- [ ] Phase 2：Search / Copy JSON / Jump to Logs
- [ ] Phase 2：Timeline Collapse / Relative-Absolute Time Toggle
- [ ] Phase 3：provider extensibility hardening
- [ ] Final：archive readiness and governance sync

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "batch-0-design-authority-and-topic-bootstrap"
  ],
  "pending_batches": [
    "phase-1-runtime-event-foundation",
    "phase-2-container-events-ux",
    "phase-3-provider-extensibility-and-hardening",
    "final-archive-readiness-and-governance-sync"
  ],
  "current_batch": "batch-0-design-authority-and-topic-bootstrap",
  "next_batch": "phase-1-runtime-event-foundation",
  "closeout_status": "batch-0-complete"
}
```
