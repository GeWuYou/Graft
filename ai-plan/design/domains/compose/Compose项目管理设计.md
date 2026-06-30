# Compose项目管理设计

本文档定义 `Graft` 对 Docker Compose Project 的产品 IA、模块边界、数据模型、导入语义、API 方向、风险边界与阶段路线。

该能力的核心定位必须保持稳定：

- `Project` 是 Compose Project 的管理与聚合层，不是新的 Runtime。
- `Container` 始终是 Runtime Authority。
- `Project` 只负责项目注册、配置解析、生命周期管理和聚合入口。
- `Project` 不得复制、替代或持久化容器运行时真相。

## 1. 启动治理与 Authority

- 任务分类：`cross-boundary`
- governance source：root `AGENTS.md`
- 设计 authority：本文档
- 后端 authority：future `server/modules/project/**`
- 前端 authority：future `web/src/modules/project/**`
- OpenAPI authority：future `openapi/**`
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

- Compose 文件、项目注册、导入、刷新、`up/down/restart`、销毁保护，由 `project` module 拥有。
- 容器详情、日志、事件、Stats、Shell、Inspect、Network、Mount 仍由 `container` module 拥有。
- 若 `project` 需要消费容器运行时摘要，必须通过稳定共享边界复用，不得直接依赖 `container` module 私有实现。

## 2. 设计目标、非目标与术语

## 2.1 目标

- 支持把本机现有 Docker Compose Project 导入 `Graft` 管理。
- 支持保存项目注册信息、工作目录、Compose 文件选择、环境文件选择与最近一次成功解析快照。
- 支持项目级 `Overview`、`Services`、`Configuration`、`Activity` 四类信息架构。
- 支持项目级 `Refresh`、`Up`、`Down`、`Restart`、`Unregister`、`Destroy` 管理动作。
- 复用现有容器运行时能力，而不是新增第二套运行时真相。
- Phase 1 只做本机 `local host`，但模型要为未来远程主机预留边界。

## 2.2 非目标

- 不复制 Compose 文件。
- 不移动项目目录。
- 不重写用户已有目录布局。
- 不在 `Project` 下新增第二套容器详情页。
- 不在 `Project` 层持久化容器日志、事件、Stats 或运行时快照。
- Phase 1 不做 Git Project、Template、目录扫描、自动发现、远程主机。
- Phase 1 不做 Configuration 编辑器、Diff、Deploy、Validate UI。
- Phase 1 不做 Project Logs API、Project Events API 或 Project Realtime Topic。

## 2.3 术语

- `Project`
  - `Graft` 中的 Compose Project 注册与聚合对象。
- `Imported`
  - 指项目文件本来就存在于宿主机某个路径，`Graft` 只登记并管理它，不复制、不移动、不重写其布局。
- `Managed`
  - 指未来由 `Graft` 在受管根目录下创建的项目；它是 Source 概念，不自动等价于“允许销毁目录”。
- `Canonical Project Name`
  - Compose Runtime Identity；CLI 执行、容器归属匹配和项目成员识别都以它为准；默认只读。
- `Display Name`
  - `Graft` UI 展示名称；允许独立修改；不影响 Docker Runtime。
- `Snapshot`
  - 最近一次成功解析的静态配置结果，不是运行时缓存。
- `Drift`
  - 当前文件签名与最近一次成功刷新快照不一致的状态。

## 3. 架构分析

## 3.1 现有模块与职责清单

