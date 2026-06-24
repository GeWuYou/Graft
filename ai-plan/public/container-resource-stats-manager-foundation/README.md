# Container Resource Stats Manager Foundation

本 README 只承载 topic recovery、阶段边界和 archive-ready 判定，不是仓库规范正文。

稳定设计 authority 以 `ai-plan/design/容器资源状态与订阅治理设计.md` 为准。

## 当前状态摘要

- 当前主题目标是为 `Graft` 容器管理建立统一资源状态层，收敛 `metadata / stats / subscription` 的 authority，并为 `List / Detail / Dashboard` 共享消费准备基础设施。
- 当前状态：Phase 0 设计与 recovery topic 已建立，尚未进入实现批次。
- 任务分类为 `cross-boundary`，涉及 container backend authority、OpenAPI 契约和 container frontend module state architecture。
- Canonical design：`ai-plan/design/容器资源状态与订阅治理设计.md`。
- 推荐执行技能：`$graft-multi-agent-loop`，loop mode 默认 `topic-completion-loop`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/容器资源状态与订阅治理设计.md` + `server/modules/container/**` + `web/src/modules/container/**`

## Owned Scope

允许修改：

- `ai-plan/design/容器资源状态与订阅治理设计.md`
- `ai-plan/public/container-resource-stats-manager-foundation/**`
- `ai-plan/public/README.md`
- `openapi/**`
- `server/modules/container/**`
- `server/internal/contract/openapi/container/**`
- `web/src/modules/container/**`
- 如 Dashboard 接入容器资源共享消费，允许触达 `web/src/modules/dashboard/**`

禁止误触：

- 不得把容器模块专属 stats state 直接升级为平台级 shared authority，除非先修订设计文档。
- 不得删除当前 `collector + cache + canonical topic` 后端主链，除非有新的上游 authority 设计并在同一主题内完成验证。
- 不得通过页面局部 patch 继续增加新的 `resource` 写入点。
- 不得为兼容旧页面而并行维护第二套长期 resource authority。

## Phase Plan

- Phase 0：设计锚定、审计结论落盘、topic 持久化。
- Phase 1：资源所有权分离，补齐 `ContainerResourceSummary.collected_at` 与 authority 注释。
- Phase 2：前端 `ContainerStatsManager` 与 `Metadata Store / Stats Store`。
- Phase 3：统一订阅管理与引用计数生命周期。
- Phase 4：Dashboard 共享资源层。
- Phase 5：可选 history store / ring buffer / trend metrics。

## Current Recovery Point

- 已完成 Arcane 与 Graft 资源数据流对照审计，结论已收敛到仓库级设计文档。
- 已确认：
  - Arcane 的列表页存在集中式 stats manager，但 Detail 与 Dashboard 不共享统一资源状态层。
  - Graft 后端已具备 `statsCollector -> resourceStatsCache -> container.stats:{id}` 主链。
  - Graft 前端当前仍缺少统一 stats state authority，资源状态分散在 list HTTP、detail HTTP 和 detail realtime 局部合并逻辑中。
- 当前下一批推荐从 Phase 1 开始，不直接跳到前端 manager 实现。

## Validation Targets

Phase 0 当前无需伪造运行时验证。

进入实现后默认验证：

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
git diff --check
```
