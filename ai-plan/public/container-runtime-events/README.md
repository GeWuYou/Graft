# Container Runtime Events

## 当前状态摘要

- 当前主题目标是为 `Graft` 容器模块建立 runtime-agnostic 的 `Runtime Events` 能力设计与实施入口。
- 当前状态：设计 authority 已建立，active topic recovery 已建立，尚未开始 server / web 产品代码实现。
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
- 当前 container module 已具备两类实时模式：
  - Boot-started stats collector
  - topic-observer-driven lazy logs streamer
- 当前 frontend 已具备可复用 primitive：
  - websocket reconnect client
  - stream viewport state resolver
  - generic ring buffer implementation
  - module-owned `stats-manager` 模式
- 当前主题的核心 architecture 决议已固定：
  - `RuntimeEvent` 不包含 `runtime`
  - 不包含 `category`
  - 不包含 `message`
  - `severity` 由 canonical mapping 决定
  - backend pipeline 为 `RuntimeEventSource -> RuntimeEventManager -> History -> Topic`
  - phase 1 采用 `history seed + live stream + reconnect backfill by seq merge`
  - phase 1 推荐直接新增 `ops.container.events`，不以 `ops.container.detail` 作为默认长期权限
  - frontend realtime topic authority 继续落在 `web/src/modules/container/contract/realtime.ts`

## Phase Plan

- Batch 0：设计 authority、topic recovery、loop bootstrap。已完成。
- Phase 1：backend runtime event foundation + detail Events tab minimum slice。未开始。
- Phase 2：container events UX hardening。未开始。
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
