# server/AGENTS.md

本文件是 `server/` 范围内后端任务的执行真相。

它约束后端架构、插件边界、Go 编码、Ent / migration 与 backend 校验链。
仓库级启动、恢复、提交、协作与跨仓库治理仍以根 `AGENTS.md` 为准；本文件不重复定义那些规则。

authority-first overlay：

- `server` owned scope 表示后端长期实现归属，不表示 `server` 可以单边决定 shared contract 的最终 authority
- bounded scope forbids unrelated expansion, not required authority repair
- 如果后端实现发现 drift 来自 OpenAPI source、shared contract、frontend bootstrap 依赖或其它上游 authority，必须升级到正确 authority owner，而不是默认要求另一端兼容

## 1. 适用范围

适用目录：

- `server/cmd/**`
- `server/internal/**`
- `server/plugins/**`

不适用目录：

- `web/**`
- 仓库根治理文件
- `ai-plan/**`

如果任务同时修改 `server` 与 `web`，先回到根 `AGENTS.md` 按跨边界任务处理；不要把本文件当作跨边界总规则。

## 2. 后端真相来源

后端任务至少以这些材料为真相来源：

- 根 `AGENTS.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`

按任务类型追加读取：

- 改动稳定契约、魔法值、shared semantics 时，读 `ai-plan/design/契约治理与魔法值治理规范.md`
- 改动注释、包文档、模块 README 或 AI 文档行为时，读 `ai-plan/design/代码注释与模块文档规范.md`

如果代码、文档与本文件冲突：

- 先判断是否是后端架构真相漂移
- 属于文档失真时，更新文档，不要默许代码继续偏离

## 3. Server Task Lifecycle

后端任务默认按这条固定生命周期执行，不为单个切片发明第二套流程。

### 3.1 Startup Preflight

- 先完成根 `AGENTS.md` 要求的 startup preflight，再进入 `server` 任务实现
- 至少确认：
  - governance source 是根 `AGENTS.md`
  - task class 是 `server`、`cross-boundary` 或 `docs/automation with server impact`
  - `.ai/environment/tools.ai.yaml` 已读取
  - `server/AGENTS.md` 已读取
- 只有在根 `AGENTS.md` 要求恢复上下文时，才继续读取 `ai-plan/public/README.md` 与对应 topic 文档
- 没有 startup receipt 时，不进入后端实现、验证结论或 closeout

### 3.2 Task Classification

- `server`
  - 只修改 `server/**`，且不会改变 `web` 的 menu、route、permission、OpenAPI 消费或共享契约语义
- `cross-boundary`
  - 同时影响 `server` 与 `web`，或改变共享 contract、menu、permission、route、OpenAPI、生命周期语义
- `docs/automation with server impact`
  - 只改治理文档、追踪文档、脚本或自动化，但这些内容会改变后端 agent 的执行方式、验证说明或 closeout 口径

如果分类结果是 `cross-boundary`，先回到根 `AGENTS.md` 追加读取 `web/AGENTS.md`，不要继续把本文件当作唯一规则来源。

### 3.3 Boundary Decision

- 开始写代码前，先判断改动归属：
  - 是 `core runtime`
  - 还是某个 `plugin`
  - 还是 `shared-stable-boundary`
- 归属判断之前，先做 authority discovery，确认当前看到的是 authority 错误还是下游 consumer 偏差
- 若 owner 无法明确，就先更新治理或设计文档，再写实现
- 任何新能力都必须先回答：
  - 谁拥有运行时生命周期
  - 谁拥有持久化真相
  - 谁拥有 route / menu / permission / message / capability

### 3.4 Implementation Path

- `core runtime` 改动只进入显式 core 边界，例如 `internal/app`、`internal/httpx`、`internal/plugin`、`internal/container`
- 业务能力默认进入 `plugins/<name>/**`
- 跨插件协作先设计 capability / contract，再写业务实现
- 数据结构变更先确认 schema owner，再进入 Ent generate 与 migration 流程

### 3.5 Validation

- 后端完成态统一走现有显式 CLI：
  - `graft validate backend --stage lint`
  - `go test` 最小直接覆盖范围
  - `go build ./cmd/graft`
  - 需要运行时证明时再跑 `graft validate smoke`
- 文档治理切片不需要伪造运行时验证，但要如实说明为什么未跑
- 不为当前切片发明第二套 shell 验证脚本、CI-only 流程或 hook-only 流程

### 3.6 Closeout

