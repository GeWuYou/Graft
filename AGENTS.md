# AGENTS.md

This document is the single source of truth for coding behavior in this repository.

All AI agents and contributors must follow these rules when writing, reviewing, or modifying code in `Graft`.

## 1. Project Intent

`Graft` is a composable admin platform, not a single-purpose business app and not an AI product.

Primary goal:

* build a backend platform that can add new capabilities quickly through plugins

Secondary goals:

* keep `server` and `web` module boundaries stable
* make repetitive admin modules easy to scaffold
* keep the codebase friendly to AI-assisted development

Do not optimize for:

* early dynamic plugin hot-loading
* third-party plugin marketplace in v1
* heavyweight framework abstractions without clear need

## 2. Source of Truth

Before changing code or structure, read the relevant documents in `ai-plan/`.

Authoritative documents:

* [ai-plan/design/项目设计.md](ai-plan/design/项目设计.md)
* [ai-plan/design/插件与依赖注入设计.md](ai-plan/design/插件与依赖注入设计.md)
* [ai-plan/design/前端架构设计.md](ai-plan/design/前端架构设计.md)
* [ai-plan/design/契约治理与魔法值治理规范.md](ai-plan/design/契约治理与魔法值治理规范.md) when the task changes
  typed contracts, magic-value governance, contract lifecycle, ownership, compatibility, drift handling, or shared
  `server` / `web` semantics
* [ai-plan/design/代码注释与模块文档规范.md](ai-plan/design/代码注释与模块文档规范.md) when the task changes
  code comments, package docs, module README rules, or AI documentation behavior
* [ai-plan/design/TDesign-MCP-辅助开发规范.md](ai-plan/design/TDesign-MCP-辅助开发规范.md) when the task changes
  TDesign Vue Next pages, components, styles, or frontend AI-assisted development workflow
* [ai-plan/roadmap/MVP实施计划.md](ai-plan/roadmap/MVP实施计划.md)
* [ai-plan/design/AI任务追踪与恢复设计.md](ai-plan/design/AI任务追踪与恢复设计.md) when the task changes
  tracking, recovery, or documentation-governance rules

If code and docs diverge, update the docs first or in the same change.

When a task changes architecture, plugin boundaries, lifecycle semantics, or frontend module conventions, the related
`ai-plan/design/` or `ai-plan/roadmap/` document must be updated before the task is considered complete.

## 3. Repository Terms

Use these names consistently in code discussions, plans, reviews, and task breakdowns:

* `server` means the backend project and its runtime, plugin, and infrastructure code
* `web` means the frontend project and its Vue 3 admin shell and feature modules
* `core` means true infrastructure owned by the platform runtime
* `plugin` means business capability registered into the platform through the plugin system

Do not use vague wording that blurs repository boundaries when a task is really about `server`, `web`, `core`, or a
plugin.

## 4. Environment Capability Inventory

Before choosing runtimes, package managers, or CLI tools:

* first read `.ai/environment/tools.ai.yaml` if it exists
* use `.ai/environment/tools.raw.yaml` only when the AI-facing inventory is missing or insufficient
* prefer repository-relevant installed tools over assumptions about what is available on the system
* if `.ai/environment/` marks a cross-environment exception such as host Windows Bun for `web`, follow that exception
  instead of defaulting to the current WSL shell toolchain
* if a change affects repository toolchain expectations or environment guidance, refresh the `.ai/environment/`
  inventory in the same change instead of leaving generated environment truth stale

If the environment inventory does not exist yet:

* inspect the repository for the actual toolchain before making assumptions
* report the missing inventory when it materially affects repeatability
* do not create fake dependencies on inventory files that are not present in the repository

When `.ai/environment/` exists:

* treat `tools.raw.yaml` and `tools.ai.yaml` as generated repository truth, not hand-maintained notes
* keep repository startup skills aligned with the inventory read order

### 4.1 Startup Governance

`AGENTS.md` is the only authoritative startup-governance source in this repository.

Other files may point to recovery materials, environment facts, or skill entrypoints, but they must not define a
second boot chain, a second receipt format, or a second set of startup gating rules.

Every repository task starts in one of these states:

* `unbooted`
  * no startup preflight has been completed for the current task turn
* `preflighted`
  * the startup preflight has completed and the task may enter repository recovery or direct execution
* `governance-lost`
  * the current task turn no longer has a trustworthy startup state and must rerun preflight before substantive work

The minimum startup preflight is:

1. confirm the repository root
2. read this root `AGENTS.md`
3. read `.ai/environment/tools.ai.yaml` when it exists; use `.ai/environment/tools.raw.yaml` only when the AI-facing
   summary is missing or insufficient
4. classify the task as one of:
   * `server`
   * `web`
   * `cross-boundary`
   * `docs/automation`
5. decide whether the current turn needs recovery context from `ai-plan/public/README.md`

The minimum startup receipt is:

* `governance source`
  * the root `AGENTS.md`
* `task class`
  * one of `server` / `web` / `cross-boundary` / `docs/automation`
* `recovery source`
  * `none`, `parent topic`, or `subtopic`

Fail-closed startup rules:

* do not start implementation, plan finalization, validation conclusions, or subagent delegation without the startup
  receipt for the current task turn
* if resume, restart, topic switching, or context loss makes the current startup state unclear, move to
  `governance-lost` and rerun the startup preflight
* recovery state does not replace startup state; reading tracking or trace files without the startup receipt is not a
  valid boot path
* lightweight lookups may happen before the receipt, but repository-level conclusions, plans, edits, and subagent
  delegation must wait until the receipt is established

Resume and restart rules:

* `continue`, `resume`, `restart`, and similar prompts must rerun the startup preflight for the current turn
* only after that preflight may the agent read `ai-plan/public/README.md` and the mapped tracking or trace files
* restoring a topic recovery point does not mean repository governance has been restored

Subagent inheritance rules:

* the main agent completing startup preflight does not mean a subagent already knows repository governance
* every subagent task must carry a minimum inherited context package containing:
  * `governance source`
    * the root `AGENTS.md`
  * `task class`
  * `recovery source`
  * `owned scope`
* a subagent must not be launched with only an objective or file target; without the inherited context package the
  task remains `governance-lost`

## 5. Repository Skills

Repository-maintained skills live under `.agents/skills/`.

Prefer the repository skills below when their trigger matches the task:

* `graft-boot`
  * use for short startup prompts, resume prompts, or when the first step should be to run the startup preflight
    defined in `4.1 Startup Governance` and then enter repository recovery when needed
* `graft-multi-agent-batch`
  * use when the user explicitly wants subagent delegation or when the work cleanly splits into disjoint parallel slices
* `graft-pr-review`
  * use when the task depends on the GitHub PR for the current branch, especially to extract AI review findings,
    failed checks, MegaLinter warnings, or failed test signals before local verification
* `graft-plugin-scaffold`
  * use when adding a new `server` plugin or shaping a plugin before implementation
* `graft-commit`
  * use when the current task slice is ready to commit and the user explicitly wants the agent to classify ownership,
    verify scope, choose a compliant Conventional Commit message, and create a scoped git commit
