# Server 多工作树并行化治理计划（Compile-Time Modular Monolith）

## Summary

- 目标是把 `server` 治理成长期可并行开发的单体：业务按插件拆分、运行仍是单进程、接线仍是 compile-time、启动顺序仍 deterministic。
- 本轮不做运行时插件平台，不引入动态发现、热加载、外部插件市场、分布式插件协议，也不把当前单体演进成微服务。
- 完成后应满足三类验收：
  - `user`、`rbac`、未来新插件可在独立工作树长期开发，主冲突面限制在少数刻意共享热点
  - 新增插件低冲突接入，不再要求手改一批中心化核心文件
  - 为未来可能的“三方插件生态”预留稳定边界，但当前实现仍然只是 compile-time modular monolith

## Governance Rules

### Compile-Time Monolith Baseline

- `server` 保持单体进程、单 runtime、单启动链，不引入第二套运行模型。
- 插件注册必须是 compile-time wiring；registry 由生成步骤产出，运行期只消费生成结果。
- 启动顺序必须 deterministic，禁止依赖运行期扫描顺序、反射发现顺序或文件系统枚举顺序。
- 当前运行行为应尽量保持不变：仍然是 `serve -> build runtime -> order plugins -> Register -> Boot -> run HTTP -> Shutdown`。

### Plugin Dependency Rules

- 插件只允许依赖：
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - 其它插件公开的 capability contract 或 stable DTO contract
- 插件禁止直接依赖：
  - 其它插件的 `service/**`
  - 其它插件的 `storeent/**`
  - 其它插件的 `ent/schema/**`
  - 其它插件的 migration 文件或 migration 目录
- 跨插件业务协作必须通过 capability interface、stable DTO、stable error semantic 或 stable event/contract 完成。
- 任何“为了方便临时 import 另一个插件内部实现”的做法都视为治理失败，不允许作为过渡常态保留。

### Migration Ownership Rules

- 表 ownership 是唯一真值；每张表只能有一个 owner：`core` 或某个插件。
- 一个 migration 只能修改：
  - 当前 owner 拥有的表
  - 或 `core-owned` 表
- 禁止：
  - `rbac` migration 修改 `user` 表
  - `audit` migration 修改 `rbac` 表
  - 任一插件 migration 修改其它插件 schema
- 跨插件关联只允许两种方式：
  - 稳定外键，由表 owner 明确声明并由依赖顺序保障生成
  - application-level contract 协作，不直接跨插件改表
- 不允许通过“顺手补字段”把 migration 热点重新集中到共享目录。

### Ent Generation Rules

- 每个插件独立进行 Ent generate。
- 生成代码只能写入 `server/plugins/<name>/ent/**`。
- `server/internal/ent/**` 只允许承载 `core-owned` schema 与其生成产物。
- 禁止：
  - 聚合式全局业务 ent generate
  - 一个插件修改其它插件的 ent 产物
  - 插件修改 `core` 的 ent runtime 或生成结果
- 目标是让 Ent 生成产物跟随 plugin owned scope，而不是重新回流到中心化冲突区。

### Registry Generation Constraints

- plugin registry 必须是 compile-time generated。
- `generated.go` 是唯一允许的集中接线产物。
- 生成结果必须 deterministic：相同输入得到相同顺序、相同输出。
- 禁止：
  - runtime filesystem scan
  - runtime plugin discovery
  - runtime hot-load
  - generalized reflection plugin system

### Plugin Capability Exposure

- capability 必须在 Builder 阶段注册，不在运行后期临时拼装。
- capability 必须生命周期稳定，能明确说明由哪个插件提供、何时可用、何时关闭。
- capability 不得暴露 repository、ORM entity、Ent client、storeent 实现或插件内部 service struct。
- capability 只允许暴露：
  - cross-plugin business ability
  - dev/reset hook
  - stable query/service contract
- capability registry 不能演变成 generalized service locator；调用方只能拿到明确声明的稳定能力。

### Shared Hotspot Whitelist

- 允许共享修改的白名单仅包括：
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - `server/internal/pluginregistry/generated.go`
  - `server/cmd/graft/**`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `ai-plan/**`
- 其它目录默认视为 owned scope，不应被多个长期工作树共同持有。
- 若某个路径需要长期共享，必须先进入白名单治理文档，而不是默认开放。

### No Business Logic Backflow

