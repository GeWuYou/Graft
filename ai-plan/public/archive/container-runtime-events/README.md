# Container Runtime Events

## 当前状态摘要

- 当前主题目标是为 `Graft` 容器模块建立 runtime-agnostic 的 `Runtime Events` 能力设计、实施入口与治理收口。
- 状态：`archived`。
- 任务分类为 `cross-boundary`。
- Canonical design：
  - `ai-plan/design/容器运行时事件能力设计.md`
  - `ai-plan/design/容器管理设计.md`
  - `ai-plan/design/容器资源状态与订阅治理设计.md`
- 已完成 Phase 1 runtime event foundation、Phase 2 container events UX、Phase 3 provider extensibility and hardening，以及 Final archive readiness and governance sync。
- 本主题已从 active topic index 移入 `ai-plan/public/archive/container-runtime-events/`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：
  - Phase 1 live at `561a9021`
  - Phase 2 live at `1ce24b39`
  - Phase 3 live at `076a3576`
  - `server/modules/container/**` owns Runtime Events domain and extension seams
  - shared realtime owns transport only

## Historical Owned Scope

允许修改：

- `ai-plan/design/容器运行时事件能力设计.md`
- `ai-plan/public/container-runtime-events/**`
- `ai-plan/public/archive/container-runtime-events/**`
- `ai-plan/public/README.md`
- `server/modules/container/**`
- `openapi/**`
- `server/internal/contract/openapi/**`
- `web/src/modules/container/**`

仅在明确 bounded implementation batch 下允许最小范围触达：

- `server/internal/realtime/**`
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

## Phase Plan

- Batch 0：设计 authority、topic recovery、loop bootstrap。
- Phase 1：backend runtime event foundation + detail Events tab minimum slice。
- Phase 2：container events UX hardening。
- Phase 3：provider extensibility and hardening。
- Final：archive readiness and governance sync。

## Current Recovery Point

- 分支为 `feat/container-realtime-logs`。
- archived topic 已落到 `ai-plan/public/archive/container-runtime-events/`。
- Phase 1 已完成：
  - backend canonical model：`RuntimeEvent` / `RuntimeEventRecord` / `RuntimeEventsHistory`
  - backend source-agnostic manager：`RuntimeEventSource -> RuntimeEventManager -> History -> Topic`
  - `ops.container.events` permission、`container.events:{id}` topic、`GET /api/ops/containers/{id}/events` history endpoint 已落地
  - frontend container module 已新增 module-owned `events-manager`，detail 页已消费 `history seed + live stream`
- Phase 2 已完成：
  - Events tab 已补齐 event type / severity filter、search、copy JSON、jump to logs、timeline collapse、relative / absolute time toggle
- Phase 3 已完成：
  - manager 已收敛到显式 source registration seam
  - duplicate suppression、history TTL read eviction、source diagnostics 与 direct tests 已落地
- Final 已完成：
  - 已按 design/topic authority 对实现做最终审计
  - 未发现需要继续回到 `server/modules/container/**`、`openapi/**`、`web/src/modules/container/**` 的真实 bounded drift
  - active topic index 已清理，主题进入 archived 终态

## Archive-Readiness Check

- design、topic recovery、implementation 已对齐：
  - `RuntimeEvent` 未引入 `runtime` / `category` / provider-authored `message`
  - `severity` 继续由 canonical mapping 决定
  - realtime authority 仍保持 `container.events:{id}` + unified transport
- Phase 1 / 2 / 3 目标结果均已存在于 live implementation 与 direct tests / UI tests 中。
- 当前 owned scope 内不存在明确且未完成的后续 bounded batch。
- 本主题最终仅新增 archive/governance sync 文档变更；实现侧未发现必须修补的 drift。
- 结论：本主题已 `archive-ready`，并已完成归档迁移。

## Validation Targets

```bash
git diff --check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```