* `graft-web-module-scaffold`
  * use when adding a new `web` feature module aligned with backend plugin semantics
* `graft-validation-runner`
  * use when choosing the smallest correct validation for `server`, `web`, or cross-boundary work

If a repository skill and this document diverge, follow `AGENTS.md` first and update the skill in the same change.

## 6. Locked Technical Choices

### 6.1 Server

* Go
* Gin
* Ent
* PostgreSQL
* Viper
* Zap
* Casbin
* robfig/cron

### 6.2 Web

* Vue 3
* TypeScript
* Vite
* TDesign Vue Next
* Pinia
* Vue Router
* Axios
* UnoCSS

### 6.3 Architecture

* plugin-oriented backend
* lightweight DI / service registry
* no heavyweight IoC container
* compile-time plugin registration for v1

Do not switch to React, Naive UI, or a full IoC framework unless the project docs are explicitly revised first.

## 7. Architecture Rules

### 7.1 Server Core

Core runtime owns:

* config
* logger
* database
* HTTP server
* migration runner
* event bus
* permission registry
* menu registry
* cron registry
* plugin manager
* service container

Core runtime surface must stay explicit and small.

Only documented runtime surfaces such as config, HTTP, migration, event, permission, menu, cron, plugin, service
container, and repository CLI entrypoints may own platform-level startup behavior. Do not hide new runtime surfaces in
unrelated packages, starter code, or ad-hoc background initialization.

Business logic must live in plugins.

Do not put business-specific behavior into the platform core.

### 7.2 Plugins

Every backend plugin should follow the same model:

* declare metadata
* declare dependencies
* register routes, menus, permissions, migrations, jobs, and public services in `Register`
* start runtime behavior in `Boot`
* release resources in `Shutdown`
* keep the plugin runtime surface explicit; if a route, menu, permission, migration, job, event subscription, or
  public service exists at runtime, its existence and ownership must be traceable to the plugin lifecycle
* `Register` must declare the complete runtime surface and cross-plugin/public contracts before `Boot` starts side
  effects
* `Boot` may only start behavior that was already declared or wired through the documented lifecycle; do not create
  hidden routes, jobs, listeners, or long-lived goroutines from package globals or ad-hoc side paths
* `Shutdown` must release everything `Boot` started, including timers, subscriptions, goroutines, and external handles

Plugins must depend on public interfaces, not on another plugin's internal implementation.

Cross-plugin contracts belong in a stable package such as `internal/pluginapi`.

### 7.3 Dependency Injection

The DI layer is intentionally small.

Allowed responsibilities:

* register singleton providers
* resolve services
* expose plugin public services
* close registered resources

Disallowed responsibilities:

* reflection-heavy auto wiring
* package scanning
* struct tag injection
* hidden magic construction
* complex scope systems

If a design requires implicit behavior to be understandable, it is too complex for this repo.

The container or service registry is a runtime composition boundary, not a general-purpose service locator.
`Resolve` belongs in explicit wiring, plugin lifecycle adapters, or other narrow composition boundaries. Do not spread
ad-hoc resolution into handlers, services, repositories, pages, stores, or other business paths just because the
container is reachable.

### 7.4 Web

Frontend is a platform shell plus feature modules.

Expected structure:

* `web/src/app`
* `web/src/layouts`
* `web/src/pages`
* `web/src/modules`
* `web/src/components`
* `web/src/api`
* `web/src/contracts`
* `web/src/stores`
* `web/src/router`

Rules:

* use TDesign as the primary component system
* avoid mixing multiple UI libraries
* keep the real `web/` application as the only frontend runtime surface; do not treat `web/ai-libs/`, starter trees,
  or demo shells as a parallel runtime baseline
* keep new modules aligned with `menu + route + page + api + permission`
* preserve module lifecycle clarity: `app` owns bootstrap and application wiring, `layouts` own shell chrome,
  `modules` own feature behavior, `contracts` own platform-level frontend stable contracts, and `components` stay
  reusable rather than becoming hidden feature containers
* keep feature boundaries inside explicit module surfaces; do not park long-lived module state, routes, permissions,
  or API semantics in shell-level directories just because a starter or demo structure already has a placeholder
* use dynamic menus driven by backend data
* keep shared state in stores and keep page-local state inside the page or module
* frontend governance baseline must treat `TypeScript strict`, `format:check`, `ESLint`, `Stylelint`, `Vitest`,
  `Husky + lint-staged`, and `commitlint` as one consistent quality gate instead of optional local preferences
* the repository `web` toolchain now treats `bun run check` as the mandatory frontend completion entrypoint; when a
  `web` task reaches a completion-state milestone such as 功能完成、任务完成, or 准备合并, it must run the full chain in the explicit order
  `format:check -> typecheck -> lint -> stylelint -> test:run -> build`
* intermediate frontend iteration may still use the smallest direct validation that covers the touched area, but the
  full `bun run check` chain is required before a `web` task is considered complete
* when the repository is used from WSL, all `web` install, validation, build, preview, and dev commands must run
  through the configured host Windows Bun instead of the WSL Bun binary
* do not refresh or regenerate `web/node_modules` from WSL Bun when host Windows Bun is the active frontend package
  manager, because mixed Bun environments can leave Windows IDE and `npm run dev` flows unusable
* frontend completion defaults to zero warning across `typecheck`, `lint`, `stylelint`, `test:run`, and `build`;
  do not treat build-time Vite/Rollup warnings, chunk warnings, or known third-party package warnings as acceptable
  completed-state noise
* JetBrains Inspection, TS language-service suggestions, and local spell-check output are local assistance by default,
  not repository completion blockers, unless the same rule is enforced through the documented CLI quality chain or a
  repository document explicitly promotes that IDE rule into project truth
* the only allowed warning exception is a controlled exception recorded in the active tracking document with the
  warning source, user-visible or engineering impact, why the current slice cannot safely eliminate it, and the next
  planned cleanup action
* `web/ai-libs/` is a local reference area for starter configuration and TDesign usage patterns, not a runtime
  dependency or a source of truth to be copied wholesale into `web`
* when reusing ideas from `web/ai-libs/`, keep only the governance or component patterns that fit Graft, and do not
  directly transplant its mock layer, frontend-only permission model, tabs-router behavior, or page scaffolding
* do not spread `any` or `as any` through page and module code to bypass `strict`; when unavoidable, confine such
  escapes to explicit adapter, client, schema, or migration-compatibility boundaries and keep the unsafe surface small
* when generating, modifying, or reviewing TDesign Vue Next code with AI assistance, query TDesign MCP or official
  TDesign docs before relying on component props, events, slots, DOM structure, or changelog details
* configure TDesign MCP on the active AI coding client, with Codex as the default AI coding entrypoint for this
  repository; Rider MCP setup is only required when using Rider AI Assistant for frontend code generation

## 8. Naming and Boundary Conventions

### 8.1 Server

