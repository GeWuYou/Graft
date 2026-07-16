# Compose项目管理设计

本文档定义 `Graft` 对 Compose Application 的产品 IA、模块边界、数据模型、导入语义、API 方向、风险边界与阶段路线。

## Application 一次性迁移裁决

- 产品对象、UI route、HTTP resource、OpenAPI schema/type 与参数语义统一使用 `Application`。
- canonical UI route 为 `/applications/**`，canonical HTTP route 为 `/api/ops/applications/**`；资源路径参数固定为
  `applicationId`，公开 ID 固定为 `app_<ULID>`。
- 持久化主表统一为通用 `applications`，当前记录固定 `application_type=compose`。`Compose Project Name` 仅表示
  Compose deployment identity，不再决定产品对象或公开资源名称。
- 公开字段固定为 `source_type`、`compose_project_name`、`workspace_path`；`host_scope` 不属于 Application
  authority，运行位置由 `runtime_target_id` 表达。
- 本次迁移在首个正式版本前一次性完成；不保留 `/projects`、`/api/ops/projects`、Project schema/type、旧字段、
  alias、redirect 或 deprecated compatibility contract。
- 已存在的 versioned migration SQL 是历史证据，不得修改；数据库重命名与数据搬迁只能通过新的前向迁移完成。
- `server/modules/project/**` 与 `web/src/modules/project/**` 可暂时保留历史实现目录名，但不能向公开 contract
  泄漏 `Project` 资源语义。

该能力的核心定位必须保持稳定：

- `Application` 是用户可见且公开的唯一产品对象；当前 `Project` module 是 Compose Application 的注册、聚合与生命周期实现 owner，不是新的 Runtime。
- Application 是 Compose 的唯一业务入口和生命周期 authority；`up`、`down`、`restart`、`redeploy` 不得迁入 Docker Provider 菜单。
- `Container` 始终是 Runtime Authority。
- `Project` 只负责项目注册、配置解析、生命周期管理和聚合入口。
- `Project` 不得复制、替代或持久化容器运行时真相。

Runtime Target 统一拥有 Provider 连接与能力发现；Compose Project 只绑定并引用目标，不能自行维护另一套 Docker、Podman 或 containerd endpoint 与凭据目录。Project 详情可以跳转到关联 Provider 资源，但跳转不得改变上述 authority。

每个 Compose Application 绑定一个具备 Compose 执行和 Workspace 访问能力的 `runtime_target_id`。当前实现只允许 Local Docker；未来 Local/Remote Docker、Local/Remote Podman 或 Containerd target 只有在对应 Provider adapter 报告真实能力后才可选择。`applications.runtime_target_id` 迁移期允许为空，以便迁移先于 Runtime Target Boot 执行；Project Boot 在发现 Local Docker 后幂等回填历史本地记录。该桥的 authority 是 Runtime Target，影响者是 Application 列表与生命周期，清理条件是生产回填观测确认无 live 空引用后另行迁移为非空。

`/applications` 是稳定 URL 下的“应用管理”页面，不以 Compose 作为页面身份。当前 Compose 只作为 `application_type=compose` 的实现和生命周期能力；列表必须首先展示应用类型、运行目标与提供者，并把筛选交给服务端。快捷筛选是用户私有、surface-scoped 的通用分页视图：保存可见筛选、每页大小与可见列，不保存当前页；同一用户同一 surface 的展示名唯一，可创建、更新、删除和复用，但不共享。

菜单图标统一由 web 的 Iconify resolver 消费 server descriptor identifier：常规导航使用 Lucide，专业补充使用 Tabler，品牌使用静态 Iconify 数据。Docker 必须使用 Tabler 的 Docker brand glyph，不能以通用 server/container 图标代替；Iconify 不得通过运行时 CDN 加载图标。

## 1. 启动治理与 Authority

- 任务分类：`cross-boundary`
- governance source：root `AGENTS.md`
- 设计 authority：本文档
- 后端 authority：future `server/modules/project/**`
- 前端 authority：future `web/src/modules/project/**`
- OpenAPI authority：`openapi/**`
- 容器运行时 authority：现有 `server/modules/container/**` + `web/src/modules/container/**`
- 相关治理文档：
  - `ai-plan/design/domains/container/容器管理设计.md`
  - `ai-plan/design/domains/container/容器运行时事件能力设计.md`
  - `ai-plan/design/domains/container/容器资源状态与订阅治理设计.md`
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
  - `ai-plan/design/architecture/前端架构设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md`
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`

本能力后续实现必须遵守 authority-first：

- Compose 文件、项目注册、导入、刷新、`up/stop/restart`、销毁保护，由 `project` module 拥有。
- 容器详情、日志、事件、Stats、Shell、Inspect、Network、Mount 仍由 `container` module 拥有。
- 若 `project` 需要消费容器运行时摘要，必须通过稳定共享边界复用，不得直接依赖 `container` module 私有实现。

## 2. 设计目标、非目标与术语

## 2.1 目标

- 支持把本机现有 Compose Project 导入 `Graft` 管理。
- 支持保存项目注册信息、工作目录、Compose 文件选择、环境文件选择与最近一次成功解析快照。
- 支持项目级 `Overview`、`Services`、`Configuration`、`Lifecycle`、`Logs` 五类信息架构。
- 支持项目级 `Refresh`、`Up`、`Stop`、`Restart`、`Unregister`、`Destroy` 管理动作。
- 复用现有容器运行时能力，而不是新增第二套运行时真相。
- Phase 1 只做本机 `local host`，但模型要为未来远程主机预留边界。

## 2.2 非目标

- 不复制 Compose 文件。
- 不移动项目目录。
- 不重写用户已有目录布局。
- 不在 `Project` 下新增第二套容器详情页。
- 不在 `Project` 层持久化容器日志、事件、Stats 或运行时快照。
- Phase 1 不做 Git Project、目录扫描、自动发现、远程主机。
- Phase 1 不做 Diff、Deploy、Validate UI。
- Phase 1 不做 Project Events API、project-owned container detail，且不持久化项目运行时日志或实时缓存。
- Phase 1 的项目详情实时快照与项目日志聚合只提供 bounded backend aggregation surface，不复制容器运行时 authority。

## 2.3 术语

- `Application`
  - 用户创建、查看和管理的业务资产；当前首期由 Compose Project registry 实现，不能把底层 `Project` module 名称当作 Runtime 或产品入口。
- `Project`
  - 仅指历史实现 module/package 名称或 Compose 原生技术术语；不再是公开产品对象、HTTP resource、schema/type
    或持久化主表名称。
- `Deployment Type`
  - 应用模型与文件/生命周期语义，例如 `compose`、`swarm`、`kubernetes`、`nomad`。Compose 基于 Compose Specification；Docker 和 Podman 不是 Deployment Type，也不得出现 `docker-compose` 或 `podman-compose` 两个一级应用模型。
- `Application Source`
  - 在已选择 Deployment Type 和 Runtime Target 后，取得或构建 Application Workspace 的方式；当前 Compose 可执行来源为 `blank`、`template`、`import`。`git` 仅可作为禁用的路线图卡片出现，不是当前创建方式。Source 不是 Deployment Type、Provider 或菜单对象。
- `Runtime Target`
  - 应用实际绑定的运行目标，拥有连接与能力发现，例如 Local Docker、Remote Docker、Local Podman。当前只有已登记且报告 Compose capability 的 Target 可选。
- `Runtime Provider`
  - Target 的底层接入实现，例如 Docker、Podman、containerd；它属于 Infrastructure，不得替代 Deployment Type。
- `Imported`
  - 指项目文件本来就存在于宿主机某个路径，`Graft` 只登记并管理它，不复制、不移动、不重写其布局。
- `Managed`
  - 指未来由 `Graft` 在受管根目录下创建的项目；它是 Source 概念，不自动等价于“允许销毁目录”。
- `Application ID`
  - 对外稳定标识，格式为 `app_<ULID>`；不因显示名、工作区或运行目标变化而改变。
  - 作为 Project 与 Task Runtime 等跨模块能力之间的资源引用；内部数值 `id` 仅限 Project 自有表关系和存储实现。
- `Workspace Key` / `Workspace Path`
  - 受管应用的稳定单层安全 key 与实际工作区路径。Graft 提议可用 key 并生成 `<application-root>/<workspace-key>`；用户可在表单中编辑 key，但不能输入任意相对目录。默认 key 冲突自动加后缀，显式 key 由服务端校验唯一性；不按 Provider 或 Deployment Type 分层。导入的外部工作区只记录其既有路径。
- `Compose Project Name`
  - Compose deployment identity；优先由顶层 `name:` 取得，缺失时由服务端生成并写入受管 Compose 文件。CLI 执行、容器归属匹配和项目成员识别都以它为准；不是 Application ID、显示名或目录名。
- `Display Name`
  - `Graft` UI 展示名称；允许独立修改；不影响 Docker Runtime。
- `Snapshot`
  - 最近一次成功解析的静态配置结果，不是运行时缓存。
- `Drift`
  - 当前文件签名与最近一次成功刷新快照不一致的状态。

## 2.4 Application 创建决策与 IA

创建流程必须按三个正交决策组织：

```text
Application list
  -> Create Application
  -> Deployment Type
  -> Runtime Target
  -> Application Source
  -> Workspace / registry creation
