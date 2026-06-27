# Container Runtime Events Trace

## 2026-06-27 Batch 0 design authority and topic bootstrap

- 建立 design authority：
  - `ai-plan/design/容器运行时事件能力设计.md`
- 建立 active topic recovery：
  - `ai-plan/public/container-runtime-events/README.md`
  - `ai-plan/public/container-runtime-events/todos/container-runtime-events-tracking.md`
  - `ai-plan/public/container-runtime-events/traces/container-runtime-events-trace.md`
- 将 Runtime Events 主题登记到 `ai-plan/public/README.md`。
- 当前已固定以下 architecture 决议：
  - `RuntimeEvent` 不包含 `runtime`
  - `RuntimeEvent` 不包含 `category`
  - contract 不包含 `message`
  - `severity` 由 canonical mapping 决定
  - backend pipeline 采用 `RuntimeEventSource -> RuntimeEventManager -> Normalize -> History -> Topic`
  - phase 1 history strategy 为 bounded in-memory replay，而不是 live-only 或 persistent store
  - phase 1 reconnect 必须通过 history backfill + `seq` merge 补洞，而不是只依赖 ws transport reconnect
  - phase 1 推荐直接新增 `ops.container.events` permission contract
  - phase 1 detail 页签与 topic contract 必须复用 live owner：`raw` tab naming 与
    `web/src/modules/container/contract/realtime.ts`
- 当前已固定以下 compatibility 结论：
  - 可直接复用 unified realtime topic pipeline
  - 可直接复用 frontend websocket reconnect
  - 可直接复用 stream viewport state primitive
  - 可直接复用 generic ring buffer implementation
  - 不复用 shell websocket

## Recovery Notes

- 后续实现应继续从 container module authority 出发，而不是先在 realtime core 或 provider 层扩散 contract。
- 如果 phase 1 实现需要扩展 `server/internal/realtime/**` 或 `web/src/shared/observability/**`，必须保持为最小 primitive
  扩展，不得把 container event semantics 抬升为平台层真值。