- 每个后端切片结束时都要留下统一 closeout 记录
- closeout 至少说明：
  - task class
  - owned scope
  - boundary decision
  - 是否涉及 schema / migration
  - 实际运行的验证命令与结果
  - 跳过的验证项与原因
- 本文件只要求 closeout 可审计，不要求在本轮把这些记录升级为 CI、hook 或脚本硬门禁

## 4. 当前后端结构边界

当前 `server` 的执行面以这些目录为主：

- `cmd/graft`
  - `graft` CLI 入口；显式承载 `serve`、`migrate`、`dev`、`validate`
- `internal/app`
  - runtime 装配、core 资源生命周期、插件调度
- `internal/cli`
  - 后端显式 CLI 命令树；不要把 runtime 魔法塞进 shell 脚本
- `internal/config`、`internal/logger`、`internal/database`、`internal/redisx`
  - core 基础设施初始化边界
- `internal/httpx`
  - Gin server、统一响应、鉴权中间件等 HTTP 运行时边界
- `internal/container`
  - 轻量单例 DI / service container
- `internal/plugin`
  - 插件契约、上下文、插件排序与生命周期管理
- `internal/pluginapi`
  - 跨插件稳定接口与 DTO
- `internal/contract`
  - 平台级稳定 typed contract
- `internal/menu`、`internal/permission`、`internal/cronx`、`internal/eventbus`、`internal/i18n`
  - 平台声明式注册面与公共运行时能力
- `internal/store`、`internal/store/entstore`
  - 仅保留现阶段尚未迁出的 core-owned 数据访问边界；长期方向不是继续集中新增业务仓储
- `internal/ent/migrate/migrations`
  - 仅保留历史共享 Atlas migration 目录，供显式/manual 诊断或回放使用；它不再属于默认 migration 链
- `plugins/*`
  - 业务插件与插件自有 contract；长期方向下每个插件还应拥有自己的 capability、store、storeent、ent 与 migrations

除这些显式边界外，不要再发明隐藏 runtime surface。新的平台级入口如果不能清楚归入现有边界，先更新设计再写代码。

## 5. 后端目标与核心边界

`server` 是组合式后台平台的运行时，不是单一业务应用。

必须保持：

- core 只拥有基础设施与扩展机制
- 业务能力只放在 `plugins/*`
- 插件之间通过稳定接口协作，不直接依赖彼此内部实现
- 装配路径显式、可追踪、可测试

不要做：

- 把业务规则塞进 `internal/app`、`internal/plugin`、`internal/container` 等 core 包
- 通过 package global、`init()`、隐式扫描或反射魔法制造运行时行为
- 把插件私有实现暴露成跨插件公共 API

## 6. 边界判定矩阵

后端任务在进入实现前，先用下表决定 owner；不能跳过这一步直接写代码。

| 目标 | 默认归属 | 说明 |
| --- | --- | --- |
| runtime 装配、启动/关闭顺序、core 生命周期 | `server/internal/app/**` | 只放平台级 runtime 行为 |
| CLI、显式迁移、显式验证、开发编排 | `server/cmd/**`、`server/internal/cli/**` | `serve`、`migrate`、`dev`、`validate` 保持显式入口 |
| HTTP 通用能力、统一响应、共性中间件 | `server/internal/httpx/**` | 不承载插件业务规则 |
| 容器、事件总线、菜单/权限/cron 注册器 | `server/internal/**` 对应 core 包 | 只放平台公共能力 |
| 业务 API、业务 service、业务 store、业务 schema、业务 migration | `server/plugins/<name>/**` | 默认 plugin-owned |
| 跨插件 capability、共享 DTO、稳定事件名 | `server/internal/pluginapi/**` | 只放稳定跨插件公开面 |
| 平台级 typed contract | `server/internal/contract/**` | 只放平台共用 contract |
| 插件私有稳定 contract | `server/plugins/<name>/contract/**` | route fragment、permission code、message key 等 |
| 历史共享 migration 回放 | `server/internal/ent/migrate/migrations/**` | 仅显式/manual 使用，不是默认链 |

补充规则：

- 新业务能力默认先问“能否直接放进 `server/plugins/<name>/**`”；除非它是平台基础设施，否则不要先放 `internal/**`
- 新的业务 schema、Ent 生成产物、业务 migration 真相不允许回流 `server/internal/ent/**`
- 插件间协作默认优先扩展 `pluginapi` 或稳定 `contract`，而不是直接引用对方内部实现

## 7. 插件生命周期与边界

