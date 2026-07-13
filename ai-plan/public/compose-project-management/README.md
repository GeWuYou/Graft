# Compose Project Management

## 当前状态摘要

- 当前主题目标是在 `Graft` 增加以 Compose Specification 为首个部署模型的 Application 管理能力。
- 当前状态：`active`；应用管理与运行目标关联已完成验收，当前进入 Application ID、稳定 Workspace 与三步创建流程的 authority repair。
- 任务分类为 `cross-boundary`，涉及 `ai-plan/design`、future OpenAPI、future `server/modules/project/**`、future `web/src/modules/project/**`，并与现有 `container` runtime authority 协作。
- Canonical design：`ai-plan/design/domains/compose/Compose项目管理设计.md`。
- 当前已完成 Phase 0、Phase 1、Phase 2 的主要实现，但主题仍处于 `active`，因为产品入口、lifecycle authority 和 topic 完成口径出现了 drift repair 待修复项。
- 当前处于同一个 `topic-completion-loop` 下继续修复入口 IA、topic truth 与实际可用性之间的偏差，而不是新主题。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/domains/compose/Compose项目管理设计.md` + `ai-plan/design/domains/container/容器管理设计.md` + future OpenAPI source + future `server/modules/project/**` + future `web/src/modules/project/**`

## Owned Scope

当前 topic 允许修改：

- `ai-plan/design/domains/compose/Compose项目管理设计.md`
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
3. `Project` 只拥有 registry、ownership、compose files、lifecycle configuration、lifecycle actions、services aggregation、activity aggregation entry。
4. `Container` 继续拥有 runtime state、stats、logs、events、shell、inspect、networks、mounts。
5. Phase 1 的 Activity 继续复用现有 container logs/events，由前端做 fan-out 聚合。
6. Phase 1 的 Configuration 保持只读，且 API 拆为 metadata/list、preview、single-file content 三类。
7. `Application ID`、Workspace Key/Path、`Display Name` 与 `Compose Project Name` 必须分离。
8. `Snapshot` 只保存最近一次成功解析结果：`normalized compose + config hash + refresh metadata`。
9. `Unregister` 是安全默认；`Destroy` 必须受 ownership proof 保护。
10. Runtime Target 是 Provider-neutral 的连接与 capability authority；当前只公开 Local Docker，未来 Podman/Containerd 作为 Target Provider 接入，而不是新部署模型。

## Phase Plan

- Phase 0：设计 authority、topic recovery、loop startup prompt。已完成。
- Phase 1：Import Existing Project、Project Registry、Overview、Services、Configuration Read Only、Activity frontend aggregation、`up/down/restart`、Refresh、Unregister、Destroy guard。
- Phase 2：Managed Project Create、Compose Editor、Env Editor、Diff、Validate、Deploy。
- Phase 3：Git Project、Templates、Directory Scan、Auto Discovery、Remote Host、Project Activity backend aggregation。

## Current Recovery Point

- 设计 authority 已创建：`ai-plan/design/domains/compose/Compose项目管理设计.md`
- active topic 已创建：`ai-plan/public/compose-project-management/`
- 当前共识：
  - 推荐新增独立 `project` module，而不是扩展 `container` module 承担项目注册。
  - 推荐静态解析使用 `compose-go`，生命周期执行使用 `docker compose` CLI。
  - 推荐 persistence 使用模块自有 `database/sql + migrations` 模式。
  - 推荐为 `project` 与 `container` 之间新增 narrow stable shared boundary，而不是直接 import container private service。
  - Phase 1 的 Activity 仍由前端复用现有 container APIs 聚合。
  - Phase 1 的配置页只读。
  - Phase 2 已在同一 topic 内完成 managed create/edit/diff/validate/deploy 的核心实现，但主入口 IA 被 Phase 3 boundary work 偏移，需要先修复入口 truth。
  - `Import Existing Project` 的主入口应是 runtime candidate，而不是 folder picker。
  - 当前 import 的 `directory browse / inspect` 能力继续保留，但只作为非主入口 inspect/file-system 复用底座。
  - `config_files` 是 runtime candidate 的 stronger authority；`working_directory` 是 hint，可在缺少 label 时由 `config_files[0]` 派生。
  - 本地项目统一收口到保存型 `Lifecycle Configuration` authority：managed 默认 `confirmed`；运行时导入在向导内强制审核配置，并与注册一起保存为 `confirmed`。
  - `update-deploy` 不再作为一等动作保留；`redeploy` 成为唯一 deploy-style lifecycle action，pull/down/prune 等语义统一由 lifecycle configuration 持有。
  - Phase 3 继续留在同一 topic 内推进，但不得再让 boundary surface 取代 Phase 1 import 或 Phase 2 managed create 主入口。
  - Application Management 已稳定消费 Runtime Target 的摘要；Compose 是当前唯一可执行 Deployment Type，Container 保持 runtime authority。
  - 用户私有 Saved View 是可复用分页列表基础能力，保存筛选、每页大小和可见列，不保存当前页；同一用户和 surface 下名称唯一。
- 当前下一步：先完成 Application ID、Workspace 和创建流程的 cross-boundary authority repair；Remote Host、Git 与 Provider 扩展不在当前公开 surface。

## Completion Scope

用户批准的创建顺序是 `Deployment Type -> Runtime Target -> Source`：当前 Deployment Type 只有 Compose；Runtime Target 只列出已登记且具备 Compose capability 的 Local Docker；来源为 `Blank`、`Template` 与 `Import Existing`。三者都在 Workspace 解析完成后复用同一 CreationCommand pipeline；创建不自动部署，部署仍是独立操作。

受管 Workspace 固定由 Graft 在 Application Root 下按唯一 `workspace_key` 创建；它不按 Docker/Podman 或 Compose/Kubernetes 分层。`Application ID (app_<ULID>)` 是公开稳定标识，`Compose Project Name` 是可变部署 identity，均不由用户在创建表单填写。数据库 registry 是元数据真相，Workspace 文件是内容真相；不引入 `graft.yaml`。

Git、Remote Host、ZIP 与 GitHub Template 被明确延后：不保留可点击页面、公开 API、创建方式枚举或占位入口。未来能力只能在其真实向导、共享契约和创建方式目录同时实现后公开。

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