| 领域 | 现有路径 | 当前职责 | 对 Compose Project 的意义 |
| --- | --- | --- | --- |
| Project | 无专用 module | 仓库当前没有项目注册或 Compose Project 管理模块 | 必须新增 `server/modules/project/**` 与 `web/src/modules/project/**` |
| Container backend | `server/modules/container/**` | 容器列表、详情、日志、Shell、start/stop/restart/remove、资源摘要、运行时事件、Compose 来源识别 | 当前唯一可复用的 Runtime Authority |
| Container frontend | `web/src/modules/container/**` | 容器列表页、详情页、日志、运行时交互 | `Project -> Services -> Container Detail` 的现成落点 |
| Docker provider | `server/modules/container/docker_runtime.go` | Docker Runtime adapter；读取容器元数据、labels、日志、stats、事件 | 当前对 Compose 的认知仅限容器 labels |
| Runtime orchestration | `server/modules/container/service.go` | 容器用例编排、权限、审计、系统配置消费、实时 topic 注册 | 当前是 HTTP-first、module-private，用于复用时需要稳定共享边界 |
| Realtime / events | `server/modules/container/runtime_events*.go`、`log_topic_streamer.go`、`resource_stats_cache.go` | 容器事件流、日志 topic、资源采集与缓存 | 可复用为 Activity 的底层事实来源，但 Phase 1 不新增项目级聚合 API |
| Filesystem access | `server/modules/container/mount_usage.go`、`server/internal/config/config.go` | 挂载扫描、`.env` 与仓库根路径发现、基础文件系统读写 | 有基础文件系统经验，但没有 Compose 文件读取抽象 |
| Configuration core | `server/internal/config/**`、`server/modules/container/config.go` | Viper 配置、系统配置定义与读取 | 可复用系统配置框架承载 `Managed Projects Root` 等配置 |
| Database persistence pattern | `server/modules/notification/store/sql_repository.go`、`server/modules/announcement/store/sql_repository.go` | 模块自有 `database/sql` repository + migration pattern | `project` module 推荐沿用该模式，而不是先回到 Ent 中央仓储 |
| Module runtime / DI | `server/internal/module/**`、`server/internal/container/**`、`server/internal/moduleapi/**` | 模块生命周期、服务注册、跨模块稳定接口 | `project` 需要在这里定义最小稳定共享边界 |
| OpenAPI / generated schema | `openapi/**`、`web/src/contracts/openapi/generated/schema.ts` | 服务端与前端共享 wire contract | `project` 的 REST contract 必须走同一路径 |

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

| 能力 | 当前状态 | 证据结论 | 可复用建议 |
| --- | --- | --- | --- |
| Compose Project detection | `partial` | 已通过 `com.docker.compose.project` / `com.docker.compose.service` labels 识别容器来源 | 复用为项目成员匹配与运行态聚合基础 |
| Compose parsing | `no` | 仓库内无 Compose 文件解析器、无 `compose-go` 使用 | 新增 `project` module 静态解析能力 |
| Compose up/down/restart | `no` | 容器 module 仅有单容器动作，没有 `docker compose up/down/restart` 执行层 | 新增项目生命周期执行器 |
| Compose logs | `indirect yes` | 已有容器日志 API 和流能力，但没有项目级日志 API | Phase 1 通过前端 fan-out 复用 |
| Compose config | `no` | 无 `docker compose config` 或等价规范化预览接口 | 新增静态规范化快照与预览能力 |
| Compose events | `indirect yes` | 已有容器事件流，但没有项目级事件聚合 API | Phase 1 通过前端 fan-out 复用 |

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
- 真实 `up/down/restart` 语义应尽量复用 Docker Compose CLI 的行为，而不是自己模拟。
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
  - 使用参数化命令执行 `docker compose up/down/restart`。
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

### Project

- `id`
- `display_name`
- `canonical_project_name`
- `canonical_project_name_source`
  - `computed | override`
- `source_kind`
  - `imported | managed | git | template`
- `host_scope`
  - Phase 1 固定 `local`
- `working_directory`
- `ownership_mode`
  - `external | managed-root-dedicated`
- `last_refresh_status`
  - `never | success | failed`
- `last_refresh_at`
- `last_refresh_error_code`
- `last_refresh_error_message`
- `last_refresh_config_hash`
- `last_observed_config_hash`
- `last_drift_checked_at`
- `drift_status`
  - `unknown | clean | changed | missing`

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
- `exists_on_last_refresh`
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

### `compose_projects`

用途：

- 项目注册真相
- Source / Ownership / Drift / Refresh 元数据

不存：

- 容器运行时明细
- 日志
- 事件
- Stats

### `compose_project_files`

用途：

- 保存 Compose 文件与 Env 文件清单
- 支持未来多文件、有序 `-f` 合并与独立文件内容读取

### `compose_project_snapshots`

用途：

- 保存最近一次成功解析的规范化 Compose 快照

约束：