```

首期 Deployment Type 页展示 `Compose`（基于 Compose Specification）、`Swarm`（Docker Stack）、`Kubernetes`（Deployment/Pod）与 `Nomad`（Job）。仅 `Compose` 可点击；其它卡片必须禁用、不可键盘触发，并在 hover/focus 说明“暂不支持”。它们是产品路线图，不得生成菜单、OpenAPI catalog、Provider contract、持久化枚举值或空实现 API。

选择 Compose 后进入 Runtime Target 页，只列出已登记、健康且具备 `compose_execution` 与 `workspace_access` capability 的 Target；当前是 Local Docker。该页复用运行目标卡片的 Provider 标识与交互，不虚构 Remote Docker、Podman 或 Containerd 卡片。随后才进入 Source：`blank`、`template`、`git`、`import`。其中 Git 必须是禁用、不可键盘触发的路线图卡片，并在 hover/focus 通过本地化 tooltip 显示“暂不支持”；它没有路由、API、创建方式枚举、持久化来源类型或占位页面。其余三个来源才取得或物化 Workspace，并进入同一 Compose Project creation pipeline。

UI route 的 canonical 语义固定为：

| UI route | 页面和约束 |
| --- | --- |
| `/applications/create` | Deployment Type picker；不在 URL 中暴露 Provider hierarchy |
| `/applications/create/target?deployment=compose` | Compose Runtime Target picker；无效或缺失 deployment 回到 Deployment Type picker |
| `/applications/create/source?deployment=compose&runtime_target_id=<target-id>` | Compose Source picker；无效或缺失选择回到上一步 |
| `/applications/create/blank?deployment=compose&runtime_target_id=<target-id>` | Compose 空白 Workspace 向导 |
| `/applications/create/template?deployment=compose&runtime_target_id=<target-id>` | Compose 模板向导 |
| `/applications/create/import?deployment=compose&runtime_target_id=<target-id>` | Compose 导入向导 |

`/applications/**` 是唯一 Application 领域 URL；不得引入 `/projects/**`、`/docker/**`、`/compose/**` 或
`/kubernetes/**` 的创建层级。实现前尚未发版，不保留旧 route、旧来源选择或 alias redirect。

## 3. 架构分析

## 3.1 现有模块与职责清单

| 领域                         | 现有路径                                                                                                     | 当前职责                                                                                       | 对 Compose Project 的意义                                            |
| ---------------------------- | ------------------------------------------------------------------------------------------------------------ | ---------------------------------------------------------------------------------------------- | -------------------------------------------------------------------- |
| Project                      | 无专用 module                                                                                                | 仓库当前没有项目注册或 Compose Project 管理模块                                                | 必须新增 `server/modules/project/**` 与 `web/src/modules/project/**` |
| Container backend            | `server/modules/container/**`                                                                                | 容器列表、详情、日志、Shell、start/stop/restart/remove、资源摘要、运行时事件、Compose 来源识别 | 当前唯一可复用的 Runtime Authority                                   |
| Container frontend           | `web/src/modules/container/**`                                                                               | 容器列表页、详情页、日志、运行时交互                                                           | `Project -> Services -> Container Detail` 的现成落点                 |
| Docker provider              | `server/modules/container/docker_runtime.go`                                                                 | Docker Runtime adapter；读取容器元数据、labels、日志、stats、事件                              | 当前对 Compose 的认知仅限容器 labels                                 |
| Runtime orchestration        | `server/modules/container/service.go`                                                                        | 容器用例编排、权限、审计、系统配置消费、实时 topic 注册                                        | 当前是 HTTP-first、module-private，用于复用时需要稳定共享边界        |
| Realtime / events            | `server/modules/container/runtime_events*.go`、`log_topic_streamer.go`、`resource_stats_cache.go`            | 容器事件流、日志 topic、资源采集与缓存                                                         | 可复用为 Activity 的底层事实来源，但 Phase 1 不新增项目级聚合 API    |
| Filesystem access            | `server/modules/container/mount_usage.go`、`server/internal/config/config.go`                                | 挂载扫描、`.env` 与仓库根路径发现、基础文件系统读写                                            | 有基础文件系统经验，但没有 Compose 文件读取抽象                      |
| Configuration core           | `server/internal/config/**`、`server/modules/container/config.go`                                            | Viper 配置、系统配置定义与读取                                                                 | 可复用系统配置框架承载 `Managed Projects Root` 等配置                |
| Database persistence pattern | `server/modules/notification/store/sql_repository.go`、`server/modules/announcement/store/sql_repository.go` | 模块自有 `database/sql` repository + migration pattern                                         | `project` module 推荐沿用该模式，而不是先回到 Ent 中央仓储           |
| Module runtime / DI          | `server/internal/module/**`、`server/internal/container/**`、`server/internal/moduleapi/**`                  | 模块生命周期、服务注册、跨模块稳定接口                                                         | `project` 需要在这里定义最小稳定共享边界                             |
| OpenAPI / generated schema   | `openapi/**`、`web/src/contracts/openapi/generated/schema.ts`                                                | 服务端与前端共享 wire contract                                                                 | `project` 的 REST contract 必须走同一路径                            |

## 3.2 当前依赖图

```text
web/src/modules/container
  -> web module api / generated schema
  -> server/modules/container/route_registration.go
    -> server/modules/container/service.go
      -> server/modules/container/docker_runtime.go
      -> runtime_events / log_topic_streamer / stats_collector
      -> moduleapi.AuthService / UserService / Authorizer / SystemConfigResolver
      -> eventbus(audit)

server/modules/notification | server/modules/announcement
  -> store/sql_repository.go
    -> database/sql
    -> module-owned migrations

server/internal/config
  -> env file discovery
  -> repository root / server root detection
  -> viper-backed config loading
```

当前 Compose 相关链路只有：

```text
Docker container labels
  -> server/modules/container/docker_runtime.go
    -> compose project / service metadata
      -> OpenAPI ContainerOrchestratorInfo
        -> web container list / detail
```

结论：

- 当前仓库已经有很强的容器运行时能力，但没有 Compose Project registry。
- 当前仓库有 Compose 来源识别，但没有 Compose 静态配置解析、生命周期执行或项目级持久化。
- 当前 `container` module 并未向其它模块暴露稳定的 Compose Project 聚合服务；它主要面向自己的 HTTP 路由和页面。

## 4. 现有能力盘点

| 能力                      | 当前状态       | 证据结论                                                                               | 可复用建议                                  |
| ------------------------- | -------------- | -------------------------------------------------------------------------------------- | ------------------------------------------- |
| Compose Project detection | `partial`      | 已通过 `com.docker.compose.project` / `com.docker.compose.service` labels 识别容器来源 | 复用为项目成员匹配与运行态聚合基础          |
| Compose parsing           | `no`           | 仓库内无 Compose 文件解析器、无 `compose-go` 使用                                      | 新增 `project` module 静态解析能力          |
| Compose up/stop/restart   | `no`           | 容器 module 仅有单容器动作，没有 `docker compose up/stop/restart` 执行层               | 新增项目生命周期执行器                      |
| Compose logs              | `yes`          | 已有容器日志 API / 流能力，且项目侧补充了 project-owned 聚合读取与实时 topic           | Phase 1 由 `project` 聚合并显式保留来源字段 |
| Compose config            | `no`           | 无 `docker compose config` 或等价规范化预览接口                                        | 新增静态规范化快照与预览能力                |
| Compose events            | `indirect yes` | 已有容器事件流，但当前仍没有项目级事件聚合 API                                         | 继续保持容器 authority，后续按需扩展        |

## 4.1 可直接复用的组件

- 容器详情、日志、事件、Shell、Mount、Inspect、Network 等全部继续由 `container` module 提供。
- 容器列表对 Compose labels 的识别可以复用为 Project 成员归属判断。
- 审计、权限、菜单、系统配置、模块 SQL repository、OpenAPI 生成链都已有成熟路径。

## 4.2 当前缺口

- 无 `Project Registry`
- 无 `Import Existing Project`
- 无 Compose 文件解析与标准化快照
- 无项目级生命周期执行器
- 无项目级所有权与销毁保护
- 无项目级 Drift model
- 无项目级前端模块
- 无容器运行时聚合的稳定共享边界

## 5. 差距分析与关键决策

## 5.1 为什么不能直接把 Project 做进 container module

因为这会把两个不同 authority 混在一起：

- `container` 的 authority 是单个容器运行时事实。
- `project` 的 authority 是 Compose Project 的注册、配置、文件与生命周期。

若直接把 Project 做进 `container` module，会产生以下漂移风险：

- 让容器运行时模块同时拥有文件系统与项目注册真相。
- 诱导在 `Project` 详情里复制容器详情字段。
- 诱导为项目新增一套日志、事件、Stats 聚合后端缓存。

因此必须拆成独立 `project` module。

## 5.2 为什么需要新增稳定共享边界

当前 `container` module 的 service 是 module-private，没有 stable `moduleapi` 暴露给其它模块。

若 `project` 直接依赖 `server/modules/container/service.go` 或 `docker_runtime.go`：

- 会破坏模块边界。
- 会把 HTTP-first / module-private 逻辑变成跨模块耦合点。
- 会让后续 runtime provider 演进更困难。

因此推荐新增一个最小共享边界，例如 future `server/internal/moduleapi/container.go`：

- 只暴露项目聚合所需的只读容器摘要能力。
- 不暴露容器 module 内部实现。
- 不把容器日志、事件、Shell 等高耦合运行时流能力拉进 `project` module。

## 5.3 Compose 技术路径决策

推荐拆分为两个边界：

- 静态解析与导入校验：`compose-go`
- 生命周期执行：`docker compose` CLI

理由：

- Compose 文件导入、合并、插值、标准化更适合静态解析库。
- 真实 `up/stop/restart` 语义应尽量复用 Docker Compose CLI 的行为，而不是自己模拟。
- 这样可以把“读配置”和“改运行态”分开治理。

## 6. 推荐架构

## 6.1 后端模块结构

future `server/modules/project/**` 推荐至少包含：

```text
server/modules/project/
├─ descriptor.go
├─ module.go
├─ module_registration.go
├─ route_registration.go
├─ service.go
├─ contract/
├─ store/
├─ migrations/
├─ compose/
│  ├─ loader.go
│  ├─ executor.go
│  └─ diagnostics.go
├─ fs/
│  ├─ resolver.go
│  └─ hashing.go
└─ locales/
```

职责划分：

- `service.go`
  - 统一编排 Project registry、import、refresh、lifecycle、ownership guard。
- `store/`
  - 模块自有 SQL repository。
- `compose/loader.go`
  - 使用 `compose-go` 读取、标准化、验证、生成快照。
- `compose/executor.go`
  - 使用参数化命令执行 `docker compose up/stop/restart`，并把 `docker compose down` 保留给 destroy。
- `fs/`
  - 负责 working directory 解析、文件存在性检查、hash 计算、symlink 安全校验。

## 6.2 前端模块结构

future `web/src/modules/project/**` 推荐至少包含：

```text
web/src/modules/project/
├─ index.ts
├─ bootstrap-routes.ts
├─ api/
├─ contract/
├─ pages/
│  ├─ list/
│  └─ detail/
├─ components/
└─ locales/
```

产品结构：

```text
Projects
  -> Project Detail
     -> Overview
     -> Services
     -> Configuration
     -> Activity
```

## 6.3 推荐未来依赖图

```text
web/src/modules/project
  -> project api / generated schema
  -> server/modules/project/route_registration.go
    -> server/modules/project/service.go
      -> ProjectRepository(database/sql)
      -> ComposeLoader(compose-go)
      -> ComposeExecutor(docker compose CLI)
      -> FilesystemResolver / Hashing
      -> moduleapi.ContainerRuntimeReadService(future narrow boundary)
      -> moduleapi.SystemConfigResolver
      -> eventbus(audit)

web Project Activity tab
  -> project services endpoint
  -> existing container logs/events endpoints
  -> existing container realtime topics
```

## 6.4 硬边界

`Project` 负责：

- Project Registry
- Source 与 Ownership
- Working Directory
- Compose Files / Env Files
- Canonical Project Name / Display Name
- Compose Snapshot
- Drift Detection
- Project Lifecycle
- Services Aggregation
- Activity Aggregation entry

`Container` 负责：

- Runtime State
- Stats
- Logs
- Events
- Shell
- Inspect
- Networks
- Mounts

明确禁止：

- `Project` 持久化容器运行时状态
- `Project` 实现自己的 Container Detail
- `Project` 保存容器 Logs / Events / Stats
- `Project` 新建第二套 Runtime Dashboard

## 7. 数据模型提案

## 7.1 领域对象

### Application

- `id`（内部数值主键）
- `application_id`（对外 `app_<ULID>`，不可变且唯一）
- `display_name`
- `application_type`
  - 当前固定 `compose`
- `compose_project_name`
- `compose_project_name_source`
  - `declared | generated | derived`
- `source_type`
  - `imported | managed | template`
- `runtime_target_id`
- `workspace_key`（受管项目必填）
- `workspace_path`
- `ownership_mode`
  - `external | managed-root-dedicated`
- `last_observed_config_hash`
- `last_drift_checked_at`
- `drift_status`
  - `unknown | clean | changed | missing`
- `lifecycle_strategy_kind`
  - 当前固定 `standard`
- `lifecycle_review_status`
  - `review_required | confirmed`
- `lifecycle_config`
  - `profiles`
  - `down_before_redeploy`
  - `pull_before_redeploy`
  - `build_before_up`
  - `force_recreate`
  - `wait_after_up`
  - `prune_images_after_redeploy`

上述 `application_id`、`application_type`、`source_type`、`workspace_key`、`workspace_path` 与
`compose_project_name` 是 canonical 字段。旧 `source_kind`、`canonical_project_name`、`working_directory`、
`host_scope` 与 `relative_project_directory` 只能作为历史迁移输入被一次性搬迁，不得进入新 contract、UI 或兼容层。

### ProjectFile

- `id`
- `project_id`
- `kind`
  - `compose | env`
- `role`
  - `primary | override | env`
- `absolute_path`
- `display_path`
- `order_index`
- `last_observed_hash`

### ProjectSnapshot

只保存：

- `project_id`
- `normalized_compose_json`
- `config_hash`
- `refreshed_at`

定位：

- 它是“最近一次成功解析结果”。
- 它不是“Project Runtime Cache”。

### ProjectServiceView

它是查询模型，不是持久化实体：

- `service_name`
- `declared image/build/ports/volumes/networks`
- `container_members`
- `running_count`
- `stopped_count`

其中静态定义来自 `normalized_compose_json`，运行态数量来自容器聚合。

## 7.2 数据库存储提案

Phase 1 推荐三张模块自有表：

### `applications`

用途：

- 通用 Application 注册真相；当前 `application_type` 固定为 `compose`
- Source / Ownership / Drift / Refresh 元数据

不存：

- 容器运行时明细
- 日志
- 事件
- Stats

### `compose_project_files`

用途：

- 保存 Lifecycle authority 使用的 Compose 文件与 Env 文件清单
- 支持未来多文件、有序 `-f` 合并与独立文件读取

边界：

- 它不是 Configuration 工作台左侧文件树的数据源。
- Configuration 工作台必须来自 `workspace_path` 的真实目录浏览接口，而不是从这里推断文件列表。
- 它不再是 workspace state 的 source of truth。
- 它只继续服务 compose parsing、preview、validation、lifecycle 与 deployment 所需的 compose/env 元数据覆盖层。

### Application Root templates

`Application Root Directory` 下的 `templates/` 是受管工作区之外的保留运行时目录。当前默认模板固定在
`<application-root>/templates/default`；它是 Template source 的内容 authority，不是模块内置文件或前端常量。

- Project module 在目录缺失时以发布随附的种子资源初始化 `templates/default`；已有目录或文件绝不覆盖，以保留管理员维护的模板。
- `default` 当前提供 `.env` 与 `compose.yaml` 两个示例文件；它们只定义 Blank/Template 的初始体验，不限制工作区可创建、读取或编辑的文件名、扩展名、层级或目录。
- `templates/` 的合法一级子目录是可发现的模板 key，`default` 是默认选择；一个模板可包含任意安全相对路径的 UTF-8 文本文件、嵌套目录和空目录，materialize 时完整复制。
- 创建 contract 由服务端提供可用模板目录清单、所选模板的安全标识及 Blank 默认草稿；web 不直接读取 Application Root，也不根据目录名推导模板内容。
- `templates/` 本身及其子路径不能作为 `workspace_key`、项目工作区、导入目标或项目文件 API 的可访问根目录。

### `compose_project_snapshots`

用途：

- 保存最近一次成功解析的规范化 Compose 快照

约束：

- 每个 project 只保留一条 latest successful snapshot
- 刷新失败时不覆盖旧 snapshot，只更新 project refresh error / drift 状态

## 7.3 推荐索引与唯一性

- `applications(application_id)` 唯一
- `applications(runtime_target_id, compose_project_name)` 唯一
  - 防止同一目标重复注册同一个 Compose deployment identity
- live `applications(workspace_path)` 唯一
- `compose_project_files(project_id, order_index)` 唯一
- `compose_project_files(project_id, absolute_path)` 唯一
- `compose_project_snapshots(project_id)` 唯一

## 7.4 必须存储的元数据

- Source：`Imported` / future `Managed`
- Ownership 模式
- Workspace Key / Workspace Path
- Compose 文件有序列表
- Env 文件有序列表
- Display Name
- Compose Project Name 及来源
- 最近一次成功解析快照
- Hash 与 Drift 状态
- 最近一次刷新时间与错误摘要
- 创建/更新审计字段

## 8. 生命周期设计

## 8.1 Create Project

结论：

- Phase 1 不实现。
- Phase 2 才实现受管项目创建。

语义：

- `Create Project` 指在 `Managed Projects Root` 下创建一个受管项目目录与基础配置。
- 它不是导入，不会覆盖已有外部目录。

## 8.2 Import Existing Project

`Imported` 的准确含义：

- Compose 项目在宿主机上已存在。
- `Graft` 只登记其工作目录、文件集合、运行时身份与快照。
- `Graft` 可以对它执行 `refresh/up/stop/restart` 等管理动作。
- `Graft` 不复制、不移动、不改写目录布局。

## 8.3 Refresh Project

`Refresh` 负责：

- 重新读取工作目录与选定文件
- 重新计算当前文件 hash
- 重新执行 Compose 静态解析与标准化
- 更新 latest successful snapshot
- 更新 drift 状态
- 重新计算 Services 聚合视图

刷新失败时：

- 保留上一份成功 snapshot
- 不额外持久化 refresh failure 字段
- 在当前调用结果中直接返回错误，并保持 Overview / Configuration 继续基于上一份成功 snapshot 工作

## 8.4 Lifecycle Configuration / Up / Stop / Restart / Redeploy

这些动作属于 `Project Lifecycle`，不属于 `Container`。

Lifecycle Configuration 是本地项目统一的生命周期 authority：

- `managed`
  - 在 create 时由 `Project` 直接生成并保存默认 lifecycle configuration
  - 默认进入 `confirmed`
- `imported`
  - 只能从 working directory、tracked compose files、canonical project name 推导出可恢复的最小 authority
  - `profiles`、`pull/build/wait/force-recreate/prune` 等不可从 Docker runtime 历史可靠恢复
  - 因此导入向导必须先展示可编辑的默认 lifecycle configuration；操作员确认后，项目注册与配置一起保存为 `confirmed`
- `template`
  - 当前作为受管 Workspace 的 materialized source，沿用 managed lifecycle configuration。
- `git`、`remote-host`
  - 尚未进入 lifecycle 或来源策略枚举；Git 仅在来源选择页以禁用路线图卡片表达，Remote Host 不展示。

标准策略当前固定为 `standard`：

- `Project` 保存结构化配置，而不是保存一整段原始 shell 命令
- working directory、ordered compose files、canonical project name 继续由 project registry authority 拥有
- lifecycle configuration 只保存可编辑的 compose 执行选项
- `additional_args` 以受限的 argv token 列表持久化，并只追加到 `up` / `redeploy` 的 `compose up`；不得承载 shell 表达式或覆盖项目 authority 的 `-f`、`-p`、profile 等参数
- UI 可展示 generated command preview，但 preview 是 derived artifact，不是第二套 authority
- 若项目仍是 `review_required`，`up/stop/restart/redeploy` 必须先被 guard 拦住，直到用户确认或更新配置

语义约束：

- `Up`
  - 基于已保存 lifecycle configuration 生成 `docker compose up -d` 预览并执行
- `Stop`
  - 基于已保存 lifecycle configuration 生成 `docker compose stop` 预览并执行
  - 仅停止当前项目运行中的服务和容器，不删除容器、网络或卷
- `Restart`
  - 基于已保存 lifecycle configuration 生成 `docker compose restart` 预览并执行
- `Redeploy`
  - canonical deploy-style lifecycle action
  - 标准策略下根据保存配置决定是否先 `down`、是否 `pull`、然后 `up -d`，并可选 image prune
- `Destroy`
  - 执行 `docker compose down`
  - Phase 1 默认不删除 volumes，只有显式 destroy 选项才允许继续破坏性清理

执行时必须显式传入：

- `-p/--project-name`
- `--project-directory`
- 有序 `-f` 文件列表
- 重复 `--profile`
- 明确 env file 参数

`update-deploy` 不再作为一等动作存在：

- `pull`、`down-before-redeploy`、`prune` 等都收口到 lifecycle configuration
- `redeploy` 成为统一的 runtime deploy-style lifecycle action
- Configuration Workspace 只负责将已确认的编辑写回工作目录；文件保存后统一由 `redeploy` 提交 lifecycle Task，不保留独立 `deploy` 动作或接口

### 8.4A Future Task Runtime Integration

当前 `up/stop/restart/redeploy` 的 Compose 业务 authority 仍归 `Project`：它验证 lifecycle configuration 和
guard，生成参数化 compose command plan，并通过 `Container` 的稳定 runtime reader 查询 health/status。

Project 已通过 `task` module 提交由 `project.compose.*` StageExecutor 构成的 `TaskPlan`，并返回 Task receipt，
不再同步等待长时间 Compose 子进程。`task` 负责阶段状态、日志、realtime、retry/cancel 和 crash recovery，且不依赖
Project。`down/pull/build/up/image-prune` 仍是 Project 定义的业务 Stage，而不是 Task Runtime 的内置知识。

Task Runtime 的 `unknown` Stage + `needs_attention` Task 语义适用于崩溃时无法判断结果的 Docker command；Project
不得自动重放这类 command，必须先由操作者完成实际 runtime reconciliation。

## 8.5 Remove Project

产品文案不建议继续用模糊的 `Remove Project`。

推荐拆成两个动作：

- `Unregister`
  - 只删除 `Graft` 的注册记录与快照
  - 不触碰宿主机目录
  - 不要求容器先停止
- `Destroy`
  - 高危操作
  - 可能执行 `down`
  - 可能删除独占 volumes
  - 可能删除受管目录
  - 最后注销注册记录

## 8.6 Ownership 规则

销毁权限不能只看 `Managed` / `Imported`。

必须看 `ownership proof`：

- `external`
  - 外部路径
  - 默认只能 `Unregister`
  - `Destroy` 不能删除工作目录
- `managed-root-dedicated`
  - 路径在受管根目录下且可证明为项目专属目录
  - 才允许目录级删除

Volume 删除要单独判断：

- 默认不删除 named volumes
- 只有显式勾选且可证明为独占引用时才允许删除

## 9. 导入流程设计

## 9.1 输入

当前 canonical import flow 应收口为 `runtime candidate -> inspect -> preview -> import`：

1. 用户打开 `Import Existing Project`
2. frontend 请求 runtime candidate 列表
3. backend 基于 `container` runtime authority 聚合 Compose import candidates
4. frontend 展示 `ready`、`already_imported` 与 `unavailable` candidates
5. 用户只能选择 `ready` candidate 进入 inspect
6. backend 基于 candidate authority 执行一次静态 inspect
7. frontend 展示 inspect preview
8. 用户只允许编辑：
   - `display_name`
   - `compose_project_name_override`
9. frontend 提交 `inspection_id` 驱动最终 import

当前 import confirmation 表单固定只包含：

- `display_name`
- `compose_project_name_override`

当前不再要求用户手填：

- `workspace_path`
- `compose_files[]`
- `env_files[]`

这些字段都必须来自 backend authority，而不是前端二次拼装。

现有 `directory browse / inspect` 能力继续保留，但它的角色是：

- 非主入口的 inspect/file-system 复用底座
- future 服务器终端或文件能力的可复用基础

它不再是当前 `Import Existing Project` 的 primary IA。

## 9.2 Runtime Candidate Authority

runtime candidate 的 authority owner 固定为 `container`，`project` 只消费其稳定输出，不直接解析 Docker labels。

candidate 至少要固定这些字段语义：

- `candidate_key`
- `compose_project_name`
- `status`
  - `ready`
  - `already_imported`
  - `incomplete_metadata`
  - `unsupported_runtime`
  - `broken_compose`
- `status_reason_codes`
- `importable`
- `runtime_type`
- `runtime_version`
- `workspace_path`
- `workspace_path_source`
- `config_files`
- `service_names`
- `container_counts`
- `warnings`

返回规则：

- runtime candidates 不能只返回 `ready`
- 对当前 runtime 中可见但不可导入的 Compose project，必须返回 `already_imported` 或 `unavailable` candidate，并给出稳定 reason code
- frontend 通过 `status + status_reason_codes` 展示不可导入原因，而不是靠候选“消失”表达失败

当前 batch 固定的最小 reason code 集：

- `already_imported`
- `missing_project_name`
- `missing_config_files`
- `invalid_config_files`
- `conflicting_runtime_metadata`
- `unsupported_runtime_type`
- `compose_parse_failed`
- `config_files_not_accessible`

## 9.3 Metadata Authority Rules

runtime candidate 的 stronger authority 固定为：

- `config_files`

`workspace_path` 只作为 hint，不作为 import identity。

规则：

- 若 runtime label 同时提供 `workspace_path + config_files`，直接使用
- 若缺少 `workspace_path` 但 `config_files[0]` 可用，允许由 `dirname(config_files[0])` 派生 working directory
- 上述派生成功时，candidate 仍可为 `ready`，但需返回 warning code 与 `workspace_path_source=derived_from_config_files`
- 只有 `config_files` 缺失、无效、不可访问，或同一 candidate 组内元数据冲突时，candidate 才进入不可导入状态

candidate grouping identity 建议固定为：

- `runtime_target_id + compose_project_name + normalized config_files digest`

而不是单独依赖 `workspace_path`。

## 9.4 文件发现

默认发现顺序：

1. `compose.yaml`
2. `compose.yml`
3. `docker-compose.yaml`
4. `docker-compose.yml`
5. `docker-compose.override.yml`

inspect / parse 层继续保留当前 live 规则：

- inspect 自动扫描：
  - 上述 compose primary / override candidates
  - `*.env`
  - `.env.*`
- 若发现多个 primary compose candidates，backend 仍按固定优先级选主文件并返回 warning。
- 合同层继续支持有序多文件；当前 import UI 不再暴露手工文件确认。

## 9.5 校验

inspect / import 前必须校验：

- selected runtime candidate 处于 `ready`
- candidate authority 对应的 Compose 文件存在
- 路径解析后仍在允许边界内
- Compose 语法与规范化解析成功
- env file 可读取
- Canonical Project Name 可计算
- 唯一性校验通过
- import 时 inspection snapshot 未过期、未 stale

保留的 directory browse / inspect path 仍需继续校验：

- selected directory 在 provider/root allowlist 边界内
- selected directory 存在且可访问

## 9.6 冲突检测

至少检测：

- 同一 `runtime_target_id + compose_project_name` 已存在
- 同一 `config_files digest` 已存在
- inspect 后文件 hash 与 import 时重算结果不一致
- candidate authority 指向的文件缺失
- 文件内容不是有效 Compose 配置

## 9.7 导入步骤

1. frontend 请求 `GET /api/ops/applications/import/runtime-candidates`
2. backend 从 runtime authority 聚合 Compose import candidates，并同时返回 `ready` 与 `unavailable`
3. 用户选择一个 `ready` candidate
4. frontend 提交 `POST /api/ops/applications/import/runtime-inspect`
5. backend 基于 candidate authority 解析 compose/env files，并只解析一次 compose authority
6. backend 生成：
   - `inspection_id`
   - `candidate_key`
   - `compose_project_name`
   - `display_name_suggested`
   - `compose_files`
   - `env_files`
   - `services / networks / volumes`
   - `warnings / conflicts`
7. frontend 只提交 `inspection_id + editable overrides` 到 `POST /api/ops/applications/import`
8. backend 校验 inspection TTL 与 file hash freshness
9. backend 持久化 `applications`、`compose_project_files`、`compose_project_snapshots`
10. 返回项目详情摘要

保留的 directory browse / inspect path 继续存在，但不再作为当前主入口 IA。

## 9.8 输出

持久化内容：

- 项目注册记录
- 文件清单
- 最近一次成功解析快照
- refresh / drift 元数据

导入完成后可用的运行态对象：

- Project Overview
- Services 聚合
- Configuration 只读视图
- Activity 前端聚合入口
- `Up / Stop / Restart / Refresh / Unregister / Destroy` 动作
- 容器详情跳转入口

## 10. API 提案

Phase 1 的 canonical OpenAPI authority 已收口到 `openapi/**`，本节继续保留 IA 与语义设计真相，不能与
`openapi/**` 漂移。

## 10.1 Application 列表与详情

项目列表可使用通用 `saved-view` module 保存用户私有的分页视图。`project` 仍是 `/api/ops/applications/saved-views` 的 HTTP 与领域授权 owner：它只在调用 generic service 前校验 `surface_key=application.list`、筛选状态和可见列。保存状态包含筛选/查询 JSON、`page_size` 与可见列键；不保存当前页，应用视图一律从第一页开始。视图不共享，展示名称在同一 `(owner_user_id, surface_key)` live scope 内唯一。

### 保存视图表设计摘要

- owner module：`saved-view`；表：`saved_views`，不是 `project` 专用偏好表。
- 生命周期：当前用户创建、更新、软删除自己的视图；所有读取固定 `deleted_at = 0`，用户和 consumer surface 都是查询边界。
- 索引：`(owner_user_id, surface_key, name) WHERE deleted_at = 0` 保障 live 名称唯一；`(owner_user_id, surface_key, updated_at DESC, id DESC) WHERE deleted_at = 0` 支撑列表。
- 状态：`query_state_json` 只保存 consumer 已校验的 JSON，`page_size` 和 `visible_columns_json` 可复用到任何分页列表；当前页不持久化，也不提供共享语义。

| Method | Path                              | 语义             |
| ------ | --------------------------------- | ---------------- |
| `GET`  | `/api/ops/applications`                         | Application 列表         |
| `GET`  | `/api/ops/applications/{applicationId}`         | Application 详情 summary |
| `GET`  | `/api/ops/applications/{applicationId}/services` | Application 服务聚合     |

建议列表返回：

- `application_id`
- `display_name`
- `application_type`
- `compose_project_name`
- `source_type`
- `ownership_mode`
- `workspace_path`
- `runtime_status`
- `service_count`
- `container_counts`
- `drift_status`

其中：

- `runtime_status` 是项目级聚合状态，不是 Docker 原始状态。
- Phase 1 当前聚合值固定为：
  - `running`
  - `degraded`
  - `stopped`
  - `transitioning`
  - `unknown`
- `container_counts` 至少返回：
  - `running`
  - `stopped`
  - `transitioning`
  - `issue`
  - `total`
- 列表页允许把 `service_count + container_counts` 合并成单列资源摘要，但不得删除 API 里的 canonical fields。

## 10.2 导入与校验

| Method | Path                                          | 语义                                              |
| ------ | --------------------------------------------- | ------------------------------------------------- |
| `GET`  | `/api/ops/applications/import/runtime-candidates` | 返回当前 runtime 可见的 Compose import candidates |
| `POST` | `/api/ops/applications/import/runtime-inspect`    | 基于一个 `ready` runtime candidate 执行 inspect   |
| `GET`  | `/api/ops/applications/import/directory-sources`  | 返回 import flow 可用的 directory roots           |
| `GET`  | `/api/ops/applications/import/directories`        | 在一个 allowed root 下分页浏览目录                |
| `POST` | `/api/ops/applications/import/inspect`            | 自动发现并 inspect 一个 selected directory        |
| `POST` | `/api/ops/applications/import/validate`           | 只校验输入与 Compose 解析，不持久化               |
| `POST` | `/api/ops/applications/import`                    | 导入并注册项目                                    |

`runtime-candidates` 返回建议包含：

- `candidate_key`
- `compose_project_name`
- `status`
- `status_reason_codes`
- `importable`
- `runtime_type`
- `runtime_version`
- `workspace_path`
- `workspace_path_source`
- `config_files`
- `service_names`
- `container_counts`
- `warnings`

`runtime-inspect` 返回建议包含：

- `inspection_id`
- `candidate_key`
- `resolved_workspace_path`
- `compose_project_name`
- `compose_project_name_source`
- `display_name_suggested`
- `compose_files`
- `env_files`
- `services`
- `networks`
- `volumes`
- `config_hash`
- `warnings`
- `conflicts`
- `validation_status`

保留的 `directory-sources` 返回：

- `provider`
- `root_id`
- `label`
- `path`
- `managed`

`directories` 返回：

- `provider`
- `root_id`
- `current_path`
- `parent_path`
- `limit`
- `offset`
- `has_more`
- `sort_by`
- `order`
- `directories[]`

保留的 `inspect` 返回建议包含：

- `inspection_id`
- `directory_ref`
- `resolved_workspace_path`
- `compose_project_name`
- `compose_project_name_source`
- `display_name_suggested`
- `compose_files`
- `env_files`
- `services`
- `networks`
- `volumes`
- `config_hash`
- `warnings`
- `conflicts`
- `validation_status`

保留的 legacy `validate` 返回建议包含：

- 自动发现的 compose / env 文件
- 解析到的 `compose_project_name`
- 规范化 preview 摘要
- 服务数
- warning / diagnostics summary
- 冲突信息

`import` 返回建议包含：

- 项目主记录
- 快照摘要
- import 使用的 inspect authority 已被消费，不接受前端重复提交 working directory / compose/env file sets

`directory-sources` / `directories` / `inspect` 当前保留的理由固定为：

- 支撑非主入口 inspect/file-system 复用能力
- future 服务器终端或文件能力复用
- 不能再被误解释为 Phase 1 主 import IA

## 10.2A Phase 2 managed root 与 create contract

`phase-2-batch-1-managed-root-and-create-contracts` 只落 authority owner，不落真实 file write create flow。

新增 canonical contract owner：

| Method | Path                                | 语义                                                                        |
| ------ | ----------------------------------- | --------------------------------------------------------------------------- |
| `GET`  | `/api/ops/applications/managed-root`    | 返回 managed create 的 system-config authority、ownership mode 与 readiness |
| `POST` | `/api/ops/applications/create/validate` | 只校验 managed create 输入、目标目录推导与 bounded authority，不写文件      |
| `POST` | `/api/ops/applications/create`          | 在 managed root 下写入 compose/env 文件并注册 managed project               |

managed root authority 约束：

- canonical config key 固定为 `ops.application.root_directory`
- config owner 固定为 `server/modules/project/**`
- 产品显示名为 `Application Root Directory` / `应用根目录`，默认值为 `/opt/graft/apps`
- root directory 必须是 server container 内可见的绝对路径；它的宿主机 bind mount、预创建和运行用户写权限属于部署配置，不得由 System Config 伪造
- empty string 是显式 override，表示禁用 managed creation；不得把它或“未配置”降级成 request payload fallback。此功能未发版，不保留 `ops.project.managed.root_directory` legacy key、alias 或双读。
- Phase 2 真实 create 只能在该 managed root 下创建 `managed-root-dedicated` 目录

Blank create request 建议至少包含：

- `display_name`
- `runtime_target_id`
- `workspace_key?`
- `workspace_entries[]`
  - `path`
  - `node_type: file | directory`
  - 文件条目的 UTF-8 `content`
- `compose_file_path`

`workspace_entries` 表达任意安全相对路径的文本文件及空目录，不按文件名、扩展名或 `file_kind` 设置创建白名单。`compose_file_path` 只标识本次创建必须存在且可解析的主 Compose 文件，不决定其他 workspace 成员资格。服务端必须拒绝绝对路径、空路径、`..` 路径逃逸、重复或冲突条目、符号链接绕过和不受支持的二进制内容；不得以现有前端文件高亮或目录隐藏配置作为创建准入规则。

Blank 向导的默认草稿由 project module 返回，前端不得硬编码模板文件内容。System Config `ops.project.blank.prefill_default_template` 的产品名称为“Blank 创建预填默认模板”，默认 `true`，通过现有 `configregistry -> SystemConfigResolver` authority 链和 `runtime-hot` 语义生效：

- 开启时，Blank 草稿完整复制 `templates/default` 的当前内容。
- 关闭时，Blank 草稿仅提供空 `.env` 与空 `compose.yaml`；在审核及创建前必须填充并通过 `compose_file_path` 的 Compose 解析。
- Template source 始终物化用户所选模板目录，且不受该开关影响。

服务端从 `workspace_key` 生成唯一的单层 Workspace Path：`Application Root Directory + workspace_key`。创建表单展示可编辑的 Workspace Key 控制项，初始值为 Graft 按显示名提议的可用 key；默认 key 冲突时服务端自动附加 `-2`、`-3` 等后缀。用户若显式修改 key，只能填写安全 slug，冲突时返回本地化错误与建议值。不得接受 `relative_project_directory`、绝对路径、路径分隔符或 `..`；用户界面不展示完整路径、受管根目录、权限或 Compose runtime identity。

受管工作区是 Application 的文件载体，不按 Docker、Podman、Compose 或 Kubernetes 分层。Runtime 切换或未来 Deployment Type 演进不移动 Workspace；runtime 自身的存储仍由 Provider 管理。数据库 registry 是元数据真相，实际 Workspace 是文件内容真相；本切片不引入 `graft.yaml`。

采用受管默认目录符合主流自托管产品的共同策略：集中工作目录使权限、备份、迁移、冲突检测和受控销毁的边界可审计，同时仍为高级操作者保留受约束的子目录选择。公开部署材料可作为实现前复核依据：Portainer 的 Stack 文档将 Compose/Swarm/Kubernetes 按独立部署方式管理；[Dockge](https://github.com/louislam/dockge) 公开 `DOCKGE_STACKS_DIR` 工作目录配置；[Coolify](https://coolify.io/docs/installation) 安装到平台受管数据目录；[Dokploy](https://docs.dokploy.com/docs/core/installation) 以平台安装与数据目录管理其工作负载。Graft 不依赖这些产品的精确路径或内部实现，只采用“平台默认受管根目录、用户可在安全边界内覆盖”的产品原则。

本批次不引入下游兼容字段，也不得用 import contract 冒充 create contract。

## 10.3 配置

Configuration workspace 的 authority 需要拆成两层：

- Project registry authority
  - 继续保存 lifecycle 执行所需的 `compose_files`、`env_files`、`workspace_path`、`compose_project_name`
- Workspace authority
  - File tree、editor open/read、dirty state、save 与 preview diff 全部围绕同一套 workspace file model 运作
  - 同一文件树编辑器同时服务创建向导中的本地 workspace draft 和已创建项目的持久化 workspace；差异仅在数据源与保存时机，不得形成两套文件树交互或文件类型规则
  - Web 侧唯一状态真值是 `ProjectWorkspaceStore` 的规范化节点模型：`nodesByKey`、`rootKeys` 与每个目录的 `childKeys` 共同表达层级；`expandedKeys`、`selectedKey`、`openedTabs`、`activeFileKey`、文件内容与 dirty 状态同属该 Store。
  - `WorkspaceTree` 只能消费 Store 生成的可见节点行，不得从 path、flat list、`parent_path` 或组件局部 state 重建父子关系；Monaco 只能消费 `activeFile`，不得修改 Tree。目录展开、深层文件打开、创建、重命名、删除与 reload 都由 Store action 维护祖先展开、选择和打开 buffer。
  - Create 与 Configuration 只在 `WorkspaceEditor` 外提供各自 toolbar 和数据源适配；编辑器本身不得按页面类型分支。
  - 左侧文件树、编辑器文件打开、保存能力全部来自 `workspace_path` 的真实目录浏览/读写接口
  - `compose_project_files`、`lifecycle_configuration`、`compose_files` 与固定文件名都不能决定工作台文件成员资格
  - Compose metadata 只允许 enrich workspace entry 的 `file_kind` / tooltip / lifecycle overlay，不拥有文件内容 authority

补充原则：

- Workspace manifest 是 runtime projection，不是新的持久化对象。
- Backend 不维护 Preview Diff 算法或规范化 hash authority，只返回原始磁盘文件内容与文件可读/可编辑状态。
- Frontend 自己维护 buffer、dirty state 与 Monaco Diff。

建议 API 拆分：

| Method | Path                                           | 语义                                                  |
| ------ | ---------------------------------------------- | ----------------------------------------------------- |
| `GET`  | `/api/ops/applications/{applicationId}/configuration`         | 项目级 configuration summary 与 workspace 状态摘要    |
| `GET`  | `/api/ops/applications/{applicationId}/configuration/preview` | 规范化 Compose preview                                |
| `GET`  | `/api/ops/applications/{applicationId}/files`                 | 基于 `workspace_path` 懒加载浏览真实目录树         |
| `GET`  | `/api/ops/applications/{applicationId}/files/content`         | 基于相对路径读取单文件内容                            |
| `PUT`  | `/api/ops/applications/{applicationId}/files/content`         | 基于相对路径保存单文件内容，只写回 `workspace_path` |
| `POST` | `/api/ops/applications/{applicationId}/files`                 | 在受控工作区创建文件或空目录                          |
| `PATCH` | `/api/ops/applications/{applicationId}/files`                | 重命名文件或目录                                      |
| `DELETE` | `/api/ops/applications/{applicationId}/files`               | 删除文件或目录；递归删除必须显式声明                  |

`configuration` 返回建议包含：

- lifecycle authority summary
  - `workspace_path`
  - `compose_files`
  - `env_files`
  - `compose_project_name`
- ownership summary
- drift summary
- last refresh summary
- workspace status summary
  - `show_hidden_supported`
  - `hidden_directories_config_key`
  - `has_unsaved_changes`（frontend session state 的回显位可选）

`files` 返回建议包含：

- `root_path`
- `current_path`
- `parent_path`
- `items[]`
  - `name`
  - `relative_path`
  - `node_type`
  - `file_kind`
  - `readable`
  - `editable`
  - `language_hint`
  - `size_bytes`
  - `hidden_by_default`
  - `has_children`

约束：

- 目录树必须支持懒加载与任意层级，不做一次性全量递归。
- 默认隐藏重目录，隐藏策略来自 `ops.project.workspace.hidden_directories`；前端只消费接口返回与 `show hidden` 开关能力。
- `compose_project_files` 继续服务 lifecycle authority，但不能再充当 workspace tree authority。
- 任意可在 workspace 中打开并编辑的文本文件，都天然属于 Preview Diff 范围；不根据后缀额外维护 diff 白名单。
- 已创建项目的创建、重命名和删除操作都以 `workspace_path` 为唯一可写根目录，并复用读取/保存接口的项目编辑权限与相对路径边界校验。目录删除只有请求显式 `recursive=true` 时才允许递归；UI 必须在非空目录删除前二次确认并显示受影响条目范围。
- Workspace tree 的根节点、目录节点和文件节点均提供右键菜单：新建文件、新建文件夹、重命名、删除。文件节点的新建目标为其父目录；根和目录节点的新建目标为当前目录。前端操作后同步树、展开状态、打开标签、活动文件及未保存 buffer。

`files/content` 返回建议包含：

- `relative_path`
- `file_kind`
- `readable`
- `editable`
- `language_hint`
- `encoding`
- `content`
- `size_bytes`

保存语义固定为：

- `PUT /files/content` 只把当前文件内容写回 `workspace_path`
- 保存不隐含“立即生效”“自动 refresh”或“自动 deploy”
- 编辑器按文件驱动，允许同时打开多个文件；保存作用域默认只限当前文件

## 10.4 生命周期动作

| Method | Path                                | 语义                                                    |
| ------ | ----------------------------------- | ------------------------------------------------------- |
| `POST` | `/api/ops/applications/{applicationId}/refresh`    | 刷新静态配置与聚合视图                                  |
| `POST` | `/api/ops/applications/{applicationId}/up`         | 执行 compose up                                         |
| `POST` | `/api/ops/applications/{applicationId}/stop`       | 执行 compose stop，仅停止运行中的服务与容器             |
| `POST` | `/api/ops/applications/{applicationId}/restart`    | 执行 compose restart                                    |
| `POST` | `/api/ops/applications/{applicationId}/validate`   | 基于当前已保存磁盘状态执行项目级配置校验                |
| `POST` | `/api/ops/applications/{applicationId}/redeploy`   | 按已保存 lifecycle configuration 提交项目重新部署任务  |
| `POST` | `/api/ops/applications/{applicationId}/unregister` | 只删注册记录                                            |
| `POST` | `/api/ops/applications/{applicationId}/destroy`    | 执行 compose down 并进入高危销毁收尾；受 ownership 保护 |

`destroy` 请求建议显式字段：

- `remove_named_volumes`
- `delete_workspace_path`
- `confirm_compose_project_name`

并要求后端返回：

- 哪些步骤已执行
- 哪些步骤被 ownership guard 拒绝
- 最终是否已注销

项目级操作语义固定为：

- `validate`、`redeploy` 读取的都是当前已保存到磁盘的项目状态，而不是前端内存草稿
- 若前端存在未保存文件，必须先提示用户是否保存
- Preview Diff 固定由 frontend buffer 对磁盘内容做本地 Monaco Diff，不提供后端 `diff` API
- `redeploy` 的未保存提示固定为“检测到未保存的修改，是否先保存？”，并提供 `保存`、`继续使用磁盘版本重新部署`、`取消`
- 其中“保存”只把当前未保存内容写回 `workspace_path`，不隐含立即生效，也不自动提交 `redeploy` 任务
- `redeploy` 始终是独立的显式项目级动作；保存不自动提交任务，必须由用户显式触发 `redeploy`

## 10.5 Phase 1 明确不提供的 API

- 不提供 `/api/ops/applications/{applicationId}/events`
- 不提供 `/api/ops/applications/{applicationId}/files`
- 不提供 `/api/ops/applications/{applicationId}/files/content`
- 不提供 `PUT` 配置编辑保存接口
- 不提供项目级 `diff` / `validate` / `deploy` API

## 10.6 Batch 1 authority 落地说明

`phase-1-batch-1-project-contract-and-data-model` 已把以下 authority owner 固定到仓库运行面：

- OpenAPI contract owner：`openapi/**`
  - route space 固定为 `/api/ops/applications/**`
  - Phase 1 只读 Configuration 固定拆为 metadata、preview、single-file content
  - 明确保留 lifecycle routes 的 contract owner，但不在本 batch 落 runtime handler
- Project module contract owner：`server/modules/project/contract/**`
  - canonical route fragments
  - source / ownership / drift / refresh / file kind 等 typed contracts
- Project module data-model owner：`server/modules/project/model.go`
  - 只定义 registry、file list、snapshot 三类 module-owned persistence model
  - 不引入容器 logs / events / stats / inspect 等 runtime 字段
- Project module migration owner：`server/modules/project/migrations/**`
  - `applications`
  - `compose_project_files`
  - `compose_project_snapshots`
- Narrow shared boundary owner：`server/internal/moduleapi/container_project.go`
  - 仅为后续 `Services` 聚合预留 project->container 的最小稳定只读边界
  - 不暴露 container detail、logs、events、stats、shell、inspect 私有实现

本批次仍明确不做：

- `server/modules/project` runtime wiring、handler、repository、service
- `web/src/modules/project/**`
- backend project logs/events aggregation
- managed create / editor / diff / deploy / validate UI

## 10.6A Batch 2.1 authority 落地说明

`phase-2-batch-1-managed-root-and-create-contracts` 已把以下 authority owner 固定到仓库运行面：

- OpenAPI contract owner：`openapi/**`
  - 新增 `/api/ops/applications/managed-root`
  - 新增 `/api/ops/applications/create/validate`
  - 新增 `/api/ops/applications/create`
- Project module contract owner：`server/modules/project/contract/**`
  - 新增 managed-root status typed contract
  - 新增 managed-create permission contract
  - 新增 managed-root config key contract
  - 新增 create route fragments
- Project module config-definition owner：`server/modules/project/config.go`
  - `ops.application.root_directory` 成为 managed create 的 canonical system-config authority，显示为 `Application Root Directory` / `应用根目录`，默认 `/opt/graft/apps`
  - empty string 是禁用 managed creation 的显式 override，而不是隐式回退到仓库路径或用户 home

本批次仍明确不做：

- web managed create UI / editors
- diff / validate / deploy flow

## 10.6B Batch 2.2 authority 落地说明

`phase-2-batch-2-server-managed-create-and-file-write-path` 已把以下 authority owner 固定到仓库运行面：

- OpenAPI contract owner：`openapi/**`
  - `POST /api/ops/applications/create` 不再复用 validate request
  - create request 拥有独立 compose/env file content payload
  - create response 改为同步创建结果，显式返回 `application_id`、目标文件路径和 snapshot summary
- Project module execution owner：`server/modules/project/**`
  - 在 managed root 下创建 bounded working directory
  - 写入 compose file 与可选 env file
  - 复用 project-owned parse + snapshot + repository import path 持久化 managed project
  - 若 registry 持久化失败，回滚本轮新建的 managed directory/file bootstrap，避免留下无主目录

本批次仍明确不做：

- web managed create form / editor 交互
- diff / validate / deploy flow
- remote host / git / template source

## 10.7 Batch 4 authority 落地说明

`phase-1-batch-4-web-project-list-detail-and-readonly-configuration` 已把以下前端 authority owner 固定到仓库运行面：

- Frontend module owner：`web/src/modules/project/**`
  - module registration：`index.ts`、`bootstrap-routes.ts`
  - route contract consumer：`contract/bootstrap.ts`、`contract/paths.ts`
  - typed API consumer：`api/project.ts`、`types/project.ts`
  - locale owner：`locales/en-US.json`、`locales/zh-CN.json`
  - page owner：`pages/list/index.vue`、`pages/detail/index.vue`
  - module-local shared UI helpers：`shared/display.ts`、`shared/navigation.ts`
- List / Detail IA owner：
  - `list` 页面固定承载 project registry list、筛选、summary、危险动作入口与 detail tab 导航
  - `detail` 页面固定承载 `Overview`、`Services`、`Configuration`、`Lifecycle`、`Activity` 五个页签
- Authority guard 已落地：
  - `Overview` 只承载 summary，不引入 runtime dashboard 指标或 timeline
  - `Services` 只消费静态定义与 container member/count 聚合，并回跳现有 Container Detail
  - `Configuration` 保持 metadata、preview、single-file content 三段只读消费
  - `Activity` 继续只做前端 fan-out，复用现有 container logs/events API
  - 未新增 backend project logs/events aggregation、managed create/editor/diff/redeploy/validate UI

## 11. UI 信息架构

## 11.1 推荐层级

```text
Projects
  -> Project Detail
     -> Overview
     -> Services
     -> Configuration
     -> Lifecycle
     -> Activity
```

## 11.2 是否需要 Overview

推荐保留 `Overview`，但必须保持极简 summary，而不是第二套 Dashboard。

`Overview` 应承载：

- Project Identity
- Runtime Status
- Source：`Managed | Imported`
- Ownership
- Working Directory
- Last Refresh
- Drift 状态
- Actions
- Services Count
- Running / Stopped / Transitioning / Issue Count 的聚合摘要

不应承载：

- CPU
- Memory
- Runtime Charts
- Recent Logs
- Events Timeline
- Metrics

## 11.3 Tradeoff

保留 `Overview` 的好处：

- 项目身份、来源、路径、ownership 与高危动作有稳定落点
- 不必把这些信息散落在 Services / Configuration / Lifecycle / Activity 四个页签
- 与“项目只是聚合入口”的定位一致

保留 `Overview` 的代价：

- 多一个页签
- 需要额外的 summary contract

结论：

- 应保留
- 但必须严格限制为 summary，而不是 dashboard

## 11.4 各页签职责

### Overview

- 摘要与动作

### Services

- 静态服务定义 + 运行态容器计数
- 点击 Service 后进入现有 Container Detail

### Configuration

- 基于项目根目录真实目录树的文件工作台
- 多文件 Monaco 编辑
- Compose Preview
- 项目级 Diff / Validate / Deploy 入口
- Lifecycle authority 摘要

### Lifecycle

- 显示 lifecycle review status
- 显示 working directory、ordered compose files、canonical project name
- 编辑标准 Compose lifecycle 选项：profiles、down/pull/build/force-recreate/wait/prune
- 实时展示 generated command preview
- custom script 只保留 future strategy boundary，不在当前 batch 实现

### Activity

- 前端 fan-out 聚合容器日志与事件
- 不新增项目级运行时 authority

## 12. Project 与 Container 的职责关系

## 12.1 Project owns what

- Registry
- Source
- Ownership
- Working Directory
- Compose Files
- Env Files
- Canonical Name
- Display Name
- Snapshot
- Drift
- Lifecycle Actions
- Lifecycle Configuration
- Services Aggregation

## 12.2 Container owns what

- Container Runtime State
- Logs
- Events
- Stats
- Shell
- Inspect
- Mounts
- Networks
- Runtime topic streaming

## 12.3 避免重复运行时信息的规则

- 项目服务页只显示聚合计数与入口，不复制容器详情原始字段。
- 项目 Activity 页只做前端 fan-out，不在项目后端持久化聚合结果。
- 项目 Overview 只显示计数与状态，不显示 CPU / Memory / Timeline。
- 任一需要单容器事实的入口都回跳现有 Container Detail。

## 13. Drift 设计

Project Drift 是该能力的重要价值之一。

## 13.1 Phase 1 最小模型

Phase 1 至少记录：

- `compose_project_snapshots.config_hash`
- `last_observed_config_hash`
- `drift_status`
- `last_drift_checked_at`

## 13.2 可见提示

Overview 至少要能显示：

- `Configuration Changed`
- `Files Missing`
- `Refresh Failed`

## 13.3 触发时机

Phase 1 不要求后台 watcher。

可接受的触发路径：

- 列表 / 详情请求时做轻量 hash 检查
- 项目列表不提供手动刷新入口；列表新鲜度由 HTTP seed + list realtime 保持
- 导入校验

## 14. 安全、所有权与风险

## 14.1 路径与符号链接

风险：

- symlink 指向受管根目录外
- 目录移动后 registry 残留
- 相对路径与 `project_directory` 解析不一致

要求：

- 存储 declared path 与 resolved absolute path
- 删除目录前必须基于 resolved path 重新校验 ownership
- 对危险文件系统动作拒绝路径逃逸

## 14.2 环境插值与 include

风险：

- `.env` 缺失或内容变化导致解析差异
- `include` / 多文件覆盖带来快照与运行态差异

Phase 1 处理：

- 导入与刷新都必须显式记录 env file 集合
- contract 允许有序多文件
- 若解析器或当前实现无法安全支持某种扩展语法，宁可显式报错，不做静默部分支持

## 14.3 Project rename

风险：

- 用户误把 Display Name 当成 Runtime Project Name
- 直接改 Canonical Name 会让现有容器归属断裂

决策：

- Phase 1 只允许修改 `display_name`
- `compose_project_name` 默认只读
- 未来如允许修改，必须作为独立高危流程处理

## 14.4 目录移动

风险：

- working directory 被外部移动或删除后，项目记录失效

处理：

- refresh / drift check 发现路径丢失时置为 `missing`
- 不自动改写 registry

## 14.5 权限与 CLI 风险

风险：

- 无法读取配置文件
- Docker socket 无权限
- `docker compose` 不可用
- 命令拼接注入

要求：

- 一律使用参数数组执行 CLI，不拼 shell 字符串
- 所有失败都返回结构化错误码与 message key

## 14.6 Destroy 风险

风险：

- 误删外部目录
- 误删共享 volumes

要求：

- `Unregister` 永远是安全默认
- `Destroy` 必须显式确认
- 删除目录只允许在 ownership proof 成立时执行
- volume 删除必须独立校验共享引用

## 15. 未来扩展评估

| 方向                                 | 是否兼容当前模型 | 设计说明                                                     |
| ------------------------------------ | ---------------- | ------------------------------------------------------------ |
| Git-based Projects                   | `yes`            | 在 `source_type` 上扩展 `git`，并追加 source metadata        |
| Templates                            | `yes`            | Template 是 Application Root `templates/<key>` 的受管输入源 |
| Directory Scan                       | `yes`            | 扫描只产出 candidates，不直接注册                            |
| Auto Discovery                       | `yes`            | 后台发现只更新 candidate / drift，不改变 runtime authority   |
| Multiple Compose Files               | `yes`            | `compose_project_files` 的 `order_index` 已为有序 `-f` 预留  |
| Compose Override                     | `yes`            | 通过 `role=override` 与有序文件列表支持                      |
| Environment Files                    | `yes`            | `kind=env` + file list 可扩展多个 env file                   |
| Remote Host                          | `partial`        | 需通过 Runtime Target 与 Provider capability 扩展，Application 不增加 host 字段 |
| Project Activity backend aggregation | `future yes`     | 需要单独定义 observability authority，Phase 1 不做           |

### 15.1 Creation Pipeline

所有来源在解析出真实 workspace 后必须进入同一条 project creation pipeline：确认 lifecycle configuration、写入 project aggregate 与来源元数据、生成 compose snapshot，并仅做只读 runtime observation。该 pipeline 不执行 `docker compose up`、pull 或 build。

受支持的创建页面可以在最终 Review 中提供默认关闭、且受 `ops.project.deploy` 权限约束的“创建后部署”选项。该选项必须在项目已成功注册为 `Ready` 后，单独调用既有 Deploy action；Deploy 失败不得回滚创建结果，也不得改变 creation pipeline 的无运行时副作用约束。

- Managed source 负责受 managed root 约束的文本 workspace materialization 与仅限本请求新建文件/目录的回滚。
- Import source 负责 runtime candidate、inspection TTL 与文件 hash freshness，并以 adopt 模式进入 pipeline，不改写被导入目录。
- `compose_project_files` 继续只登记 Compose/Env 解析输入；完整 workspace 以实际目录为唯一内容真相，不能把任意文本文件伪装成 Compose inventory。
- `source_metadata_json` 持久化来源专属、无密钥 provenance；Template adapter 从 Application Root 模板目录发现并物化 workspace，未来来源 adapter 也只能在解析/物化 workspace 后调用同一 pipeline。

## 16. 分阶段实施路线

后续路线图建议按 `Management`、`Observability`、`Configuration` 三类能力组织，而不是按 `Read/Write` 组织。

## 16.1 Phase 0

- 设计 authority 与 topic recovery 持久化
- 明确 `Project != Runtime`
- 明确 `Container` 是 runtime authority

## 16.2 Phase 1

Management：

- Project model
- Import Existing Project
- Project Registry
- Refresh
- Up / Stop / Restart
- Unregister
- Destroy with ownership protection

Observability：

- Overview summary
- Services aggregation
- Activity frontend fan-out aggregation

Configuration：

- Lifecycle authority metadata
- Real file-tree browse/read/write API
- Read-only / editable file classification
- Normalized preview
- Download

Phase 1 明确不包含：

- Managed Create
- Configuration editor
- Diff
- Validate UI
- Deploy
- Project logs/events backend aggregation
- Remote host

## 16.3 Phase 2

Management：

- Managed Project Create
- Managed root workflow

Observability：

- 更细粒度 project status / diagnostics

Configuration：

- 文件工作台
- 多文件编辑
- Diff
- Validate
- Deploy

## 16.3A Batch 2.4 authority 落地说明

`phase-2-batch-4-diff-validate-and-deploy-flow` 把以下 authority owner 固定到仓库运行面：

- OpenAPI contract owner：`openapi/**`
  - 新增 `GET /api/ops/applications/{applicationId}/files`
  - 新增 `GET /api/ops/applications/{applicationId}/files/content`
  - 新增 `PUT /api/ops/applications/{applicationId}/files/content`
  - 新增 `POST /api/ops/applications/{applicationId}/files`、`PATCH /api/ops/applications/{applicationId}/files`、`DELETE /api/ops/applications/{applicationId}/files`，分别承担创建、重命名和显式递归删除
  - 新增 `POST /api/ops/applications/{applicationId}/validate`
  - 新增 `PUT /api/ops/applications/{applicationId}/lifecycle-configuration`
  - 移除 `POST /api/ops/applications/{applicationId}/update-deploy` 作为一等 lifecycle action
- Project module execution owner：`server/modules/project/**`
  - Configuration workspace 左侧文件树 authority 改为 `workspace_path` 的真实目录扫描，不再从 tracked files 推断
  - Workspace file membership / content authority 全部收口到 `workspace_path` 的 runtime browse/read/write service
  - `compose_project_files` 降级为 compose/env 元数据 overlay，不再承载 workspace state 或 Preview Diff authority
  - `files` / `files/content` 及文件树变更接口使用 path-based browse/read/write contract，并统一做相对路径与根目录边界约束；不以文件扩展名、语言高亮或隐藏目录配置限制创建
  - validate 只针对当前已保存磁盘状态做静态解析，不消费前端未保存草稿
  - 本地项目统一保存 lifecycle configuration：managed 默认 `confirmed`；运行时导入必须在导入向导内审核服务端提供的默认配置，并与项目注册一起保存为 `confirmed`
  - 保存只允许写回 `workspace_path` 的可编辑文件；保存本身不触发 refresh 或 redeploy
  - Configuration Workspace 的重新部署只读取当前已保存磁盘状态，并提交 project-owned lifecycle configuration 的 `redeploy` Task
  - redeploy 成为统一 runtime deploy-style lifecycle action；pull/down/prune 等语义都收口到 lifecycle configuration
  - 不新增 project runtime persistence、project logs/events aggregation 或 project-owned container detail
- Frontend module owner：`web/src/modules/project/**`
  - `detail -> configuration` 承载基于真实目录树的文件工作台、多文件编辑、Preview Diff、validate 与 redeploy 入口
  - Monaco Diff 直接消费 workspace current content 与 frontend buffer proposed content，不再调用 backend diff API
  - `detail -> lifecycle` 承载 lifecycle configuration 编辑、review 提示和 generated command preview
  - 仍保持 `detail` 页属于 `list-form-detail` page type，不把 Overview 变成 runtime dashboard

## 16.3B Phase 2 archive-readiness check

`phase-2-batch-5-phase-2-validation-drift-guard-and-governance-sync` 完成后，Phase 2 以同一 topic 内的 bounded batches 达到可审计验收状态：

- managed root create、Compose/Env editor、Preview Diff、validate、redeploy 路径均已落地并通过完整验证链
- `Project` 继续只拥有 project registry、workspace browse/read/save、validate 与 lifecycle/redeploy orchestration
- 不新增 project runtime persistence、project logs/events aggregation 或 project-owned container detail
- `Container` 继续保持 runtime authority
- Topic 不进入 `archive-ready`，因为 Phase 3 仍需按更小的 bounded batches 继续推进

## 16.4 Phase 3

Management：

- Git Project
- Directory Scan
- Auto Discovery
- Remote Host

Observability：

- Project Activity backend aggregation

Configuration：

- Multi-file override UX 强化
- Git / template source metadata 深化

## 16.4A Phase 3 rebatching建议

为保持 `topic-completion-loop` 可继续执行，Phase 3 不应再保留单个大阶段占位，建议至少拆成以下安全 batches：

- `phase-3-batch-1-git-template-source-contract-and-boundary`
  - 只固定 git/template source metadata、route/permission/menu contract 与 authority boundary
  - 不落 directory scan、auto discovery、remote host 或 backend activity aggregation
- `phase-3-batch-2-directory-scan-and-auto-discovery-candidates`
  - 只落 scan/discovery candidate model、候选结果 contract 与 bounded authority
  - candidate 只产出发现结果，不直接注册 project，不改变 runtime authority
- `phase-3-batch-3-remote-host-boundary-and-activity-authority`
  - 先收敛 remote host 扩展边界与 project activity backend aggregation authority
  - 未完成该批之前，不应把 project activity backend aggregation 当作 implementation-ready scope

## 16.4D Phase 3 Batch 3 authority 落地说明

`phase-3-batch-3-remote-host-boundary-and-activity-authority` 只收敛 remote-host 扩展边界与 project activity authority，不直接实现 remote execution 或 backend aggregation：

- OpenAPI contract owner：`openapi/**`
  - remote host 只能通过 Runtime Target 与 Provider capability 规划，不增加 Application host 字段
  - `project list/detail` 固定 `activity_authority` 字段，明确当前是 `frontend-fanout` 还是 `backend-planned`
  - `source_metadata` 允许的新增 planned 字段仅包括：
    - `remote_host_key`
    - `remote_compose_path`
    - `activity_authority`
    - `activity_rollup_scope`
- Project module owner：`server/modules/project/**`
  - `remote-host` 只作为 source selector / route / permission / metadata owner 进入 source catalog
  - 不新增 remote host credential persistence、remote command execution、backend project logs/events aggregation 端点或 project-level runtime cache
  - 当前本机 `local` project 的 `activity_authority` 仍固定为 `frontend-fanout`
  - future `remote` project 或 backend aggregation 只保留 `backend-planned` authority 标识，不视为 implementation-ready
- Web module owner：`web/src/modules/project/**`
  - `/applications/create/remote-host` 只保留 planned boundary 页面
  - project detail 明确展示当前 `activity authority`
  - 当前 Activity tab 继续只做前端 fan-out；若 authority 为 `backend-planned`，UI 只提示 future boundary，不伪造后端数据

当前 batch 的 hard boundary：

- 不新增 remote host 持久化或连接测试
- 不执行 remote `docker compose`
- 不新增 backend project logs/events aggregation endpoint
- 允许新增 project list summary realtime topic，用于项目列表摘要自动更新；不在本批扩展 remote host 或后端 activity aggregation realtime
- 不把 discovery candidate 扩大成 auto-registration 或 unmanaged runtime ownership

## 16.4B Phase 3 Batch 1 authority 落地说明

`phase-3-batch-1-git-template-source-contract-and-boundary` 收敛 Git source 的 entry authority，并复核已落地 Template source 与同一 creation pipeline 的边界：

- OpenAPI contract owner：`openapi/**`
  - 新增 `GET /api/ops/applications/sources` 作为 source catalog authority
  - `project list/detail`、`managed root`、`managed create validate/create` 增加最小 `source_metadata` / `source_type` 字段
- Project module owner：`server/modules/project/**`
  - source catalog 只声明 `managed | git | template` entrypoint、route path、permission、metadata field 列表与当前状态
  - managed source 继续沿用现有执行逻辑，但路由边界收口到 `/create/managed`
  - Git 仅在隔离临时目录 clone/checkout 无凭据来源；Template 只从 Application Root `templates/<key>` 读取、发现并完整物化安全文本 workspace，不使用模块内置内容。二者都不扩展到目录扫描、remote host 或 backend activity aggregation。
- Web module owner：`web/src/modules/project/**`
  - `/applications/create` 固定为 source selector
  - `/applications/create/managed` 承接现有 managed create 页面
  - `/applications/create/git` 与 `/applications/create/template` 承接 source adapter 创建页；两者都必须在真实 workspace 解析后进入同一 creation pipeline，且不得自动 deploy

IA guardrail:

- `source selector` 只是 Phase 3 boundary inspection surface，不得替代 Phase 1 `Import Existing Project` 主入口。
- `managed create` 是 Phase 2 的真实入口，应继续由 `/applications/create/managed` 承载。
- 如果列表页或空态只能给一个主按钮，默认必须先给 `Import Existing Project`，不能默认把用户送进 planned boundary。

当前批次允许的 `source_metadata` 范围：

- managed
  - `managed_root_key`
  - `managed_relative_directory`
  - `managed_compose_file_name`
  - `managed_env_file_name`
- git
  - `git_repository_url`
  - `git_reference`
  - `git_compose_subpath`
- template
  - `template_key`
  - `template_version`
  - `template_instance_name`

禁止把这些 metadata 提前扩大成：

- project-level runtime persistence
- git clone state / sync state
- template materialization job state
- backend project logs/events aggregation state

## 16.4C Phase 3 Batch 2 authority 落地说明

`phase-3-batch-2-directory-scan-and-auto-discovery-candidates` 只收敛 discovery candidate authority，不引入 project registry 自动写入或后台发现任务：

- OpenAPI contract owner：`openapi/**`
  - 新增 `GET /api/ops/applications/discovery-candidates`
  - 固定 candidate 只读 contract：`candidate_key`、`candidate_kind`、`source_type`、`workspace_path`、`compose/env file list`、`declared_service_names`、`config_hash`、`warnings`、`conflicts`、`recommended_action`
- Project module owner：`server/modules/project/**`
  - 以当前 `managed root` 作为 bounded local scan authority
  - 本批只做本机受限目录扫描和 auto-discovery preview 结果投影
  - candidate 只作为 discovery/preview surface，不写 registry、不自动调用 import、不产生 project-level runtime persistence
  - candidate 冲突只复用现有 registry conflict 规则进行 `review/import` 建议，不新增 compatibility layer
- Web module owner：`web/src/modules/project/**`
  - 在 `/applications/create` source selector 下新增 hidden discovery preview surface
  - UI 只展示 authority root、候选状态、建议动作和冲突/文件预览，不直接注册项目

当前 batch 固定的 candidate 字段语义：

- `candidate_kind`
  - `directory-scan`
  - `auto-discovery`
- `status`
  - `ready`
  - `conflict`
  - `skipped`
- `recommended_action`
  - `import`
  - `review`

当前 batch 的 hard boundary：

- 不自动注册 project
- 不写数据库 candidate 持久化
- 不新增后台 auto discovery scheduler / watcher
- 不扩展到 remote host
- 不引入 backend project activity aggregation

## 17. 最终结论

`Compose Project` 在 `Graft` 中的定位应固定为：

- Runtime Workspace
- Registry + Configuration + Lifecycle + Aggregation

而不是：

- 第二套容器运行时
- 第二套容器详情
- Compose IDE
- Project Dashboard

必须长期坚持：

- `Container` 是 Runtime Authority
- `Project` 聚合 Runtime，而不是复制 Runtime
- `Snapshot` 是最近一次成功解析结果，而不是运行时缓存
- `Overview` 是 Summary，而不是 Dashboard

## 18. 当前来源范围与扩展口

当前公开且可执行的 Deployment Type 只有 `compose`，它基于 Compose Specification，不是 Docker Provider 的同义词。Deployment Type picker 只为它提供可点击入口；Swarm、Kubernetes 与 Nomad 在各自真实 Target、Provider capability、lifecycle 和 Source adapter 落地前保持 disabled placeholder。Podman 不是 Deployment Type；它在未来作为 Compose Runtime Target Provider 接入。

当前公开且可执行的 Compose Application Source 只有：

- `Managed`：编辑器生成 Workspace 并在 Managed Root 内 materialize。
- `Template`：Application Root `templates/<key>` 模板目录生成 Workspace。
- `Import Existing`：运行时候选经检查后以 adopt 模式进入同一创建管线。

MVP 必须由 canonical OpenAPI 定义 Deployment Type 与 Runtime Target capability 的最小选择 contract；创建请求不再接受 canonical name 或相对目录。`GET /api/ops/applications/creation-methods` 只列出当前已实现的 `blank`、`template` 与 `import`，并只返回稳定的可用性与阻塞原因。UI 统一入口是 `/applications/create`，依次进入 deployment、target、source 与向导。Git 可在 Source 页面仅作为禁用卡片展示，并通过本地化 tooltip 显示“暂不支持”；它与 Remote Host、ZIP、GitHub Template 一样不得预先暴露 API、路由、菜单、创建方式枚举、持久化来源类型或占位页面。

未来 Deployment Type 或 Runtime Target Provider 必须先由 canonical OpenAPI 定义 kind、capability、可用性和 stable reason code，再同时交付生命周期与 Source adapter；不得用占位卡片提前建立 wire model。未来创建方式只能在其真实向导、OpenAPI contract 和创建方式目录同时实现后公开，并且必须遵循 `Deployment Type -> Runtime Target -> Application Source -> Workspace -> CreationCommand -> lifecycle/review -> aggregate/snapshot -> read-only runtime sync`。创建方式负责获取或构建 Workspace；共享 CreationCommand 负责配置确认、注册和快照，不能复制项目创建逻辑。