当前 backend plugin 都遵循 `Name / Version / DependsOn / Register / Boot / Shutdown` 契约。

`server` 的长期并行开发方向保持为 compile-time modular monolith：

- 单体进程
- compile-time wiring
- deterministic startup
- 不做 runtime plugin loading / discovery / hot-load
- 不做 generalized reflection plugin system
- 不做 generalized service locator

后续治理允许新增 `plugin.Descriptor`、`plugin.Builder` 与 compile-time generated plugin registry，作为显式装配
抽象；这些抽象的目的仅是降低多工作树并行开发冲突，不是把当前仓库扩展成运行时插件平台。

### 7.1 生命周期规则

- `Register`
  - 只做声明式注册
  - 允许注册路由、菜单、权限、message key、事件处理器、定时任务定义、公开服务、配置语义
  - 不允许启动 goroutine、阻塞 I/O、长时间初始化、隐式修改外部状态
- `Boot`
  - 只启动已经在 `Register` 阶段声明过的运行时行为
  - 可以依赖其它插件已经注册的稳定公开服务
  - 不允许新增未声明的路由、权限、菜单、message、job、公开服务
- `Shutdown`
  - 负责释放 `Boot` 启动的所有资源
  - 必须停止 goroutine、ticker、timer、event subscription、scheduler job、外部句柄
  - 不得以 `context.Background()` 逃避关闭语义；优先使用 runtime 注入的生命周期上下文

### 7.2 依赖规则

- 插件依赖通过 `DependsOn()` 声明
- 服务依赖通过稳定接口解析
- 缺失依赖、循环依赖、重复注册都属于阻断错误
- 插件只能依赖：
  - `internal/pluginapi/**`
  - `internal/contract/**`
  - 其它插件公开的 capability contract 或 stable DTO contract
- 插件不能直接 import：
  - 其它插件的 `service/**`
  - 其它插件的 `storeent/**`
  - 其它插件的 `ent/schema/**`
  - 其它插件的 migration 文件或 migration 目录
- 插件不能直接依赖其它插件的内部 repository、handler、store、Ent entity
- 若需要跨插件业务能力，必须通过 capability interface 或 stable DTO contract 暴露

### 7.3 插件公开面规则

插件运行时可见能力必须能追溯到生命周期：

- 路由
- 菜单
- 权限
- message key / bundle
- 事件订阅
- cron job 定义
- 公开服务

如果某个运行时能力无法追溯到 `Register -> Boot -> Shutdown`，就说明边界失控。

## 8. Plugin Implementation Checklist

当后端任务属于 plugin 路径时，默认按这份 checklist 检查实现是否完整：

- `descriptor`
  - 是否声明稳定 plugin ID、版本、依赖、migration path、builder
- `plugin lifecycle`
  - 是否实现 `Name / Version / DependsOn / Register / Boot / Shutdown`
  - `Register -> Boot -> Shutdown` 职责是否清晰
- `routes`
  - 路由是否留在插件边界内，且只编排输入输出、鉴权和响应映射
- `service`
  - 业务用例是否下沉到插件 service，而不是堆在 handler 或 core
- `store / storeent`
  - 持久化边界是否留在插件内，没有回流 `internal/store/**`
- `migration`
  - schema 变更是否落到插件自有 migration，且 owner 明确
- `messages / permissions / menus`
  - 对外可见能力是否在 `Register` 阶段声明，并与 contract 对齐
- `tests`
  - 是否覆盖直接受影响的 service、route、store、contract 或 lifecycle 路径
- `README`
  - 插件 README 是否能说明职责边界、主要入口、关键依赖与不负责的范围

## 9. `internal/pluginapi` 与契约边界

跨插件公开接口统一收敛到稳定边界：

- `server/internal/pluginapi`
  - 放跨插件能力接口、共享 DTO、稳定错误语义、稳定事件名
- `server/internal/contract`
  - 放平台级稳定 contract，例如 header、auth scheme、error code、平台消息 key
- `server/plugins/<plugin>/contract`
  - 放插件自有稳定 contract，例如 route fragment、permission code、message key

规则：

- 跨插件只暴露 capability-oriented interface，不暴露 repository、Ent client、plugin private struct
- 跨插件返回值优先使用稳定 DTO，不直接返回 Ent entity 或数据库模型
- capability 必须在 Builder 或其它 compile-time 装配阶段注册，而不是在运行后期临时拼装
- capability 生命周期必须稳定，明确由哪个插件提供、何时可用、何时关闭
- capability 只允许暴露：
  - cross-plugin business ability
  - dev/reset hook
  - stable query/service contract