- 每个 project 只保留一条 latest successful snapshot
- 刷新失败时不覆盖旧 snapshot，只更新 project refresh error / drift 状态

## 7.3 推荐索引与唯一性

- `compose_projects(canonical_project_name, host_scope)` 唯一
  - 防止同一主机重复导入同一个 runtime identity
- `compose_projects(working_directory)` 普通索引
- `compose_project_files(project_id, order_index)` 唯一
- `compose_project_files(project_id, absolute_path)` 唯一
- `compose_project_snapshots(project_id)` 唯一

## 7.4 必须存储的元数据

- Source：`Imported` / future `Managed`
- Ownership 模式
- Working Directory
- Compose 文件有序列表
- Env 文件有序列表
- Display Name
- Canonical Project Name 及来源
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
- `Graft` 可以对它执行 `refresh/up/down/restart` 等管理动作。
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
- 记录 `last_refresh_status=failed`
- 在 Overview / Configuration 显示错误与“快照已过期”

## 8.4 Up / Down / Restart

这些动作属于 `Project Lifecycle`，不属于 `Container`。

语义约束：

- `Up`
  - 执行 `docker compose up -d`
- `Down`
  - 执行安全默认的 `docker compose down`
  - Phase 1 默认不删除 volumes
- `Restart`
  - 执行 `docker compose restart`

执行时必须显式传入：

- `--project-name`
- `--project-directory`
- 有序 `-f` 文件列表
- 明确 env file 参数

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

Phase 1 导入表单建议包含：

- `working_directory`
- `compose_files[]`
  - UI 默认自动发现；Phase 1 可只选择一个主文件，但 contract 必须支持有序多文件
- `env_files[]`
  - 默认自动识别 `.env`
- `display_name`
- `canonical_project_name_override`
  - 高级选项，可空

说明：

- 如果用户只输入 `working_directory`，系统先尝试自动发现标准 Compose 文件。
- `display_name` 与 `canonical_project_name` 必须分离，不再使用一个字段混合两种语义。

## 9.2 文件发现

默认发现顺序：

1. `compose.yaml`
2. `compose.yml`
3. `docker-compose.yaml`
4. `docker-compose.yml`

Phase 1 规则：

- 如果未显式选择文件，按上述顺序自动选取首个存在文件。
- 如果存在多个候选文件，UI 应允许用户显式确认。
- 合同层支持有序多文件；若 Phase 1 UI 只落单文件，数据模型也不能被锁死成单文件。

## 9.3 校验

导入前必须校验：

- `working_directory` 存在且可访问
- 选定 Compose 文件存在
- 路径解析后仍在允许边界内
- Compose 语法与规范化解析成功
- env file 可读取
- Canonical Project Name 可计算
- 唯一性校验通过

## 9.4 冲突检测

至少检测：

- 同一 `host_scope + canonical_project_name` 已存在
- 同一 `working_directory + compose file set` 已存在
- 选定文件缺失
- 文件内容不是有效 Compose 配置

## 9.5 导入步骤

1. 解析并规范化输入路径
2. 自动发现或确认 Compose 文件列表
3. 读取 env file 集合
4. 用 `compose-go` 做静态解析、合并与标准化
5. 计算 `canonical_project_name`、`config_hash`、`normalized_compose_json`
6. 执行唯一性与 ownership 预检查
7. 持久化 `compose_projects`、`compose_project_files`、`compose_project_snapshots`
8. 触发一次初始 `refresh` 结果写回
9. 返回项目详情摘要

## 9.6 输出

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
- `Up / Down / Restart / Refresh / Unregister / Destroy` 动作
- 容器详情跳转入口

## 10. API 提案

Phase 1 的 canonical OpenAPI authority 已收口到 `openapi/**`，本节继续保留 IA 与语义设计真相，不能与
`openapi/**` 漂移。

## 10.1 项目列表与详情

| Method | Path | 语义 |
| --- | --- | --- |
| `GET` | `/api/ops/projects` | 项目列表 |
| `GET` | `/api/ops/projects/{id}` | 项目详情 summary |
| `GET` | `/api/ops/projects/{id}/services` | 项目服务聚合 |

建议列表返回：