* plugin names are short, stable, lowercase
* exported cross-plugin interfaces should be capability-oriented
* do not expose repositories as public plugin APIs
* permission codes should be namespaced, for example `user.read`, `user.create`
* config keys should be namespaced by plugin
* public cross-plugin return types should be stable capability DTOs, not raw database models
* detailed `server` Go constraints for file/package/type naming, layering, context propagation, transactions,
  concurrency, resource lifecycle, API/DTO boundaries, config loading, runtime wiring, auth handling, logging, and
  AI-generated code are defined in `9. Go 代码组织与命名规范`

### 8.2 Web

* route names should be stable and unique
* page components should reflect module intent, not UI widget names
* stores should be reserved for shared state, not page-local form state
* CRUD page layouts should stay consistent across modules
* module pages should align with backend permissions and menu metadata instead of inventing parallel frontend-only
  access rules

### 8.3 Contract Governance

High-risk runtime literals must follow `ai-plan/design/契约治理与魔法值治理规范.md`.

For this repository, contract means stable values or typed boundaries that callers may depend on, including:

* permission code
* route name and special route path
* storage key
* header name and auth scheme
* error code
* message key
* event name
* config key / env key
* feature flag key
* shared status enum or other values that affect `server` / `web` behavior across module boundaries

Rules:

* before adding a new high-risk contract, search the existing module contract, platform contract, and stable
  cross-boundary contract first; reuse the canonical definition instead of creating a parallel one
* do not create new global `constants` or `enums` dump files to satisfy contract governance
* `server` high-risk string contracts should prefer capability-oriented typed string boundaries when they cross plugin,
  runtime, or registration surfaces
* `server` route path contracts should keep one canonical truth; prefer `group path + route fragment` over parallel
  `full path` constants, and compose full paths locally when needed
* do not add route aliases such as `Foo` + `FooPath` + `FooRoute` for the same server route semantic
* `web` high-risk contracts should prefer literal unions or other explicit typed boundaries at router, storage, request,
  API, and module contract surfaces instead of scattering raw strings
* any new or changed high-risk shared contract must have an explicit owner boundary and lifecycle state
  (`experimental` / `stable` / `deprecated` / `removed`); deprecated contracts also need a replacement or cleanup plan
* alias compatibility is temporary only; do not treat alias contracts as permanent second truths
* if a contract rename, removal, or compatibility change affects both `server` and `web`, update the contract design
  doc in the same change instead of relying on code comments or PR text alone

Phase-1 expectations:

* prioritize canonical ownership, naming, typed boundary, and lifecycle clarity before automation exists
* do not claim contract drift, deprecated usage, or duplicate-semantic checks passed unless the repository entrypoint
  for that check exists and was actually run
* when automation has not landed yet, report the documentation or registry alignment performed and the exact validation
  gap instead of implying the governance work is complete by default

## 9. Go 代码组织与命名规范

This section is the detailed source of truth for hand-written Go code under `server`.

It applies to runtime, plugins, service/usecase, repository/store, Gin handlers, middleware, config, Ent schema,
Atlas migration hand-written boundaries, and related tests.

Generated code, third-party code, and migration artifacts themselves may follow repository-specific exemptions, but
their hand-written wrappers, adapters, schema truth, runtime wiring, and public boundaries must still follow this
section.

### 9.1 文件命名规范

* Go 文件名必须使用小写。
* 多个单词必须使用下划线分隔，例如 `user_role.go`、`refresh_token.go`、`auth_middleware.go`、
  `password_policy.go`。
* 禁止新增 `userrole.go`、`refreshtoken.go` 这类连续拼接且可读性差的文件名。
* 测试文件必须使用 `*_test.go`。
* Ent schema 文件必须与 schema 类型语义对应，例如 `user.go -> User`、`role.go -> Role`、
  `user_role.go -> UserRole`、`role_permission.go -> RolePermission`。
* 文件名优先表达业务实体、边界或职责，不得把 `misc.go`、`common.go`、`utils.go`、`helper.go`
  当作默认落点，除非该 package 本身就是边界清晰且被仓库显式认可的工具包。
* 一个文件只承载一个主要职责；当 handler、service、schema、middleware、DTO、store 已经跨越多个独立关注点时，
  必须拆文件，而不是继续堆入同一文件。

### 9.2 包命名规范

* package 名必须短、小写、无下划线。
* package 名必须表达领域或职责，例如 `auth`、`rbac`、`database`、`config`、`httpx`。
* 禁止使用 `manager`、`helper`、`common`、`utils` 作为万能包名。
* package 名不得重复父目录已经表达的语义。
* 不得为了迁就文件名而拆包；优先保证 package 内聚、边界清晰和依赖单向。
* 跨插件稳定接口应继续放在 `internal/pluginapi` 或等价稳定边界，不得因为局部实现方便把接口散落到实现包。

### 9.3 类型命名规范

* 导出类型必须使用 `PascalCase`，例如 `UserService`、`TokenManager`、`PasswordPolicy`。
* 非导出类型必须使用 `lowerCamelCase`，例如 `userRepository`、`loginRequest`。
* 类型名必须表达业务语义，禁止新增 `BaseManager`、`CommonService`、`DataHandler` 这类泛化命名。
* 接口命名应优先描述行为，例如 `UserStore`、`TokenIssuer`、`PasswordHasher`、`PermissionChecker`。
* 不得为了“面向接口”而预先抽象接口；只有存在多实现、测试隔离或跨边界依赖时才定义接口。
* 接口优先放在消费方 package，而不是实现方 package。
* 跨插件公开类型必须优先返回稳定 capability DTO，而不是暴露数据库模型、Ent entity 或 repository 细节；该约束与
  `8.1 Server` 的公开边界规则一致，本节负责把它落实到日常 Go 代码。

### 9.4 方法与函数命名规范

* 函数名必须使用动词或动宾结构，例如 `CreateUser`、`IssueTokenPair`、`ValidatePassword`、
  `LoadConfig`。
* 返回 `bool` 的方法优先使用 `Is`、`Has`、`Can`、`Allow` 前缀，例如 `IsExpired`、`HasRole`、
  `CanAccess`。
* 构造函数优先使用 `NewXxx`。
* 打开外部资源的函数可使用 `Open`，但必须在注释、返回契约或调用约定中明确 `Close` 所有权。
* 事务辅助函数命名必须体现事务语义，例如 `WithTx`、`RunInTx`。
* 不得在语义不清的上下文里使用 `Do`、`Handle`、`Process`、`Run` 这类泛化名称。
* 私有函数也必须命名清晰，不得因为非导出就随意缩写。

### 9.5 结构体字段规范

* 导出字段使用 `PascalCase`，非导出字段使用 `lowerCamelCase`。
* 字段名不得使用无意义缩写。
* 依赖字段命名应体现角色，例如 `db`、`logger`、`users`、`tokens`、`checker`。
* 配置结构体字段应表达配置语义，不得直接照抄环境变量原文命名。
* JSON 结构体字段必须显式写 tag。
* API request/response、cross-plugin DTO、持久化模型字段的语义边界必须可追踪，不能混成“同一结构体到处复用”。
* Ent schema 字段命名必须同时保持数据库语义、Go 语义和 API 语义可追踪。

### 9.6 Context 规范