- `auth`、`user`、`rbac`、`resource` 的治理边界固定为：
  - `auth` 只拥有认证与会话生命周期，回答“你是谁”
  - `user` 只拥有用户资料与用户管理能力，不等于 `auth`
  - `rbac` 只拥有授权模型，回答“你能做什么”
  - `resource` 只表达被保护对象概念边界，当前继续由 `rbac` 承载 resource/action/permission metadata
- `user` 作为 `rbac` 的上游插件时，只暴露稳定用户能力：
  - 用户存在性检查
  - 用户基础身份查询
  - 用户删除前约束检查
- `user` 不拥有 `user_roles`，也不对外暴露角色分配实现细节、`user_roles` repository、`user_roles` schema 或对应 Ent 包
- `rbac` 若需要校验 `user_id`，必须通过 `user` 暴露的稳定 capability / contract 完成；禁止直接 import `user` 的 Ent 包或其它私有持久化实现
- 同一高风险语义只能有一个 canonical definition
- route contract 优先保持 `group path + route fragment` 真相，不要为同一语义并存多套 full path 常量
- `permission code`、`event name`、`message key`、`header name`、`auth scheme`、共享状态枚举都属于高风险 contract
- 新增或修改高风险 contract 时，必须明确 owner 与 lifecycle：`experimental` / `stable` / `deprecated` / `removed`
- auth route contract 的 canonical owner 固定为 `server/plugins/auth/contract`；兼容期即使暂时仍由其它插件承载运行时实现，也不得把 owner 真相写回 `user`
- 管理员按用户维度的 session 治理若继续暴露在 `/users/:id/sessions`，它也只是 `user` 的管理入口；session 持久化、token/cookie、rotation/revoke 真相仍归 `auth`
- 兼容 alias 只能临时存在，不能演变成永久第二真相

## 10. DI 与运行时装配

`internal/container` 是轻量显式单例容器，不是通用 service locator。

容器只负责：

- 注册单例 provider
- 解析单例实例
- 复用并发构造结果

容器不负责：

- 包扫描
- 反射自动注入
- struct tag 注入
- 隐式依赖图生成
- request scope / session scope
- 业务路径中的随手 `Resolve`
- generalized capability lookup

规则：

- 依赖通过构造函数或生命周期 wiring 显式注入
- `Resolve` 只允许出现在 runtime 装配、plugin lifecycle adapter、middleware wiring 等窄组合边界
- handler、service、repository、store、DTO、Ent schema 里不要散落容器解析
- 共享资源如 logger、database、redis、event bus、scheduler 由 runtime 管理生命周期
- 不允许在业务代码里临时 new 基础设施依赖来绕开 runtime

## 11. Core、store 与 HTTP 边界

规则：

- `internal/app` 只负责 runtime 资源装配、插件编排、关闭顺序，不承载业务用例
- `internal/httpx` 负责 HTTP 运行时共性，不承载某个插件专有业务规则
- `internal/store` 与 `internal/store/entstore` 只允许承载 core-owned 或尚未迁移完成的历史集中边界，不应继续接纳新的业务插件真相
- 长期方向下，业务插件应收敛到：
  - `plugins/<name>/store/**`
  - `plugins/<name>/storeent/**`
  - `plugins/<name>/service/**`
  - `plugins/<name>/routes/**`
- 插件的 handler / route 文件只编排 HTTP 输入输出与授权边界，不直接堆业务事务脚本
- 一旦某个业务边界迁移到插件目录，禁止把 repository、service、handler 重新回流到 `internal/store/**`、`internal/app/**` 或其它 core runtime 包

## 12. Ent 与 migration 规则

Ent 与 Atlas 是后端数据库真相链路的一部分。

多工作树 Phase 1 治理补充：

- 这里说的“zero-shared”指功能开发工作树的零共享，不是要求所有 tracked file 都绝对零共享
- 长生命周期的 `server` feature worktree 正常情况下只拥有 `server/plugins/<name>/**`
- `internal/pluginapi/**`、`internal/contract/**`、`internal/pluginregistry/generated.go`、`cmd/graft/**`、
  `AGENTS.md`、`ai-plan/**` 与 migration 入口变更都属于短生命周期 integration / core slice，不属于长期 feature
  worktree 的 standing ownership