- `id`
- `display_name`
- `canonical_project_name`
- `source_kind`
- `ownership_mode`
- `working_directory`
- `runtime_status`
- `service_count`
- `container_counts`
- `last_refresh_at`
- `last_refresh_status`
- `drift_status`

## 10.2 导入与校验

| Method | Path | 语义 |
| --- | --- | --- |
| `POST` | `/api/ops/projects/import/validate` | 只校验输入与 Compose 解析，不持久化 |
| `POST` | `/api/ops/projects/import` | 导入并注册项目 |

`validate` 返回建议包含：

- 自动发现的 compose / env 文件
- 解析到的 `canonical_project_name`
- 规范化 preview 摘要
- 服务数
- warning / diagnostics summary
- 冲突信息

`import` 返回建议包含：

- 项目主记录
- 快照摘要
- 初次 refresh 结果

## 10.2A Phase 2 managed root 与 create contract

`phase-2-batch-1-managed-root-and-create-contracts` 只落 authority owner，不落真实 file write create flow。

新增 canonical contract owner：

| Method | Path | 语义 |
| --- | --- | --- |
| `GET` | `/api/ops/projects/managed-root` | 返回 managed create 的 system-config authority、ownership mode 与 readiness |
| `POST` | `/api/ops/projects/create/validate` | 只校验 managed create 输入、目标目录推导与 bounded authority，不写文件 |
| `POST` | `/api/ops/projects/create` | 保留 managed create canonical route；本 batch 只返回 accepted contract，不执行真实创建 |

managed root authority 约束：

- canonical config key 固定为 `ops.project.managed.root_directory`
- config owner 固定为 `server/modules/project/**`
- root directory 必须是绝对路径
- empty string 表示 managed create 尚未配置，不允许把“未配置”降级成 request payload fallback
- Phase 2 真实 create 只能在该 managed root 下创建 `managed-root-dedicated` 目录

managed create request 建议至少包含：

- `display_name`
- `canonical_project_name`
- `relative_project_directory`
- `compose_file_name`
- `env_file_name?`

本批次明确不做：

- 实际目录创建
- compose/env 文件写入
- editor / diff / validate / deploy flow
- 下游兼容字段或用 import contract 冒充 create contract

## 10.3 配置

为支持未来 Monaco / Diff / Download / Version，Configuration API 建议拆分：

| Method | Path | 语义 |
| --- | --- | --- |
| `GET` | `/api/ops/projects/{id}/configuration` | 文件列表、元数据、ownership、diagnostics summary |
| `GET` | `/api/ops/projects/{id}/configuration/preview` | 规范化 Compose preview |
| `GET` | `/api/ops/projects/{id}/configuration/files/{fileId}` | 单文件内容 |

Phase 1 的 `configuration` 返回建议包含：

- Compose 文件列表
- Env 文件列表
- 文件 metadata
- ownership summary
- drift summary
- last refresh summary

Phase 1 的单文件内容返回建议包含：

- `file_id`
- `kind`
- `path`
- `content`
- `encoding`
- `read_only=true`
- `download_name`

## 10.4 生命周期动作

| Method | Path | 语义 |
| --- | --- | --- |
| `POST` | `/api/ops/projects/{id}/refresh` | 刷新静态配置与聚合视图 |
| `POST` | `/api/ops/projects/{id}/up` | 执行 compose up |
| `POST` | `/api/ops/projects/{id}/down` | 执行 compose down，默认不删 volumes |
| `POST` | `/api/ops/projects/{id}/restart` | 执行 compose restart |
| `POST` | `/api/ops/projects/{id}/unregister` | 只删注册记录 |
| `POST` | `/api/ops/projects/{id}/destroy` | 高危销毁；受 ownership 保护 |

`destroy` 请求建议显式字段：

- `remove_named_volumes`
- `delete_working_directory`
- `confirm_canonical_project_name`

并要求后端返回：

- 哪些步骤已执行
- 哪些步骤被 ownership guard 拒绝
- 最终是否已注销

## 10.5 Phase 1 明确不提供的 API

- 不提供 `/api/ops/projects/{id}/logs`
- 不提供 `/api/ops/projects/{id}/events`
- 不提供项目级 realtime topic
- 不提供配置编辑保存接口

## 10.6 Batch 1 authority 落地说明