* 请求链路必须优先透传 `context.Context`。
* `context.Context` 必须作为函数第一个参数。
* 不得在请求链路内部随意使用 `context.Background()`。
* `repository/store/database/redis/http client` 必须接收并透传 `context.Context`。
* 超时、取消、`traceId`、`requestId` 等请求级元数据必须通过 `context` 传播。
* 除启动期与测试代码外，禁止主动丢弃上游 `context`。
* 来源于请求链路的 goroutine 必须考虑 `context cancel`，并把退出语义设计为显式可追踪。
* 不允许在深层依赖中重新创建脱离请求链路的新 `context`。
* `context.Value` 只允许承载请求级元数据，不得作为 service、repository、config、logger 等一般依赖的万能参数容器。
* 该规则统一约束 `Gin -> service/usecase -> repository/store -> database/redis/http client` 的上下文传播边界。

### 9.7 API 与 DTO 规范

* handler 层禁止直接暴露 Ent entity 到 HTTP API。
* request/response DTO 必须与数据库模型解耦。
* API request/response 结构体必须显式定义。
* DTO 命名必须体现语义，例如 `LoginRequest`、`LoginResponse`、`CreateUserRequest`、
  `UserProfileResponse`。
* 禁止把 `ent.Entity` 或其聚合结果直接作为 JSON response 返回。
* API 层不得泄漏数据库字段语义、内部外键、Ent edge 细节或 schema 中间态。
* handler 不得依赖数据库 schema 作为 API 真值。
* API response 必须通过统一响应结构输出。
* 时间、权限、状态等字段必须保持 API 语义稳定，不得随底层 schema 细节漂移。
* 不允许为了图省事把 `map[string]any` 作为主响应结构。

### 9.8 配置规范

* 配置统一通过 `config` 模块加载。
* 不允许在业务代码中直接 `os.Getenv`。
* 配置解析、默认值、校验必须集中管理。
* 配置结构体必须表达业务语义，而不是环境变量原文。
* 环境变量名仅作为外部输入边界，不得在业务分支中反复传播。
* `secret`、`token`、`password` 类配置不得写死默认值。
* 配置缺失时必须明确 fail fast。
* runtime startup 必须明确打印关键配置阶段，但不得打印敏感值。
* 配置结构体必须避免“万能 Config”；应按 runtime、plugin、capability 边界拆分。
* 不允许把运行时状态、缓存句柄、请求上下文或其它动态值塞进 config。

### 9.9 Runtime Wiring 与依赖注入规范

* 依赖关系必须显式 wiring。
* 不允许隐藏全局单例。
* 不允许通过 `init()` 偷偷注册运行时依赖。
* runtime、plugin、service、store 依赖必须可追踪。
* service 的依赖必须通过构造函数注入。
* 不允许在业务代码内部临时 `new` 基础设施依赖，例如 logger、database、redis、event bus、scheduler。
* `logger`、`database`、`redis`、`event bus`、`scheduler` 等共享资源必须由 runtime 管理生命周期。
* plugin 不得绕过 runtime 直接控制其他 plugin 内部状态。
* wiring 必须保持单向依赖。
* 禁止滥用 service locator；仓库现有容器只用于显式注册与解析边界，不能把“到处 Resolve”当作普通业务编码模式。
* package `init` 不得承载复杂初始化逻辑。
* 本小节与 `7.3 Dependency Injection` 一致，只补强日常编码中的显式装配约束，不引入新的容器抽象。

### 9.10 安全与鉴权规范

* 不允许在 handler 内手写散乱权限判断。
* 鉴权逻辑必须统一通过 middleware、auth service 或 permission checker。
* JWT、refresh token、session、secret 必须统一管理。
* token 校验失败必须返回统一错误语义。
* 不允许通过字符串硬编码角色逻辑散落业务代码。
* 权限判断必须显式可追踪。
* 不允许把敏感内部错误直接返回前端。
* 默认拒绝未知权限。
* 所有认证相关时间必须统一使用 UTC 或仓库统一时区策略。
* refresh token 必须具备可吊销能力。
* logout 后不得继续接受旧 refresh token。
* 该规则用于约束当前 auth/RBAC MVP 实现边界，不得借机引入额外 framework 级鉴权抽象。

### 9.11 事务规范

* 事务边界优先放在 `service/usecase` 层。
* `repository/store` 默认不主动开启事务，除非该 package 的唯一职责就是显式事务执行器。
* handler 不得直接编排数据库事务。
* 同一个业务事务中的 `repository/store` 调用必须共享同一 `tx`。
* `Rollback` 必须通过 `defer` 保证。
* `Commit` 成功后不得继续使用旧 `tx`。
* 不允许隐藏事务边界。
* 不允许在 `repository` 内部偷偷开启新事务覆盖上层事务。
* 跨 `repository` 的事务协调必须由 `service/usecase` 负责。
* 事务开始、提交、回滚路径必须可追踪，并与资源生命周期注释规则保持一致。

### 9.12 错误处理规范

* Go 代码必须显式处理 `error`，禁止忽略关键 `error`。
* 包装错误必须使用 `fmt.Errorf("context: %w", err)`。
* 错误上下文必须说明当前操作，例如 `load config: %w`、`connect redis: %w`、
  `apply atlas migrations: %w`。
* 不得为了通过编译而吞掉错误、返回无理由的 `nil`，或用空分支掩盖失败路径。
* 除启动期不可恢复的编程错误外，底层函数不得直接 `panic`。
* HTTP handler 中不得直接泄漏底层数据库错误或内部依赖错误给前端。
* token、密码、secret、数据库内部错误等敏感失败细节不得直接拼进外部错误响应。

### 9.13 并发与资源生命周期规范

* 新增 goroutine 必须明确生命周期与退出条件。
* 禁止无边界后台 goroutine。
* goroutine 必须可取消、可回收、可观测。
* channel 的创建方负责 `close`。
* 使用 `time.Ticker`、`time.Timer` 后必须 `Stop`。
* 不允许无限重试循环且无 `sleep`、`backoff` 或 `context cancel`。
* 并发共享状态必须显式同步。
* 优先使用 `context` 控制协程退出。
* 不允许 silently `recover` panic 后继续运行。
* `WaitGroup` 必须保证 `Add` 与 `Done` 成对出现。
* `db tx`、`rows`、`redis pubsub`、`http response body`、`file`、`ticker`、`timer` 等资源句柄必须明确
  `Close`、`Stop`、`cancel`、`Rollback` 生命周期。
* `Close`、`cancel`、`Rollback` 不得遗漏。

### 9.14 日志规范

* `server` 禁止直接使用 `fmt.Println`、`log.Println` 输出业务日志。
* 后端日志统一通过日志模块输出。
* 请求链路日志必须包含 `traceId`、`requestId`。
* 错误日志必须携带上下文信息。
* `debug` 日志不得污染生产错误日志。
* 不允许记录 `password`、`token`、`jwt`、`refresh token`、`secret`、`cookie`、
  `authorization header` 等敏感信息。
