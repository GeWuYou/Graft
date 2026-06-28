# Container Runtime Events Trace

## 2026-06-27 Batch 0 design authority and topic bootstrap

- 建立 design authority：
  - `ai-plan/design/domains/container/容器运行时事件能力设计.md`
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

## Recovery Notes

- 后续实现应继续从 container module authority 出发，而不是先在 realtime core 或 provider 层扩散 contract。
- 如果 phase 1 实现需要扩展 `server/internal/realtime/**` 或 `web/src/shared/observability/**`，必须保持为最小 primitive
  扩展，不得把 container event semantics 抬升为平台层真值。

## 2026-06-27 Phase 1 runtime event foundation

- backend foundation：
  - 在 `server/modules/container/**` 新增 canonical `RuntimeEvent` domain model、`RuntimeEventSource` extension interface 和 source-agnostic `RuntimeEventManager`
  - manager 维护 bounded per-container in-memory history，并以 `seq` 作为 reconnect-safe merge / dedupe authority
  - 新增 `GET /api/ops/containers/{id}/events` history endpoint，history response 暴露 `seq`
  - 新增 `ops.container.events` permission contract 与 `container.events:{id}` realtime topic ownership
- Docker provider：
  - 当前 Docker runtime 已接入 `StreamRuntimeEvents`
  - provider 只上抬 canonical event type + attributes 输入，未把 Docker-specific event model、provider severity 或 provider message 暴露到上层
- frontend minimum slice：
  - 在 `web/src/modules/container/contract/realtime.ts` 扩展 container module-owned realtime contract
  - 新增 module-owned `web/src/modules/container/shared/events-manager.ts`
  - 在 detail page 新增 `Events` tab，并放在 `Logs` 前
  - Events tab 已消费 HTTP history seed + realtime topic append，reconnect 后通过 `seq` merge / dedupe 回补
- generated / required spill：
  - OpenAPI source、server openapi generated artifact、web generated schema 已同步
  - 为 RBAC permission catalog 补齐 `ops.container.events` 展示文案，形成最小 required spill
- validation：
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`

## 2026-06-27 Phase 2 container events UX

- authority / scope
  - 维持 `web/src/modules/container/**` 作为 Events UX owner；未新增 shared realtime path、compat DTO 或第二事件存储
  - 继续复用 phase 1 `events-manager` 的 history seed + live stream + `seq` merge pipeline
- implementation
  - 在 container detail `Events` tab 补齐 presentation-level `Event Type` / `Severity` filter 与 search
  - 新增事件时间 `Relative / Absolute` toggle、整表 `Timeline Collapse` 与单条 detail collapse
  - 新增 `Copy JSON` 与 `Jump to Logs` 操作；`Jump to Logs` 复用现有 logs owner，以事件 `occurred_at` 回填现有 `since` 查询
  - 维持 filter 为当前已加载窗口内的 presentation-level 处理，未扩为后端筛选参数
- validation
  - `git diff --check`
  - `cd web && bunx vitest run src/modules/container/pages/detail/index.test.ts`
  - `cd web && bun run check`

## 2026-06-27 Phase 3 provider extensibility and hardening

- backend provider seam hardening
  - `runtimeEventManager` 现已接收显式 `runtimeEventSourceRegistration` 列表，而不是在 manager 内部隐式依赖 `runtimeLoader -> RuntimeEventSource` 的单点 type assert
  - `service.startRuntimeEventManager()` 继续保持当前单 runtime 行为，但已把 source ownership 收敛到 registration seam，后续 Podman / containerd 仅需新增 registration
- ordering / dedupe / retention hardening
  - manager 现在会在当前 bounded history window 内按 canonical event id 抑制重复事件，避免重复 source 输入导致 `seq` 无意义递增
  - `History()` 读取路径会执行 TTL 驱逐，避免 removed / inactive container 仅在后续追加事件时才被清理
  - per-resource stream context 现在与 history 一起保存，避免未来多 source / runtime 扩展时把默认 runtime 假设泄漏到历史读取面
- diagnostics and tests
  - 新增 source diagnostics：`streamStarts`、`streamErrors`、`invalidDrops`、`duplicateDrops`、`lastError`、`lastEventAt`
  - 新增 direct tests 覆盖 duplicate suppression、TTL read eviction、source registration start idempotency 和 diagnostics 记录
- validation
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend`

## 2026-06-27 Final archive readiness and governance sync

- final audit：
  - 复核 design authority、topic recovery 与 live implementation，未发现需要继续回到
    `server/modules/container/**`、`openapi/**`、`web/src/modules/container/**` 的真实 bounded drift
  - 复核 Phase 1 / 2 / 3 目标结果均已在 live implementation 中存在：
    - canonical severity 仍由 container module mapping 决定
    - history HTTP seed 与 `container.events:{id}` realtime topic 仍共用 `seq` merge / dedupe 语义
    - detail `Events` tab 的 filter / search / copy JSON / jump to logs 仍在 module-owned UI 中
    - provider seam、TTL read eviction、duplicate suppression、source diagnostics 仍有 direct tests 覆盖
- governance sync：
  - 将 topic 从 active topic index 移出
  - 将 recovery 资料迁入 `ai-plan/public/archive/container-runtime-events/`
  - 当前主题终态改记为 `archive-ready` / `archived`
- final validation baseline：
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- terminal verdict：
  - 本主题已 `archive-ready`
  - 当前无剩余 pending batch
  - 后续如需继续扩展 Podman / containerd / Kubernetes provider，必须另开新 topic，而不是重新打开本归档主题

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