- `internal/pluginregistry/generated.go` 当前仍保持 tracked；但长生命周期 feature worktree 不得直接修改它，相关改动必须回到显式集成切片统一协调
- 共享 `internal/ent` Go 代码与 schema 兼容层已经删除；不得在该路径重新引入业务 schema、生成产物或 runtime 依赖
- `internal/ent/migrate/migrations/**` 仍保留为历史共享 migration 目录，但仅允许显式/manual 使用；它不是默认 migration 链的一部分
- 当前后端 ownership checkpoint 允许 fresh DB rebuild；不要求为历史 mixed migration 继续维持兼容回放能力

规则：

- `internal/ent/migrate/migrations/**` 只允许承载历史共享 Atlas migration 真相；不得新增新的默认 migration、业务 schema 真相或新的 owner-aligned 基线
- 每个业务插件应长期收敛到自己的：
  - `plugins/<name>/ent/**`
  - `plugins/<name>/migrations/**`
- live Ent 生成产物只允许存在于 `plugins/<name>/ent/**`；不得把生成代码重新集中回 `internal/ent/**`
- schema 变化必须通过显式 Ent 生成与显式 migration 流程落地，不允许靠 runtime 自动同步数据库
- `graft migrate up` 是显式迁移入口；`graft serve` 不得隐式修改 schema
- migration 文件必须保持可审计、可回放、可按版本追踪；不要把业务初始化偷偷塞进不可追踪的启动逻辑
- 一个 migration 只能修改：
  - 当前 owner 拥有的表
  - 或 core-owned 表
- `user_roles` 的最终表 owner 是 `rbac`
- `rbac` 拥有 `user_roles` 的 Ent schema、repository、migration 与测试
- 历史 mixed Atlas migration 不重写；若要把 ownership checkpoint 写入迁移链，只允许通过 `rbac` 的 forward-only migration 增量记录
- 禁止：
  - `rbac` migration 修改 `user` 表
  - `audit` migration 修改 `rbac` 表
  - 任一插件 migration 修改其它插件 schema
- 跨插件关联只允许：
  - 稳定外键，由表 owner 明确声明
  - application-level contract 协作
- 每个插件独立进行 Ent generate，生成代码只能写入 `plugins/<name>/ent/**`
- `user_roles` 的协作边界保持为 `user_id / role_id` 标识符级别；不要通过跨插件 Ent edge、跨插件 Ent entity 或跨插件 repository 暴露角色分配耦合
- 禁止聚合式全局业务 Ent generate、禁止一个插件修改其它插件的 ent 产物、禁止插件修改 core ent runtime

当任务修改以下任一内容时：

- 任一 live Ent schema 路径，例如 `plugins/<name>/ent/schema/**`
- 任一 live Ent 生成入口，例如 `plugins/<name>/ent/generate.go`
- Atlas/Ent 迁移生成相关配置
- 任何影响 schema 语义的手写代码

完成态必须额外执行：

- 对受影响的 live Ent 包运行对应的 `go generate`
- 通过现有显式 migration 流程生成或更新 migration 文件
- 对受影响的 Ent 包运行最小直接 `go test`
- `cd server && go test ./...`

不要声称 schema 已完成治理但缺少生成结果或 migration 对应更新。若未来需要恢复 core-owned Ent 生成入口，必须先在文档中显式声明该路径，而不是静默复活 `internal/ent/**`。

## 12.1 多工作树 owned scope

`server` 的长期多工作树 owned scope 以插件优先：

- `shared-stable-boundary`
  - `internal/pluginapi/**`
  - `internal/contract/**`
- `generated-shared-hotspot`
  - `internal/pluginregistry/generated.go`
- `plugin-owned`
  - `plugins/<name>/**`
- `core-owned`
  - `internal/app/**`
  - `internal/plugin/**`
  - `internal/httpx/**`
  - `internal/config/**`
  - `internal/logger/**`
  - `internal/database/**`
  - `internal/container/**`
  - `internal/eventbus/**`
  - `internal/menu/**`
  - `internal/permission/**`
  - `internal/cronx/**`
  - `internal/redisx/**`
  - `internal/migration/**`
  - `internal/ent/migrate/migrations/**` 仅限历史共享 Atlas migration 目录

允许长期共享修改的白名单仅包括：