- 一旦插件边界迁移完成，以下区域禁止重新接纳业务逻辑：
  - `server/internal/store/**`
  - `server/internal/ent/**`
  - `server/internal/app/**` 与其它 core runtime 包
- 禁止回流的内容包括：
  - repository
  - service
  - handler
  - migration
  - business schema
- 只有被明确认定为 platform/core capability 的内容才允许进入 core。
- 任何新增业务若无法落到 `plugins/<name>/**`，必须先更新设计真相，再决定是否属于 core。

### Third-Party Future Compatibility

- 当前不实现 Jenkins-style runtime plugin ecosystem。
- 但设计上预留以下未来稳定边界：
  - `plugin.Descriptor` 作为未来插件元数据基础
  - `internal/pluginapi/**` 与 `internal/contract/**` 作为未来唯一稳定公开 API
  - capability interface 作为未来跨插件调用边界
  - 插件不得依赖其它插件内部实现
- 当前明确不实现：
  - runtime plugin loading
  - external plugin marketplace
  - hot reload plugin lifecycle
  - sandbox execution
  - distributed plugin protocol

### AI Anti-Drift Constraints

- 本轮只解决并行开发冲突治理，不把需求外推成动态插件平台。
- 禁止引入：
  - runtime plugin loading
  - generalized IoC framework replacement
  - distributed plugin protocol
  - microservice decomposition
  - generalized service locator
- 必须保持：
  - 单体进程
  - compile-time wiring
  - deterministic startup
  - 当前运行行为基本不变

## Owned Scope Definition

- `core-owned`
  - `server/internal/app/**`
  - `server/internal/plugin/**`
  - `server/internal/pluginregistry/generated.go`
  - `server/internal/config/**`
  - `server/internal/logger/**`
  - `server/internal/database/**`
  - `server/internal/httpx/**`
  - `server/internal/container/**`
  - `server/internal/eventbus/**`
  - `server/internal/menu/**`
  - `server/internal/permission/**`
  - `server/internal/cronx/**`
  - `server/internal/redisx/**`
  - `server/internal/migration/**`
  - `server/internal/ent/**` 仅限 core-owned schema
- `shared-stable-boundary`
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
- `plugin-owned`
  - `server/plugins/<name>/**`
  - 包含该插件的 `contract/**`、`capability/**`、`service/**`、`store/**`、`storeent/**`、`ent/**`、`migrations/**`、`routes/**`、`tests/**`
- `generated-shared-hotspot`
  - `server/internal/pluginregistry/generated.go`
- `governance-owned`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `ai-plan/**`

## Public Interfaces And Structural Changes

- 新增 `plugin.Descriptor`
  - 定义插件元数据、依赖、builder 入口、capability 暴露元信息
- 新增 `plugin.Builder`
  - 负责 compile-time 装配插件私有依赖、注册 capability、返回 runtime plugin 实例
- 新增 `server/internal/pluginregistry`
  - 只承载 compile-time registry 生成入口与 `generated.go`
- `plugin.Context`
  - 只保留生命周期共性能力，不继续承载业务聚合 store factory
- 废弃 `server/internal/store.Factory` 作为业务插件总入口
- 迁移到插件私有边界：
  - `server/plugins/<name>/store/**`
  - `server/plugins/<name>/storeent/**`
  - `server/plugins/<name>/ent/**`
  - `server/plugins/<name>/migrations/**`
- `cmd/graft`、`serve`、`migrate`
  - 统一消费 compile-time registry，而不是手写 import 全部插件

## Execution Phasing

### Phase 1

- 落地 `plugin.Descriptor`
- 落地 `plugin.Builder`
- 落地 compile-time generated registry
- 把 `serve` / `migrate` wiring 改为消费 registry
- 保持现有业务代码位置基本不变，先只替换集中接线方式
- 增加 deterministic registry generation 与 plugin ordering 测试

### Phase 1 验收标准

- 新增一个插件后，不再手改 `serve.go` 插件列表
- `serve` 与 `migrate` 都从 compile-time registry 读取插件集合
- registry 生成结果 deterministic，可重复生成且无顺序漂移
- 不引入 runtime scan、runtime discovery、hot-load
- 可单独提交，不依赖 store / ent 大迁移

### Phase 2

