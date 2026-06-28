# Compose Project Management Trace

## 2026-06-28 Phase 0 authority and recovery persistence

- 建立 Compose Project 设计 authority：`ai-plan/design/Compose项目管理设计.md`。
- 建立 active topic：`ai-plan/public/compose-project-management/README.md`。
- 建立 tracking：`ai-plan/public/compose-project-management/todos/compose-project-management-tracking.md`。
- 建立 trace：`ai-plan/public/compose-project-management/traces/compose-project-management-trace.md`。
- 建立 `$graft-multi-agent-loop` 启动提示：`ai-plan/public/compose-project-management/startup-prompt.md`。
- 将 active topic 注册进 `ai-plan/public/README.md`。

## 2026-06-28 Locked decisions

- `Project` 是 Compose Project 的管理与聚合层，不是新的 Runtime。
- `Container` 继续是 Runtime Authority。
- 当前仓库只有 Compose labels 识别能力，没有 Compose Project registry、解析、生命周期执行或项目级持久化。
- 推荐新增独立 `project` module，而不是让 `container` module 吞并项目注册。
- 推荐静态解析使用 `compose-go`，生命周期执行使用 `docker compose` CLI。
- 推荐 persistence 使用模块自有 `database/sql + migrations` 模式。
- 推荐为 `project` 新增最小的 container runtime read shared boundary，而不是直接 import container private service。
- Phase 1 Activity 继续复用现有 container logs/events，由前端 fan-out 聚合。
- Phase 1 Configuration 保持只读。

## 2026-06-28 Initial implementation direction

- Phase 1 优先级固定为：
  - Import Existing Project
  - Project Registry
  - Overview
  - Services
  - Configuration Read Only
  - Activity frontend aggregation
  - `up/down/restart`
  - Refresh
  - Unregister
  - Destroy guard
- Phase 2 再进入 managed create、editor、diff、validate、deploy。
- Phase 3 再进入 git/template/discovery/remote-host/backend aggregation。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-authority-and-topic-persistence"
  ],
  "pending_batches": [
    "phase-1-batch-1-project-contract-and-data-model",
    "phase-1-batch-2-server-project-module-import-and-refresh",
    "phase-1-batch-3-server-lifecycle-and-container-aggregation-boundary",
    "phase-1-batch-4-web-project-list-detail-and-readonly-configuration",
    "phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync",
    "phase-2-managed-create-editor-and-deploy",
    "phase-3-discovery-git-template-and-remote-host"
  ],
  "current_batch": "phase-0-design-authority-and-topic-persistence",
  "next_batch": "phase-1-batch-1-project-contract-and-data-model",
  "closeout_status": "phase-0-completed"
}
```
