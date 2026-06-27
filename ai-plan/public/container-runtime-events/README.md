# Container Runtime Events

## 当前状态摘要

- 当前主题目标是为 `Graft` 容器模块建立 runtime-agnostic 的 `Runtime Events` 能力设计与实施入口。
- 当前状态：Phase 2 container events UX 已完成实现并通过前端完成态校验；下一批次进入 Phase 3 provider extensibility and hardening。
- 任务分类为 `cross-boundary`。
- 当前设计 authority：
  - `ai-plan/design/容器运行时事件能力设计.md`
  - `ai-plan/design/容器管理设计.md`
  - `ai-plan/design/容器资源状态与订阅治理设计.md`

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：
  - backend domain owner 是 `server/modules/container/**`
  - realtime transport owner 是 `server/internal/realtime/**`
  - frontend feature owner 是 `web/src/modules/container/**`
  - shared stream primitive owner 是 `web/src/shared/realtime/**` 与 `web/src/shared/observability/**`

## Owned Scope

允许修改：

- `ai-plan/design/容器运行时事件能力设计.md`
- `ai-plan/public/container-runtime-events/**`
- `ai-plan/public/README.md`
- 后续实现批次允许按 bounded slice 修改：
  - `server/modules/container/**`
  - `server/internal/contract/openapi/**`
  - `openapi/**`
  - `web/src/modules/container/**`
  - 若仅为 shared primitive 扩展，允许最小范围修改：
    - `web/src/shared/realtime/**`
    - `web/src/shared/observability/**`

禁止误触：

- 不得把 Docker-specific event model 抬升为 canonical contract
- 不得新增第二套 websocket gateway
- 不得把 shell private websocket 重用为 Runtime Events transport
- 不得在 `RuntimeEvent` 中加入 `runtime`
- 不得同时维护 `category` 与 `event_type`
- 不得把 provider-authored `severity` 直接暴露给上层
- 不得在 contract 中加入 provider-authored `message`
- 不得新增持久化事件表作为 phase 1 前置条件

## Current Recovery Point

- 当前仓库已具备可复用的 realtime transport：
  - `POST /realtime/subscriptions`
  - `GET /ws`
  - `TopicIssuerRegistry`
  - `Hub`
- Phase 1 已交付以下 live slice：
  - backend canonical model：`RuntimeEvent` / `RuntimeEventRecord` / `RuntimeEventsHistory`
  - backend source-agnostic manager：`RuntimeEventSource -> RuntimeEventManager -> History -> Topic`
  - Docker provider 已作为首个 `RuntimeEventSource` adapter 接入，且 severity 继续由 canonical mapping 决定
  - `ops.container.events` permission、`container.events:{id}` topic、`GET /api/ops/containers/{id}/events` history endpoint 已落地
  - frontend container module 已新增 module-owned `events-manager`，并在 detail 页中以 `Events` tab 方式消费 `history seed + live stream`
  - reconnect backfill 语义已按 `seq` merge / dedupe 落在 manager 内
- 当前主题的核心 architecture 决议继续保持：
  - `RuntimeEvent` 不包含 `runtime`
  - 不包含 `category`
  - 不包含 `message`
  - `severity` 由 canonical mapping 决定
  - frontend realtime topic authority 继续落在 `web/src/modules/container/contract/realtime.ts`

## Phase Plan

- Batch 0：设计 authority、topic recovery、loop bootstrap。已完成。
- Phase 1：backend runtime event foundation + detail Events tab minimum slice。已完成。
- Phase 2：container events UX hardening。已完成。
- Phase 3：provider extensibility and hardening。未开始。
- Final：archive readiness and governance sync。未开始。

## Validation Targets

规划文档批次：

```bash
git diff --check
```

后续若进入 server / web 实现批次：

```bash
cd server && graft validate backend
cd web && bun run check
```

本轮实际已运行：

```bash
git diff --check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```