- `internal/pluginapi/**`
- `internal/contract/**`
- `internal/pluginregistry/generated.go`
- `cmd/graft/**`
- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/**`

除白名单外，其它目录默认视为 owned scope，不应被多个长期工作树共同持有。

overlay 解释：

- 这些 owned scope / hotspot 规则用于降低长期冲突，不用于阻止必要的 authority escalation
- 若 authority 位于共享契约、OpenAPI source、compile-time wiring 或跨端 bootstrap 语义，允许通过最小必要 cross-boundary 切片修复
- 不得把 worktree 隔离语义误用成“由下游 consumer 继续兼容上游 drift”

长期多工作树默认分两类：

- `main` 共享基线 worktree
  - 只负责共享治理、共享热点收口、active topic/worktree 映射准备、以及尚未稳定下沉前的短期过渡修整
  - 不应长期承担某个业务插件的日常 feature 开发
- dedicated long-lived worktree
  - 一条长期 worktree 只对应一个清晰 owned scope
  - owned scope 必须在 tracking 或治理文档中写明，不允许靠“当前是谁在改”临时推断
  - 若某项改动同时需要多个长期 worktree 频繁碰同一目录，说明该 owned scope 尚未稳定，应先回到 `main` 共享基线治理

长期 worktree 的 owned scope 声明至少要回答：

- 该 worktree 拥有哪些 `plugin-owned` 或 `core-owned` 目录
- 允许触碰哪些 `shared-stable-boundary`
- 是否允许触碰 `generated-shared-hotspot`
- 遇到 `internal/ent/migrate/migrations/**`、`internal/app/**`、`internal/plugin/**` 这类 core 共享面时，是回到 `main` 治理还是切出单独 core worktree

shared hotspot 处理规则如下：

- `shared-stable-boundary`
  - 只允许承载稳定 capability、DTO、typed contract 与共享治理文字真相
  - 进入该边界的改动必须同时说明 canonical owner 与 consumer；不要把临时业务实验塞成长期共享接口
- `generated-shared-hotspot`
  - `internal/pluginregistry/generated.go` 是唯一允许的集中接线产物
  - 该文件只能承载 compile-time registry 的机械生成结果，不允许手写业务规则、兼容分支或第二套插件真相
  - 若多个长期 worktree 同时需要修改它，应把该变更视为可预期冲突面，并通过短生命周期集成或共享基线串行收口，而不是扩大共享编辑范围
- `core-owned` 高冲突面
  - `internal/ent/migrate/migrations/**`、`internal/app/**`、`internal/plugin/**`、`internal/migration/**` 默认不是长期共享编辑面
  - 某个插件 worktree 一旦需要持续修改这些目录，必须先明确它是在进行 core-owned 治理还是插件 feature 开发；两者不要混在同一长期 worktree 里无限扩张

从 `main` 共享基线切换到 dedicated long-lived worktree 前，至少满足：

- 该方向已经有稳定 owned scope，而不是仍在反复争抢共享热点
- 该方向的共享热点白名单已明确，不再依赖“默认可以改所有白名单”
- 该方向的 tracking / trace 恢复入口已准备好，能让后续会话恢复时直接知道 branch、worktree、owned scope 与验证责任
- 该方向的日常验证路径已经清楚，避免 worktree 建好后仍靠 `main` topic 兜底判断完成态

切换完成后应收紧职责：

- dedicated worktree 默认只改自己的 `plugin-owned` 或明确声明的 `core-owned` 目录
- `main` 共享基线只保留共享热点治理、跨 worktree 对齐、topic/worktree 映射与归档调整
- 如果 dedicated worktree 需要新增共享边界或改变 ownership，先更新治理文档，再扩展代码面

与 `user` / `rbac` 边界直接相关的多工作树规则再补充为：

- `RBAC` worktree 可以修改 `user_roles` 相关的 schema、repository、migration、测试与 plugin-local contract
- `User` worktree 不直接修改 `user_roles`
- `User` worktree 若需要配合角色分配语义，只能修改 `user` 自有稳定 capability / contract，并通过共享治理文档或共享稳定边界与 `RBAC` worktree 对齐

## 13. 明确禁止项

以下事项默认禁止，除非先更新治理与设计真相：

- 在 `Register` 阶段启动 goroutine、定时任务、长时间 I/O、阻塞初始化或其它运行时行为
- 在 `Boot` 阶段做 schema 修改、隐式迁移、运行时补注册路由/权限/菜单/message/service
- 让 `graft serve` 隐式执行 migration、schema sync 或其它数据库结构修改
- 让 plugin 直接引用其它 plugin 的内部 repository、service、handler、storeent、Ent entity、schema 或 migration
- 在 `server/internal/ent/**` 重新引入新的业务真相、业务 schema、业务生成产物
- 实现 runtime plugin scan、dynamic discovery、hot plug、reflection-heavy plugin system
- 把 container 当成通用 service locator，在业务路径里随手 `Resolve`
- 通过 `init()`、package global、隐式扫描把运行时行为偷偷塞进 core 或 plugin

## 14. Go 编码规则

本节适用于 `server` 下手写 Go 代码。

### 14.1 文件与包

- 文件名全小写，多个单词用下划线
- 测试文件使用 `*_test.go`
- 不新增 `misc.go`、`common.go`、`utils.go`、`helper.go` 这类默认落点
- 一个文件只承载一个主要职责；跨越多个独立关注点就拆文件
- package 名短、小写、无下划线，并表达明确职责
- 不用 `manager`、`helper`、`common`、`utils` 充当万能包名

### 14.2 类型、函数与字段命名

- 导出标识符用 `PascalCase`
- 非导出标识符用 `lowerCamelCase`
- 类型名表达业务语义，不用 `BaseManager`、`CommonService`、`DataHandler`
- 接口优先描述能力，例如 `UserService`、`Authorizer`、`Factory`
- 只在确有多实现、跨边界依赖或测试替身需要时定义接口
- 构造函数优先 `NewXxx`
- 布尔方法优先 `Is`、`Has`、`Can`、`Allow`
- 函数名使用清晰动词；不要滥用 `Do`、`Handle`、`Process`、`Run`
- 结构体字段名必须表达角色，不使用难懂缩写

### 14.3 Context

- 请求链路必须透传 `context.Context`
- `context.Context` 必须是函数第一个参数
- 请求链路中不要随意新建 `context.Background()`
- handler -> service -> store -> database / redis / http client 必须保持上下文传递
- 请求派生 goroutine 必须响应 `context cancel`
- `context.Value` 只用于请求级元数据，不用来塞 service、logger、config、repository

### 14.4 HTTP、DTO 与 API 边界

- handler 不直接暴露 Ent entity
- request / response 必须显式定义 DTO
- API response 通过统一响应结构输出
- 不把数据库字段、内部外键、Ent edge 细节直接泄漏给外部 API
- 不把 `map[string]any` 当主响应结构
- route handler 先做输入校验、鉴权、调用 service，再做响应映射；不要把业务编排堆在 Gin handler 里

### 14.5 配置

- 配置统一通过 `internal/config` 加载
- 业务代码不直接 `os.Getenv`
- 默认值、校验、fail fast 都集中在配置边界处理
- 不给 `secret`、`token`、`password` 之类敏感配置写死默认值
- 配置结构表达业务语义，不照抄环境变量原文命名
- 不把运行时状态、句柄、请求上下文塞进 config

### 14.6 Wiring 与依赖注入

- 依赖必须显式 wiring
- 不允许隐藏全局单例
- 不允许通过 `init()` 偷偷注册运行时依赖
- service 依赖通过构造函数注入
- 插件不能绕过 runtime 直接控制其它插件内部状态
- wiring 依赖保持单向；core 不反向依赖业务实现

### 14.7 鉴权与安全

- 权限判断通过 middleware、auth service 或 permission checker 统一处理
- 不在 handler 内散落硬编码角色判断
- token、session、secret 的校验与签发语义必须集中管理
- token 校验失败返回稳定错误语义
- 默认拒绝未知权限
- 不向前端泄漏敏感内部错误、数据库细节、token 内容、secret 内容
- 认证相关时间语义统一使用 UTC 或仓库统一时区策略

### 14.8 事务

- 事务边界优先放在 service / usecase 层
- handler 不直接编排数据库事务
- repository / store 默认不自行开启隐藏事务
- 同一业务事务中的 store 调用必须共享同一 `tx`
- `Rollback` 必须通过 `defer` 保证
- `Commit` 后不得继续使用旧 `tx`

### 14.9 错误处理

- 显式处理 `error`
- 包装错误统一使用 `fmt.Errorf("context: %w", err)`
- 错误上下文必须说明当前操作
- 不为了过编译吞错、返回无理由 `nil`、或用空分支掩盖失败路径
- 除启动期不可恢复的编程错误外，底层逻辑不直接 `panic`
- handler 不把底层数据库错误直接返回给前端

### 14.10 并发与资源生命周期

- 新增 goroutine 必须有明确生命周期与退出条件
- 禁止无边界后台 goroutine
- ticker、timer、rows、pubsub、response body、file、tx 等资源必须显式 `Stop`、`Close`、`cancel` 或 `Rollback`
- channel 由创建方负责关闭
- 不允许无限重试循环且没有 `sleep`、`backoff` 或 `context cancel`
- 不允许 silently recover panic 后继续假装系统健康

### 14.11 日志与注释

- 业务日志统一通过日志模块输出；不要用 `fmt.Println` 或 `log.Println`
- 请求链路日志应带稳定请求标识
- 不记录 password、token、secret、cookie、authorization header 等敏感值
- 高频路径不要滥打 `info/debug`
- 导出类型、函数、常量写 GoDoc，首句以标识符开头
- 注释解释职责、边界、副作用、生命周期，不复述显然代码

## 15. 后端验证链

后端完成态的仓库内显式 CLI 入口是：

- `cd server && go run ./cmd/graft validate backend`

如果已经构建出 `graft` 可执行文件，`graft validate backend` 只是同一入口的另一种调用方式；不要再发明第二套 blocking validation 命令。

### 15.1 固定规则

- backend blocking lint gate 唯一入口是 `graft validate backend --stage lint`
- 统一使用 `golangci-lint v2.12.2`
- lint gate 以 changed-file scoped、`--new-from-rev=<merge-base> --whole-files` 语义执行；不要把 untouched backlog 混成当前切片阻断项
- 新代码不能扩大 lint backlog
- full backend completion 顺序固定为：
  - `graft validate backend --stage lint`
  - `go test` 最小直接覆盖范围
  - `go build ./cmd/graft`
  - 需要运行时证明时再跑 `graft validate smoke`

### 15.2 选择最小正确验证

- 只改 plugin 内业务逻辑时，优先测受影响 package，并补 `go build ./cmd/graft`
- 改 `internal/httpx`、`internal/plugin`、`internal/container`、`internal/app` 等 core 边界时，默认扩大到覆盖相关 `internal/...` 测试
- 改 schema、migration、store、plugin public contract 时，不要只跑单包 smoke 代替单元或集成验证
- 只有当任务确实需要证明迁移与运行时启动链条时，才追加 `graft validate smoke`

### 15.3 不允许的完成态说法

以下情况不能称为“后端已完成”：

- 没跑统一 lint gate
- 只跑 `go test ./...`，但没有经过 `graft validate backend --stage lint`
- 修改 Ent schema 后没生成代码、没补 migration 或没做相关测试
- 用“已有历史 warning”或“CI 以后会看”来跳过当前切片验证

## 16. Closeout 记录模板

后端切片 closeout 默认至少包含这些字段；可以写在最终说明、tracking、trace 或任务 closeout 中，但字段不能缺失。

- `Task class`
  - `server` / `cross-boundary` / `docs/automation with server impact`
- `Owned scope`
  - 本轮确认拥有的目录、文件或共享边界
- `Boundary decision`
  - 本轮改动属于 `core runtime`、`plugin-owned` 还是 `shared-stable-boundary`
- `Schema/migration touched`
  - `yes` / `no`
  - 若是 `yes`，补充受影响路径与 owner
- `Validation commands`
  - 实际运行过的命令
- `Validation results`
  - pass / fail / not run，并说明范围
- `Skipped validations and reasons`
  - 任何没跑的预期验证都要写原因
- `Shared/governance docs touched`
  - 是否修改了 `AGENTS.md`、`ai-plan/**`、共享治理文档

推荐格式：

```text
Task class: server
Owned scope: server/plugins/audit/**, server/AGENTS.md
Boundary decision: plugin-owned audit slice with shared governance doc touch
Schema/migration touched: no
Validation commands:
- cd server && go run ./cmd/graft validate backend --stage lint
- cd server && go test ./plugins/audit/...
Validation results:
- lint passed
- plugin tests passed
Skipped validations and reasons:
- graft validate smoke not run; this slice does not change runtime startup or migration behavior
Shared/governance docs touched: yes
```

## 17. 评审关注点

后端评审默认优先看这些问题：

- plugin boundary 是否被 core 或其它插件内部实现穿透
- `Register / Boot / Shutdown` 是否混淆
- `internal/pluginapi`、`internal/contract`、插件 contract 是否出现重复语义
- 容器是否被当成普通 service locator 滥用
- handler 是否泄漏 Ent entity、事务细节或底层错误
- schema / migration / generated output 是否失配
- 验证链是否与改动范围匹配

如果代码能跑，但这些边界被破坏，仍然算不合格实现。