`phase-1-batch-1-project-contract-and-data-model` 已把以下 authority owner 固定到仓库运行面：

- OpenAPI contract owner：`openapi/**`
  - route space 固定为 `/api/ops/projects/**`
  - Phase 1 只读 Configuration 固定拆为 metadata、preview、single-file content
  - 明确保留 lifecycle routes 的 contract owner，但不在本 batch 落 runtime handler
- Project module contract owner：`server/modules/project/contract/**`
  - canonical route fragments
  - source / ownership / drift / refresh / file kind 等 typed contracts
- Project module data-model owner：`server/modules/project/model.go`
  - 只定义 registry、file list、snapshot 三类 module-owned persistence model
  - 不引入容器 logs / events / stats / inspect 等 runtime 字段
- Project module migration owner：`server/modules/project/migrations/**`
  - `compose_projects`
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
  - 新增 `/api/ops/projects/managed-root`
  - 新增 `/api/ops/projects/create/validate`
  - 新增 `/api/ops/projects/create`
- Project module contract owner：`server/modules/project/contract/**`
  - 新增 managed-root status typed contract
  - 新增 managed-create permission contract
  - 新增 managed-root config key contract
  - 新增 create route fragments
- Project module config-definition owner：`server/modules/project/config.go`
  - `ops.project.managed.root_directory` 成为 managed create 的 canonical system-config authority
  - empty string 表示未配置，而不是隐式回退到仓库路径或用户 home

本批次仍明确不做：

- managed create file write path
- managed create persistence bootstrap
- web managed create UI / editors
- diff / validate / deploy flow

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
  - `detail` 页面固定承载 `Overview`、`Services`、`Configuration`、`Activity` 四个页签
- Authority guard 已落地：
  - `Overview` 只承载 summary，不引入 runtime dashboard 指标或 timeline
  - `Services` 只消费静态定义与 container member/count 聚合，并回跳现有 Container Detail
  - `Configuration` 保持 metadata、preview、single-file content 三段只读消费
  - `Activity` 继续只做前端 fan-out，复用现有 container logs/events API
  - 未新增 backend project logs/events aggregation、managed create/editor/diff/deploy/validate UI

## 11. UI 信息架构

## 11.1 推荐层级

```text
Projects
  -> Project Detail
     -> Overview
     -> Services
     -> Configuration
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
- Running / Stopped Count

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
- 不必把这些信息散落在 Services / Configuration / Activity 三个页签
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

- Compose Files
- Env Files
- Preview
- Download

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

- `last_refresh_config_hash`
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
- 手动 `Refresh`
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
- `canonical_project_name` 默认只读
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

| 方向 | 是否兼容当前模型 | 设计说明 |
| --- | --- | --- |
| Git-based Projects | `yes` | 在 `source_kind` 上扩展 `git`，并追加 source metadata |
| Templates | `yes` | Template 实质上是 future `Managed Create` 的输入源 |
| Directory Scan | `yes` | 扫描只产出 candidates，不直接注册 |
| Auto Discovery | `yes` | 后台发现只更新 candidate / drift，不改变 runtime authority |
| Multiple Compose Files | `yes` | `compose_project_files` 的 `order_index` 已为有序 `-f` 预留 |
| Compose Override | `yes` | 通过 `role=override` 与有序文件列表支持 |
| Environment Files | `yes` | `kind=env` + file list 可扩展多个 env file |
| Remote Host | `partial` | 需在 `host_scope` 与连接配置上再扩展，但 registry 模型可保留 |
| Project Activity backend aggregation | `future yes` | 需要单独定义 observability authority，Phase 1 不做 |

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
- Up / Down / Restart
- Unregister
- Destroy with ownership protection

Observability：

- Overview summary
- Services aggregation
- Activity frontend fan-out aggregation

Configuration：

- Compose / Env file metadata
- Read-only file content
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

- Compose Editor
- Env Editor
- Diff
- Validate
- Deploy

## 16.4 Phase 3

Management：

- Git Project
- Templates
- Directory Scan
- Auto Discovery
- Remote Host

Observability：

- Project Activity backend aggregation

Configuration：

- Multi-file override UX 强化
- Git / template source metadata 深化

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