* 日志字段命名必须稳定一致。
* `panic/recover` 日志必须带 stack 信息。
* 不允许在循环、高频路径或热路径中打印大量 `info/debug` 日志。
* HTTP access log 与业务错误日志职责必须分离。
* migration、bootstrap、runtime 初始化日志必须清晰标识阶段。

### 9.15 注释规范

* 导出类型、导出函数、导出常量必须写 GoDoc 注释。
* GoDoc 第一句必须以被注释标识符开头。
* 注释必须说明职责、所有权、边界或副作用，不得复述代码。
* 非导出代码只有在存在复杂业务规则、资源生命周期、并发、安全边界、事务语义或兼容约束时才写注释。
* 不要求给每个入参/返回值机械写注释；只有语义不明显时才补充。
* Context 边界、事务边界、goroutine 退出条件、`Open/Close` 所有权、敏感日志限制等复杂路径，应按
  `18.1 Server Documentation` 与 `18.3 Inline Comment Rules` 写清楚边界和副作用。

### 9.16 代码组织规范

* handler 只负责 HTTP 参数解析、调用应用服务、返回响应。
* `service/usecase` 负责业务流程编排。
* `repository/store` 负责数据访问。
* `config` 负责配置加载和校验。
* `middleware` 负责认证、request id、日志上下文等横切逻辑。
* 不允许在 handler 直接堆复杂业务逻辑。
* 不允许 `repository` 反向依赖 handler 或 service。
* 不允许为了方便引入循环依赖。
* 不允许在业务逻辑中直接把数据库、Redis、HTTP client、scheduler、event bus 的细节扩散到 API 边界。
* plugin 内部边界应继续优先复用现有 `runtime/plugin/service/store` 组织方式，不得为局部需求临时发明第二套分层。

### 9.17 Ent 相关规范

* Ent schema 类型必须使用单数业务名。
* 多词 schema 文件必须使用下划线，例如 `user_role.go`。
* `Mixin`、`Hook`、`Policy`、`Edge`、`Field` 应按职责拆分，避免单文件无限膨胀。
* schema、field、edge 命名必须服务于查询可读性。
* 迁移相关变更必须考虑 Atlas 校验与 hash。
* 修改 migration 文件后必须重新执行 `atlas migrate hash`，除非只是新增尚未应用 migration。
* handler、service、cross-plugin public API 不得把 Ent schema 或 generated entity 当作外部契约真值。

### 9.18 测试规范

* 测试文件必须使用 `*_test.go`。
* 测试函数命名必须使用 `TestXxx`。
* 表驱动测试变量优先命名为 `tests`。
* 测试用例字段至少包含 `name`。
* 重要错误分支必须覆盖。
* 涉及生命周期、权限、事务、资源释放、token 轮换、context cancel、handler fail-closed 的回归路径必须补测试。
* 当变更涉及可疑 API 使用、并发、错误包装、Context 传播或资源句柄约定时，应优先补充 `go vet` 可覆盖的直接校验。
* 变更 `server` Go 代码后，最小直接校验范围必须符合 `12.1 Server Validation`。
* 完成态默认必须满足 `gofmt`、`golangci-lint run`、`go test`、`go build ./cmd/graft` 与仓库统一 backend
  completion entrypoint 的要求；`go vet` 不是默认完成态统一入口，但在当前切片有直接覆盖价值时应显式纳入；
  具体命令顺序和完成态入口以 `12.1 Server Validation` 为准。

### 9.19 AI 生成代码约束

* AI 生成代码必须优先复用现有 `runtime`、`plugin`、`service`、`store` 边界。
* AI 不得把 starter/demo/reference 目录升级为并行 runtime surface、并行模块基线或第二套架构真值。
* AI 不得擅自新增 framework、ORM、DI container、logging framework。
* AI 不得无理由扩大 abstraction layer。
* AI 不得为仅供 Gin 注册或测试拼接使用的 server 路由常量引入无语义类型别名、`String()` 包装或并行 full-path
  常量集。
* AI 不得借“临时过渡”之名绕开既有 module lifecycle、feature boundary、plugin lifecycle 或 validation
  entrypoint 约束。
* AI 不得新增“未来可能会用到”的接口或扩展点。
* AI 不得生成未使用代码。
* AI 不得生成死代码、占位 `TODO` 或伪实现。
* AI 不得为了通过编译吞掉错误。
* AI 必须优先保持现有架构一致性。
* AI 修改高风险契约时必须先复用已有 canonical contract，不得为局部方便重复定义同义常量、平行 alias 或新的
  `constants` 垃圾桶文件。
* AI 不得在新代码中继续引入已标记 `deprecated` 的 contract；如兼容窗口尚未结束，只能在兼容桥接层保留旧 contract。
* AI 修改 migration 时必须考虑 `atlas migrate hash`。
* AI 修改 `server` 代码后必须满足 `gofmt`、`go test ./...`、`go build ./cmd/graft`、`golangci-lint run`，
  并与 `12.1 Server Validation` 的统一入口保持一致。
* AI 不得通过修改规则、跳过测试、删除校验来绕过完成态。

### 9.20 禁止事项

* 禁止新增 `userrole.go`、`refreshtoken.go` 这类不可读文件名。
* 禁止新增 `common.go`、`utils.go`、`helper.go` 作为垃圾桶文件。
* 禁止在 handler 中写大段业务逻辑。
* 禁止随意引入全局变量或隐藏单例。
* 禁止忽略 `Close`、`cancel`、`Rollback` 等资源释放。
* 禁止为了通过编译而吞掉 `error`。
* 禁止无理由扩大接口、抽象层或 package 边界。
* 禁止在没有需求的情况下引入框架级复杂设计。

## 10. Implementation Priorities

When building new functionality, prefer this order:

1. stabilize docs and interfaces
2. implement platform primitives
3. implement a minimal end-to-end slice
4. add breadth only after the extension path is proven

For v1, prioritize:

* user
* rbac
* audit
* scheduler

Do not start Docker, SSH, monitor, or workflow plugins before the core extension path is stable.

## 11. Execution Rules

### 11.1 Module Placement

When asked to add a new capability:

* first identify whether it belongs in `server/core`, a `server` plugin, or a `web` feature module
* default to a plugin unless the capability is true infrastructure
* default to a `web/src/modules/<name>` entry path unless the page is a shell-level concern
* define the capability's runtime surface and lifecycle owner before implementation; entrypoints, menus, routes,
  permissions, jobs, public services, and boot/shutdown responsibilities must all have one clear home
* define menu, route, permission, API, and public service boundaries before writing code

### 11.2 Explicitness

When unsure:

* choose the more explicit implementation
* choose the narrower public interface
* keep the next contributor's mental load low
* prefer direct construction and visible wiring over hidden framework behavior
* prefer preserving the current repository architecture over introducing a second baseline, second shell, or second
  validation contract for temporary convenience

### 11.3 New Dependencies

When asked to introduce a new dependency:

* justify why the existing stack is insufficient
* prefer smaller, explicit libraries
* avoid adding abstractions that hide control flow
* reject dependencies that materially weaken plugin boundaries or increase hidden runtime magic without clear benefit

## 12. Validation Rules

Every completed task must pass at least one validation that directly covers the changed code before it is considered
done.

