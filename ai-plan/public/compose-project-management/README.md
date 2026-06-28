# Compose Project Management

## 当前状态摘要

- 当前主题目标是在 `Graft` 增加 Docker Compose Project 管理能力。
- 当前状态：`active`。
- 任务分类为 `cross-boundary`，涉及 `ai-plan/design`、future OpenAPI、future `server/modules/project/**`、future `web/src/modules/project/**`，并与现有 `container` runtime authority 协作。
- Canonical design：`ai-plan/design/Compose项目管理设计.md`。
- 当前已完成 Phase 0：设计 authority、topic recovery 与 loop 启动提示持久化。
- 当前尚未开始业务实现。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/Compose项目管理设计.md` + `ai-plan/design/容器管理设计.md` + future OpenAPI source + future `server/modules/project/**` + future `web/src/modules/project/**`

## Owned Scope

当前 topic 允许修改：

- `ai-plan/design/Compose项目管理设计.md`
- `ai-plan/public/compose-project-management/**`
- `ai-plan/public/README.md`
- future `openapi/**` project contract source
- future `server/modules/project/**`
- future `server/internal/moduleapi/**` 中项目实现所需的最小稳定共享边界
- future `web/src/modules/project/**`
- 必要的 generated OpenAPI artifacts 与模块装配接入文件

禁止误触：

- 不得让 `Project` 持久化容器运行时状态、日志、事件或 Stats。
- 不得为 `Project` 新增第二套 Container Detail。
- 不得让 `Project` 成为第二套 Runtime。
- Phase 1 不得新增 project logs/events backend API 或 realtime topic。
- Phase 1 不得把 Overview 做成 dashboard。
- Phase 1 不得把配置编辑、Diff、Deploy、Validate UI 偷渡进来。

## Locked Architecture Decisions

1. `Project` 是 Compose Project 的管理与聚合层，不是新的 Runtime。
2. `Container` 是 Runtime Authority。
3. `Project` 只拥有 registry、ownership、compose files、lifecycle、services aggregation、activity aggregation entry。
4. `Container` 继续拥有 runtime state、stats、logs、events、shell、inspect、networks、mounts。
5. Phase 1 的 Activity 继续复用现有 container logs/events，由前端做 fan-out 聚合。
6. Phase 1 的 Configuration 保持只读，且 API 拆为 metadata/list、preview、single-file content 三类。
7. `Canonical Project Name` 与 `Display Name` 必须分离。
8. `Snapshot` 只保存最近一次成功解析结果：`normalized compose + config hash + refresh metadata`。
9. `Unregister` 是安全默认；`Destroy` 必须受 ownership proof 保护。
10. Phase 1 只做 `local host`，但数据模型必须预留 future remote host。

## Phase Plan

- Phase 0：设计 authority、topic recovery、loop startup prompt。已完成。
- Phase 1：Import Existing Project、Project Registry、Overview、Services、Configuration Read Only、Activity frontend aggregation、`up/down/restart`、Refresh、Unregister、Destroy guard。
- Phase 2：Managed Project Create、Compose Editor、Env Editor、Diff、Validate、Deploy。
- Phase 3：Git Project、Templates、Directory Scan、Auto Discovery、Remote Host、Project Activity backend aggregation。

## Current Recovery Point

- 设计 authority 已创建：`ai-plan/design/Compose项目管理设计.md`
- active topic 已创建：`ai-plan/public/compose-project-management/`
- 当前共识：
  - 推荐新增独立 `project` module，而不是扩展 `container` module 承担项目注册。
  - 推荐静态解析使用 `compose-go`，生命周期执行使用 `docker compose` CLI。
  - 推荐 persistence 使用模块自有 `database/sql + migrations` 模式。
  - 推荐为 `project` 与 `container` 之间新增 narrow stable shared boundary，而不是直接 import container private service。
  - Phase 1 的 Activity 仍由前端复用现有 container APIs 聚合。
  - Phase 1 的配置页只读。
- 当前下一步：按 topic-completion-loop 推进 Phase 1 的第一个实现 batch。

## Pending Batch Direction

- `phase-1-batch-1-project-contract-and-data-model`
- `phase-1-batch-2-server-project-module-import-and-refresh`
- `phase-1-batch-3-server-lifecycle-and-container-aggregation-boundary`
- `phase-1-batch-4-web-project-list-detail-and-readonly-configuration`
- `phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync`
- `phase-2-managed-create-editor-and-deploy`
- `phase-3-discovery-git-template-and-remote-host`

## Validation Targets

当前文档切片：

```bash
git diff --check
```

后续实现切片默认目标：

```bash
git diff --check
node scripts/openapi-bundle.mjs
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Loop Entry

- 推荐使用：`ai-plan/public/compose-project-management/startup-prompt.md`
- 推荐执行模式：`$graft-multi-agent-loop`
