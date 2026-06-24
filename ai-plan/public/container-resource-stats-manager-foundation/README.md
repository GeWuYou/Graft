# Container Resource Stats Manager Foundation

本 README 只承载 topic recovery、阶段边界和 archive-ready 判定，不是仓库规范正文。

稳定设计 authority 以 `ai-plan/design/容器资源状态与订阅治理设计.md` 为准。

## 当前状态摘要

- 当前主题目标是为 `Graft` 容器管理建立统一资源状态层，收敛 `metadata / stats / subscription` 的 authority，并为 `List / Detail / Dashboard` 共享消费准备基础设施。
- 当前状态：`archive-ready`。
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
- 已完成 Phase 1 authority repair 目标：
  - `ContainerResourceSummary.collected_at` 回到 canonical contract
  - HTTP `resource` 明确为 seed snapshot / latest-known projection
  - collector / cache / canonical topic 主链保持不变
- 已完成 Phase 2 frontend foundation：
  - `web/src/modules/container/shared/stats-manager.ts` 建立 module-owned metadata/stats foundation
  - list/detail 已通过 selector 读取统一 stats authority
  - detail 不再用页面局部 merge/patch 长期保留 realtime resource
- 已完成 Phase 3 subscription manager unification：
  - `ContainerStatsManager` 收口 acquire / release / ref-count / idle-grace cleanup
  - list/detail 共享同一份 `container.stats:{id}` 订阅生命周期
  - list 加载失败只清理 list metadata，不再全局 reset module-owned stats authority
- 已完成 Phase 4 dashboard shared resource consumption：
  - dashboard 通过 container module contract facade 读取容器资源概览
  - dashboard 复用同一份 canonical stats authority 与 subscription lifecycle
  - container stats manager 增加 collection-key 隔离，避免 dashboard metadata projection 覆盖 list projection
- 已完成 Phase 5 optional history store：
  - container module 内新增短时 history ring buffer
  - detail resources 区可读取独立 history state，latest/history 显式分离
  - 未引入新的 server/OpenAPI authority 或 dashboard authority
- 当前补充修复：
  - container resource stats 默认采样周期调整为 1 秒，并通过 module system config 保持可覆盖
  - list/detail/dashboard 对 CPU / 内存 realtime 变化增加 module-owned visual emphasis，不改变 authority
- archive-ready 判定：
  - 所有计划 Phase 已完成并分别按批次提交
  - authority 主链保持为 `statsCollector -> resourceStatsCache -> container.stats:{id}`
  - container module 继续拥有 latest stats 与 optional history 的长期 authority
  - 无剩余 in-scope batch 需要继续推进

## Validation Targets

Phase 0 当前无需伪造运行时验证。

进入实现后默认验证：

```bash
cd server && go run ./cmd/graft validate backend
cd web && bun run check
git diff --check
```