### 12.1 Server Validation

For `server` changes:

* use pinned `golangci-lint v2.12.2` as the unified backend lint runner; do not use `latest`
* the repository now treats `graft validate backend` as the mandatory backend completion entrypoint for any `server`
  task that is being closed, handed off, or prepared for merge
* when a `server` task reaches a completion-state milestone such as 功能完成、任务完成, or 准备合并, it must run the
  full backend quality chain in the explicit order `golangci-lint run -> go test (smallest direct scope) -> go build
  ./cmd/graft -> graft validate smoke` when runtime startup validation is needed
* intermediate backend iteration may still use the smallest direct validation that covers the touched area, but the
  full backend quality chain is required before a `server` task is considered complete
* agent, local development, and CI must use the same backend entrypoint and the same pinned lint version instead of
  maintaining ad-hoc shell chains or a second lint parameter set
* backend completion defaults to zero unresolved lint issues across the configured `golangci-lint` set; do not treat
  known lint findings as acceptable completed-state noise
* lint issues in directly affected `server` code are blocking by default
* a backend lint issue may be retained only as a controlled exception recorded in the active tracking document with the
  issue source, user-visible or engineering impact, temporary retention reason, and next cleanup action
* directly affected `server` code that violates `9. Go 代码组织与命名规范` is not considered validation-complete,
  even when the code still builds
* run the smallest `go test` scope that still covers the touched packages when tests exist
* run `go build ./cmd/graft` as the default backend compile gate, and widen the build scope only when the current
  change materially affects other `cmd/*` entrypoints
* prefer wider validation such as `go test ./...` or `go build ./...` when the task changes shared abstractions, plugin
  contracts, lifecycle code, dependency resolution, or startup wiring

### 12.2 Web Validation

For `web` changes:

* run the repository's actual frontend validation command once it exists
* the repository now requires host Windows Bun `bun run check` as the default full validation entrypoint for any `web`
  task that is being closed, handed off, or prepared for merge
* when the repository runs from WSL, interpret frontend `bun` validation commands as host Windows Bun commands unless
  the environment inventory explicitly says otherwise
* the standard `web` quality chain should include `format:check`, type checking, lint, stylelint, unit tests, and
  production build in that order
* a completed `web` validation run is expected to finish without unresolved warnings in `typecheck`, `lint`,
  `stylelint`, `test:run`, or `build`; build-time Vite/Rollup and dependency warnings remain in scope until cleared or
  explicitly accepted through the controlled exception path
* do not claim a `web` task failed repository completion only because of IDE-only inspections, TS suggestion-level
  diagnostics, or local spell-check findings that are not mirrored by the repository CLI chain
* prefer type checking plus production build when both are available
* at minimum, use the smallest validation that proves changed routes, modules, pages, and TypeScript contracts compile

### 12.3 Cross-Boundary Validation

If a task changes contracts shared across `server` and `web`, or changes menu/permission/route semantics that affect
both sides:

* validate both `server` and `web`
* keep the corresponding contract governance docs aligned in the same change
* if typed enforcement, drift detection, or compatibility checks are expected by the active contract lifecycle but no
  repository automation entrypoint exists yet, report that gap explicitly instead of claiming the contract slice is fully
  validated

### 12.4 Validation Reporting

If validation cannot be run:

* state exactly which command was expected
* state why it could not be run
* do not claim the task is fully complete without that caveat
* distinguish full repository entrypoints from focused direct checks and from execution-stage slices such as
  `graft validate backend --stage ...`

Warnings or failures in directly affected modules are part of the task scope. Do not ignore them unless the user
explicitly narrows the task.
When a frontend warning or backend lint issue is retained as a controlled exception, the corresponding tracking
document must record its source, impact, temporary retention reason, and next cleanup action instead of calling it
non-blocking by default.
README, skills, tracking docs, and CI workflows may point to repository entrypoints or narrower execution slices, but
they must not redefine validation order, acceptance criteria, or local-vs-CI environment rules into a second source of
truth. When wording diverges, root `AGENTS.md` plus the repository entrypoints it names win.

## 13. Git Workflow Rules

For repository work:

* default to a dedicated branch and PR for repository work
* direct development on `main` is allowed only for emergency fixes or when the user explicitly authorizes it
* use branch names in the form `<type>/<topic-or-scope>`
* decide change ownership before staging or committing; a validated change is auto-committable only when its ownership
  is reliably known
* when one logical feature slice reaches a directly validated milestone, commit it before starting the next unrelated
  slice unless the user explicitly asks to batch them
* if the working tree already mixes multiple feature points, split them back to feature-granularity commits before
  considering the task complete; do not leave validated slices piled up as uncommitted changes
* default to one logical closure per commit; for larger tasks, split commits into readable stages such as
  schema/migration, runtime implementation, tests, docs, or cleanup/refactor
* each commit should remain as buildable or testable as the current slice reasonably allows; do not rely on hidden
  local context to make an intermediate commit understandable

Automatic commits are allowed only after ownership is classified:

* scenario 1: if the working tree was clean before the task and the validated change was produced entirely by the
  agent, the agent may create the commit unless the user explicitly says not to commit
* scenario 2: if the working tree was already dirty, but the agent can reliably distinguish the task's owned files or
  hunks through `git status` and `git diff`, the agent may commit only the owned scope it can prove
* scenario 3: if user edits, unknown edits, or unrelated topic edits are mixed together and ownership cannot be
  reliably separated, the agent must not auto-commit; explain the mixed state to the user and limit the next step to
  one of these paths: commit only the confirmable scope, let the user specify the commit scope, or leave the changes
  uncommitted

Explicit commit trigger:

* when the user explicitly invokes a repository commit trigger such as `$graft-commit`, treat it as permission to
  create one scoped commit for the current validated task slice, but still apply the ownership and mixed-change rules
  above before staging anything
* the trigger grants permission to commit the confirmed owned scope only; it does not permit bundling unrelated files,
  unknown changes, or all current working tree changes by default
* if the current slice is not yet validated to the level required by its task class, finish the required validation
  before committing or explain why that validation cannot be completed yet
* if the working tree is mixed and the owned scope cannot be separated confidently, the trigger does not override the
  fail-closed rule; stop and report the ambiguity instead of forcing a commit

For staging and mixed-ownership files:

* never stage or commit existing user changes, unknown-origin changes, unrelated files, or cross-topic files together
  with the current task just because they are present in the working tree
* default to staging only files or hunks whose ownership is confirmed
* do not use `git add .`, `git add -A`, or `git commit -am` unless the user explicitly requests committing all
  current changes
* if one file contains both user-owned and agent-owned edits, commit only the owned hunks when they can be reliably
  separated
* if mixed ownership inside one file cannot be reliably separated at hunk level, the agent must not auto-commit that
  file
* a file being relevant to the current task is not enough to justify committing the whole file when ownership is mixed

For commit hygiene:

* do not create noise commits such as `wip`, `update`, `fix typo`, temporary debug snapshots, or commits that mix
  unrelated formatting with behavior changes