- 引入插件私有 `store/**` 与 `storeent/**`
- builder 阶段装配插件私有 repository 依赖
- capability exposure 在 builder 阶段完成
- 移除 `server/internal/store.Factory` 的业务聚合职责
- 把 dev/reset 类能力改成 capability 或 dev hook，而不是 core 直接依赖插件内部函数

### Phase 2 验收标准

- `user`、`rbac` 不再依赖中心化业务 store factory
- 跨插件能力只能通过 capability interface 或 stable contract 获取
- core CLI 不再直接 import 插件内部 service/repository helper
- 一个插件的业务仓储修改不会要求碰另一个插件的 `storeent` 或中心化 `internal/store`
- 可单独提交，运行行为与现有接口保持兼容

### Phase 3

- 完成 Ent ownership 迁移
- 完成 migration ownership 迁移
- 清理跨插件 relation 和 schema backflow
- `internal/ent/**` 收缩到 core-owned schema
- 业务表迁移到各插件自有 `ent/**` 与 `migrations/**`

### Phase 3 验收标准

- 每个插件独立 ent generate，只写本插件 `ent/**`
- 每个 migration 只修改 owner 自己的表或 core-owned 表
- `rbac` 不再通过源码级 schema 依赖 `user` 内部实现
- 两个并行分支分别修改 `plugins/user/**` 与 `plugins/rbac/**`，合并时不再要求解决共享 schema、共享 migration、共享 store factory 冲突
- 可单独提交，必要时通过数据迁移顺序和兼容窗口平滑落地

## Standard Plugin Onboarding Flow

- 1. 定义插件 owned scope：`server/plugins/<name>/**`
- 2. 新建 `plugin.Descriptor` 与 Builder 声明，填入稳定插件名、版本、依赖、capability 暴露信息
- 3. 在插件目录内创建私有 `contract/**`、`service/**`、`store/**`、`storeent/**`、`ent/**`、`migrations/**`
- 4. 若需要跨插件协作，先定义 capability interface 或 stable DTO；优先放 `internal/pluginapi/**`，仅当确实跨插件稳定时才进入共享边界
- 5. 运行 registry generation，更新唯一集中接线产物 `server/internal/pluginregistry/generated.go`
- 6. 运行该插件自己的 ent generate 与 migration 生成流程
- 7. 运行最小直接测试、`graft validate backend --stage lint`、必要的 `go build ./cmd/graft`
- 8. 若修改了共享白名单路径，同步更新治理文档与 active tracking

## Shared Hotspots

- 刻意共享热点：
  - `server/internal/pluginapi/**`
  - `server/internal/contract/**`
  - `server/internal/pluginregistry/generated.go`
  - `server/cmd/graft/**`
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `ai-plan/**`
- 非刻意共享热点：
  - `server/plugins/<name>/**`
  - `server/plugins/<name>/store/**`
  - `server/plugins/<name>/storeent/**`
  - `server/plugins/<name>/ent/**`
  - `server/plugins/<name>/migrations/**`
- 解释规则：
  - 共享热点是“刻意受控共享”，不是“所有人都可以随意改”
  - 非白名单路径默认按 owner 管理，PR 设计与工作树边界都应围绕 owner 划分

## Future Compatibility Vs Not Now

### 未来三方插件兼容预留

- `plugin.Descriptor` 作为未来插件元数据基础
- compile-time Builder 模型作为未来扩展成“更正式插件接入协议”的前置抽象
- `internal/pluginapi/**` 作为未来唯一稳定 capability API 面
- `internal/contract/**` 作为未来稳定 typed contract 面
- capability interface 作为未来跨插件调用边界
- 插件内部实现与稳定公开边界严格分离，避免未来无法抽象出“外部插件接入契约”

### 当前不实现

- runtime plugin loading
- runtime plugin discovery
- external plugin marketplace
- hot reload lifecycle
- sandboxed plugin execution
- distributed plugin protocol
- microservice split
- generalized reflection-based plugin container
- generalized service locator

## Assumptions And Defaults

- 默认继续使用 compile-time registration，不引入第二套启动模型。
- `generated.go` 是唯一允许的集中接线文件；除该文件外，不再接受手写集中插件列表。
- 未来若出现第三方插件需求，也必须先复用 `Descriptor + pluginapi + contract + capability` 这条稳定边界，不允许绕过它直接暴露内部实现。
- 若某个功能无法明确落在 `core-owned` 或某个 `plugin-owned` 范围，默认先判为插件能力，而不是回流到 core。