* do not run repository-wide formatting unless the user explicitly asks for it
* do not let IDE actions, formatter passes, organize-imports actions, or `--fix` flows introduce broad unrelated diffs
* treat formatting drift outside the current task scope as a high-risk change by default
* formatting changes may be committed only within the files or hunks that belong to the current task; if the drift
  cannot be contained to that scope, it must not be auto-committed

Commit messages must use Conventional Commits:

* format: `<type>(<scope>): <summary>`
* the title must default to English
* `scope` is required and must be explicit
* keep the title focused on what changed, not on AI behavior or the implementation process
* do not place literal escaped control text such as `\n`, `\t`, or `\r` inside the commit title or body

Commit type rules:

* use `feat` for user-facing or plugin/platform capability additions
* use `fix` for behavior corrections
* use `refactor` for non-feature restructuring
* use `perf` for observable performance improvements
* use `docs`, `test`, `build`, `ci`, `chore`, or `style` for their literal categories

Do not use `feat` for documentation-only changes.

When a commit needs a body:

* use unordered bullet items
* start each bullet with a verb such as `新增`、`修复`、`优化`、`更新`、`补充`、`重构`
* make each bullet describe one independent change point
* write the title and body as real multi-line text
* if a commit message is generated by automation, expand escaped text into actual line breaks and indentation before
  invoking `git commit`

## 14. Automation and CI/CD Rules

Repository automation should follow the same boundary rules as local development.

### 14.1 Pull Request Validation

When the repository adds CI workflows:

* keep pull request validation and release automation in separate workflows
* validate `server` and `web` as separate jobs when both sides exist
* when backend lint governance is active, keep `server` lint and `server` build/test as separate jobs instead of one
  opaque backend script step
* when CI keeps split jobs or stage flags, document them as execution-layer decomposition of the same repository
  validation truth, not as independent acceptance contracts
* prefer a fast quality or security track plus a build or test track instead of one opaque monolithic job
* cache dependencies by ecosystem, such as Go modules and frontend package manager caches
* upload useful failure artifacts or summaries when they materially improve debugging
* keep current-stage workflows honest about repository maturity; prefer smoke validation over fake full builds when the
  actual toolchain or artifacts are not stable yet
* backend CI must reuse the same `graft validate backend` entrypoint and pinned `golangci-lint` version as local
  development instead of rebuilding a second lint parameter set inside workflow YAML
* when local `web` development in WSL requires host Windows Bun, keep that rule explicit in repository docs; a Linux CI
  runner reusing the same `bun run check` entrypoint is an execution environment difference, not permission to relax
  the local WSL rule

### 14.2 Release Automation

When the repository later adds release workflows:

* build artifacts once and reuse them across publish steps
* keep release gating stricter than pull request validation
* use explicit concurrency control for release or docs publish workflows
* do not introduce package publishing complexity that the repository does not actually need yet

### 14.3 Security and Maintenance Automation

When adding repository maintenance workflows:

* prefer CodeQL or equivalent scanning for the actual languages in this repository
* prefer secret scanning on pull requests
* prefer Dependabot or equivalent automation for Go modules, frontend dependencies, and GitHub Actions
* keep optional workflows such as docs publish or benchmarks separate from the main CI path

## 15. License Governance

This repository is licensed under Apache License 2.0.

Contributors must preserve that licensing posture when changing code, docs, automation, or dependencies.

### 15.1 Repository License Files

* do not remove or weaken the top-level `LICENSE` file
* if the repository later requires a `NOTICE` file or third-party license inventory, keep those files aligned with the
  actual distributed contents
* do not add repository rules that conflict with Apache-2.0 distribution terms

### 15.2 Source File Headers

The repository does not currently enforce a header script or SPDX baseline, so contributors must not invent a fake
mandatory workflow.

If the project later adopts source header enforcement:

* prefer SPDX-style Apache-2.0 identifiers that are easy to validate automatically
* apply the policy consistently across supported source and configuration file types
* document exclusions for generated files, third-party code, lockfiles, and build output

### 15.3 Dependency and Distribution Compliance

When introducing a new dependency, package, or distributable artifact:

* check whether its license is compatible with Apache-2.0 distribution
* record any required attribution or notice obligations when they apply
* avoid adding copyleft or distribution-restrictive dependencies without an explicit repository decision
* keep future CI license checks lightweight until the repository has a real release pipeline and artifact inventory

## 16. Subagent Usage Rules

Use subagents only when the task is complex, the context is likely to grow too large, or the work can be split into
independent parallel subtasks.

The main agent must identify the critical path first. Do not delegate the immediate blocking task if the next local
step depends on that result.

Use subagents this way:

* use `explorer` subagents for read-only discovery, comparison, tracing, and narrow codebase questions
* use `worker` subagents only for bounded implementation tasks with an explicit file or subsystem ownership boundary

Every delegation must specify:

* the inherited startup context required by `4.1 Startup Governance`
* the concrete objective
* the expected output format
* the files or subsystem the subagent owns
* any constraints about tests, diagnostics, or compatibility

Subagents are not allowed to revert or overwrite unrelated user changes or parallel agent changes. They must adapt to
concurrent work instead of assuming exclusive ownership of the repository.

The main agent remains responsible for:

* critical-path selection
* validation planning
* review and acceptance of every subagent result
* final integration
* final completion judgment

Repository subagent usage is allowed in this project when it follows these rules.

## 17. Complex Task Tracking

For complex, multi-step, or multi-agent work:

* keep an explicit execution record if the task would be hard to resume safely from chat history alone
* use the repository-local `ai-plan/` workflow instead of inventing ad-hoc tracking files
* after completing the startup preflight in `4.1 Startup Governance`, read `ai-plan/public/README.md` before scanning
  active topics when resuming or booting into complex work
* keep repository-wide design truth in `ai-plan/design/` and `ai-plan/roadmap/`
* keep active topic recovery state under `ai-plan/public/<topic>/`

`ai-plan/` uses these directory semantics:

* `ai-plan/design/`
  * repository-wide architecture and design truth
* `ai-plan/roadmap/`
  * repository-wide implementation plans and staged delivery documents
* `ai-plan/public/README.md`
  * shared recovery index that maps branches or worktrees to active topics after startup preflight
* `ai-plan/public/<topic>/todos/`
  * recovery-safe tracking documents for one active topic
* `ai-plan/public/<topic>/traces/`
  * execution traces for one active topic
* `ai-plan/public/<topic>/subtopics/<name>/todos/`
  * recovery-safe tracking documents for one bounded subtopic inside an active topic
* `ai-plan/public/<topic>/subtopics/<name>/traces/`
  * execution traces for one bounded subtopic inside an active topic
* `ai-plan/public/<topic>/design/`
  * topic-specific design documents that do not belong in repository-wide design truth
* `ai-plan/public/<topic>/roadmap/`
  * topic-specific implementation plans that do not belong in repository-wide roadmap truth
* `ai-plan/public/<topic>/archive/`
  * archived stage-level artifacts for an active topic
* `ai-plan/public/archive/<topic>/`
  * completed-topic archive that should not be treated as default boot context

Use these workflow rules:

* `ai-plan/public/README.md` must list only active topics
* when a branch or worktree has an active-topic mapping, read its tracking and trace files after startup preflight and
  before substantive recovery work
* when an active topic defines subtopics, read the parent topic first and then continue into the relevant subtopic based
  on the current `server`, `web`, or cross-boundary task boundary
* when working from a tracked topic, update the corresponding tracking document in the same change
* when work is clearly scoped to one subtopic, update that subtopic tracking document in the same change and keep the
  parent topic focused on cross-boundary milestones, shared risks, and shared next steps
* for complex work, maintain a matching trace that records the current date, key decisions, validation milestones, and
  the immediate next step
* keep active tracking and trace files concise enough to serve as recovery entrypoints
* when a stage inside an active topic is complete, move detailed history into that topic's `archive/` and keep only the
  active recovery point in the default recovery path
* when a topic is fully complete, move the entire topic directory under `ai-plan/public/archive/<topic>/` and remove it
  from `ai-plan/public/README.md` in the same change
* never record absolute file-system paths in `ai-plan/**`; use repository-relative paths, branch names, commit ids, PR
  numbers, and validation commands instead

## 18. Commenting and Documentation Rules

All generated or modified code must include clear and meaningful comments where required by the rules below.

### 18.1 Server Documentation

For Go code:

* all hand-written exported packages, types, interfaces, functions, methods, and constants must have Go-style doc
  comments
* all hand-written Go comments must use Chinese, while preserving stable technical terms in English when needed
* comments must explain intent, contract, usage constraints, or design reasons instead of restating syntax
* for functions and methods, prefer a two-layer style: explain responsibility first, then add boundary, ordering, or
  failure semantics; only add `参数：` / `返回值：` sections when the function's inputs, outputs, lifecycle ordering,
  or side effects are not obvious from the signature
* use `server/internal/cli/dev.go` as a function-comment example for complex orchestration entrypoints, but do not
  mechanically force every function into the same parameter-list template
* `revive` and `stylecheck` should be treated as executable enforcement of exported-comment baselines and common Go
  style, but passing lint does not replace the deeper comment-quality requirements in this document
* plugin lifecycle types and methods must document registration order, boot semantics, shutdown expectations, and
  failure behavior when relevant
* cross-plugin interfaces must document stability expectations and what callers may depend on
* package comments should live in `doc.go` when practical and explain responsibility plus boundary intent
* do not generate mechanical comments such as `Name 是名称` or `ID 是 ID`
* when code and old comments conflict, verify the implementation context first and then update the comment in the same
  change
* generated code, third-party code, migration artifacts, and build outputs are exempt, but their hand-written wrapper
  layers still follow the documentation rules
* field comments are required only for key fields such as lifecycle-sensitive, shared, nullable, or constraint-heavy
  fields; do not mechanically document every field
* top-level test functions should state the scenario or contract they lock down; helper functions only need comments
  when their intent is not obvious from the test shape
* context propagation boundaries, transaction ownership, runtime wiring, goroutine exit conditions, resource cleanup
  ownership, API/DTO boundaries, and sensitive logging constraints should be documented according to
  `9. Go 代码组织与命名规范` when they are not obvious from the code shape

### 18.2 Web Documentation

For TypeScript and Vue code:

* comments must use Chinese when they are needed
* add comments for non-trivial routing assembly, permission gating, dynamic menu composition, and complex page-state
  synchronization
* document why a store exists when the same state could have been page-local
* document backend contract assumptions when the UI depends on menu, permission, or plugin metadata semantics

### 18.3 Inline Comment Rules

Add inline comments for:

* non-trivial logic
* concurrency behavior
* lifecycle sequencing
* business rules that are not obvious from the code shape
* compatibility constraints
* registration order assumptions
* workarounds and edge cases

Prefer standalone line comments ahead of the logic block for complex behavior instead of trailing end-of-line comments.

Do not add trivial or mechanical comments that only restate the code.

### 18.4 Architecture-Level Documentation

Core framework components and plugin-extension primitives must explain:

* responsibilities
* lifecycle
* interaction with other components
* why the abstraction exists
* when to use it instead of simpler alternatives

### 18.5 Module README Rules

Module-level `README.md` files are navigation documents, not detailed design documents.

Rules:

* add `README.md` to module-level directories with independent responsibilities such as `server/internal/<module>`,
  `server/plugins/<name>`, and `web/src/modules/<name>`
* use `README.md` consistently; do not mix with `ReadMe.md`
* explain module purpose, boundary, main entrypoints, upstream/downstream relationships, and extension guidance
* keep detailed architecture decisions in `ai-plan/design/` instead of duplicating them inside module READMEs

### 18.6 Comment Priority

When time or scope is limited, prioritize comments in this order:

* public API comments
* architecture-boundary comments
* concurrency and lifecycle comments
* business-rule comments
* ordinary implementation comments

Missing required documentation is a standards violation. Code that does not meet these documentation rules is
incomplete.

## 19. Change Management

When making substantial changes:

* explain which `ai-plan/design/` or `ai-plan/roadmap/` section the change follows
* keep architecture changes aligned with `ai-plan/`
* avoid silent changes to core conventions

If a task reveals that the current docs are wrong:

* update the relevant doc
* state the new rule clearly
* then implement against the updated rule

## 20. Code Review Expectations

Review for:

* boundary violations between core and plugins
* hidden coupling between plugins
* unnecessary framework complexity
* divergence from Go + Gin + Ent + Casbin server rules
* divergence from Vue 3 + TDesign web rules
* duplicate canonical contract definitions, undocumented contract ownership, or missing lifecycle / compatibility notes
  for high-risk contract changes
* missing tests around plugin lifecycle, dependency ordering, authorization, and dynamic menu/route behavior
* undocumented public interfaces or lifecycle-sensitive code
* divergence between `ai-plan/design/`, `ai-plan/roadmap/`, and active topic recovery documents

A change is not acceptable if it makes adding the next plugin or frontend module harder.

## 21. Definition of Done

A task is done only when all relevant items below are satisfied:

* the change follows the current `ai-plan/` documents, or the docs were updated first
* `server` and `web` boundaries are still clear
* new module work keeps the `menu + route + page + api + permission` path explicit
* `server` code in scope complies with `9. Go 代码组织与命名规范`, including context propagation, transaction
  boundaries, resource cleanup, API/DTO boundaries, config loading, runtime wiring, auth handling, logging, and
  AI-code governance constraints
* any new or changed high-risk contract follows the canonical ownership, lifecycle, and compatibility rules in
  `ai-plan/design/契约治理与魔法值治理规范.md`
* affected code has the required comments and documentation
* the changed area passed direct validation, or the exact validation gap was reported
* `server` work reached its completion state only after `graft validate backend` passed with the full backend quality
  chain and no unresolved backend lint issue remained outside an explicitly documented controlled exception
* `web` work reached its completion state only after the host Windows Bun full quality chain passed and no unresolved
  frontend warning remained outside an explicitly documented controlled exception
* the final summary states the important behavior change, validation result, and any remaining blockers

If any of these are missing, the task is incomplete even if the code compiles.
