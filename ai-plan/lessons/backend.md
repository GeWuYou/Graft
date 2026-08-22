# Backend Lessons

## LESSON-BACKEND-DESTRUCTIVE-CONTRACT-001：删除资源与执行销毁命令必须分型治理

- Status: active
- Level: L2
- Applies to:
  - `openapi/**` 中的软删除、关系解除、不可逆销毁和批量消除性操作
  - `server/modules/*` 的删除授权、审计、持久化与 Task 提交边界
  - `web/src/modules/*` 的删除 mutation 与批量结果消费
- Source:
  - 2026-08-19 消除性操作契约收敛设计中，用户指出 tombstone 重删、不可逆硬删除命令和批量 partial/atomic 语义不能继续由各模块自行解释。
- Problem:
  仅凭 HTTP `DELETE` 名称统一接口，会把资源状态变化、关系解除、不可逆命令和外部系统销毁混成一种行为；同时让各模块自定义批量结果与失败提交策略，会令重试、权限、审计、MCP 暴露和前端刷新行为不可预测。
- Correct pattern:
  普通软删除和关系解除使用幂等 `DELETE`，tombstone 重删返回成功而普通查询默认隐藏；不可逆硬删除与外部资源销毁使用显式 `POST .../deletions|remove|destroy|untag` 命令，高风险命令持久化幂等回执，外部副作用经 Task Runtime 返回 `202`。批量操作必须在契约中声明 `partial` 或 `atomic`，并复用统一 summary/results envelope；RBAC 等安全敏感批量操作默认 atomic。
- Anti-pattern:
  认为 `DELETE` 方法本身足以证明业务幂等；同步等待 Docker、Kubernetes 或云资源删除；让每个模块定义自己的批量删除响应；或在未声明 owner check、审计、确认和 MCP 暴露策略时仅靠人工判断危险性。
- Enforcement:
  每个消除性 OpenAPI operation 必须声明 `x-graft-destructive`；OpenAPI 校验器检查方法、状态码、幂等键、Task receipt、统一批量 envelope 和 MCP 暴露一致性。模块回归测试必须覆盖重删、查询隐藏、权限/审计副作用次数以及批量提交模式。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
  - `openapi/components/schemas/graft-destructive-operation-metadata.yaml`
  - `server/internal/cli/validate_openapi_destructive.go`
- Updated at:
  2026-08-19

## LESSON-BACKEND-CAPABILITY-AUTHORITY-001：能力健康必须与平台可达性分层

- Status: active
- Level: L2
- Applies to:
  - `server/internal/moduleapi/capability.go`
  - `server/internal/capability/**`
  - 能力 provider、OpenAPI projection 与 web shell consumer
- Source:
  - 2026-08-07 平台可用性机制补充设计指出，`Degraded` 内的 Database、Redis、Docker、Registry 等能力不能继续只停留在概念层。
- Problem:
  将浏览器控制面不可达与单项能力故障放进同一个状态源，会导致能力降级误触发整页接管；让每个模块自行维护健康状态，又会产生重复 registry、重试和诊断语义。
- Correct pattern:
  `PlatformAvailabilityStore` 只拥有浏览器控制面可达性；`CapabilityCoordinator` 只拥有服务端 capability observation。能力通过静态 compile-time descriptor/provider 注册，经统一 coordinator 归一化状态、TTL 和 impact，再由 OpenAPI 与 Dashboard 派生消费。
- Anti-pattern:
  用 Monitor 页面或前端 Dashboard 推断能力 authority；把 Runtime Target、Resource 或 Operation 状态复制成平台 capability；为插件/MCP 动态发现另造 registry。
- Enforcement:
  新 capability 必须声明稳定 key、category、impact、provider 和状态契约，并通过 coordinator/API 进入 consumer；新增平台级接管逻辑必须只读取 PlatformAvailabilityStore，不能读取 capability 状态直接重定向。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `server/internal/moduleapi/capability.go`
  - `server/internal/capability/coordinator.go`
  - `web/src/store/modules/platform-availability.ts`
  - `openapi/paths/platform-capabilities.yaml`
- Updated at:
  2026-08-07

## LESSON-BACKEND-TASK-OWNER-001：跨模块 Task owner 必须使用资源公开稳定标识

- Status: active
- Level: L2
- Applies to:
  - `server/internal/moduleapi/task.go`
  - `server/modules/*` 中提交或授权 Task 的 consumer
  - 消费 Task 历史的 `web/src/modules/*`
- Source:
  - 2026-07-15 应用管理状态入口以 `application_id` 查询最近 Task，但 Project 仍把内部数值 `id` 写入 `compose_project` owner，导致授权路径返回隐藏式 404。
- Problem:
  业务模块新增公开资源标识后只更新 HTTP 和前端消费面，未同步 Task owner 的创建、授权与历史数据，会让同一资源在跨模块能力中同时使用公开 ID 和私有主键，产生不可访问的历史记录或错误的资源不存在响应。
- Correct pattern:
  `TaskOwner` 保持通用 `type + id` 契约，由每个 consumer 选择其公开、不可变且可授权解析的资源标识。Project 的 `compose_project` owner 使用 `application_id`；consumer 同步迁移提交、owner authorizer、历史 Task 数据和前端查询，Task module 不反向依赖 Project 实现。
- Anti-pattern:
  将模块私有数值主键写入通用 Task owner；为前端临时暴露私有主键；或让 Task module 为某个业务模块增加 owner 类型分支与兼容解析。
- Enforcement:
  新增或迁移 consumer owner 标识时，测试 Task 提交、owner authorization 和前端 owner-scoped 查询使用同一公开 ID。历史数据变更必须由拥有 `tasks` 表的 Task migration 回填，并保持其它 owner type 与无法映射记录不变。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `server/modules/project/service_lifecycle.go`
  - `server/modules/project/task_authorizer.go`
  - `server/modules/task/migrations/202607150003_project_task_owner_application_id.sql`
  - `ai-plan/design/architecture/任务执行运行时设计.md`
- Updated at:
  2026-07-15

## LESSON-BACKEND-SAVED-VIEW-001：分页保存视图必须分离通用存储与消费页面语义

- Status: active
- Level: L2
- Applies to:
  - `server/modules/saved-view/**`
  - `server/internal/moduleapi/**`
  - 任何需要保存快捷筛选、每页大小和可见列的分页列表模块
- Source:
  - 2026-07-12 用户将 project 专用筛选偏好要求纠正为可供访问日志、审计、应用日志等复用的保存视图能力
- Problem:
  把快捷筛选存成某个业务模块的专用偏好表会复制用户隔离、名称唯一性、软删除与列表状态持久化；反过来暴露不带消费模块授权的 generic HTTP API，又会让通用层错误承担领域权限和筛选语义。
- Correct pattern:
  `saved-view` 只提供按 `(owner_user_id, surface_key)` 隔离的存储 capability，持久化 consumer-validated query JSON、`page_size` 和 visible column keys，并用 live-name 部分唯一索引保证名称唯一。消费模块在自己的授权路由上固定并校验 `surface_key`、筛选字段和列键；应用视图时从第一页开始，不保存当前页，也不默认共享。
- Anti-pattern:
  新建 project、audit 或 access-log 专用筛选偏好表；将 query/filter 语义解析进 generic service；或新增没有消费域授权的通用 saved-view HTTP endpoint。
- Enforcement:
  新 consumer 必须通过 `moduleapi.SavedViewService` 使用存储能力，并在其 module-owned route/service 测试中覆盖 owner/surface 隔离、live 名称冲突、非法筛选/列键和当前页不持久化。迁移需包含 `deleted_at = 0` 查询语义、部分唯一索引和用户/页面列表索引。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `server/internal/moduleapi/saved_view.go`
  - `server/modules/saved-view/**`
  - `server/modules/project/saved_views.go`
  - `ai-plan/design/domains/compose/Compose项目管理设计.md`
- Updated at:
  2026-07-12

## LESSON-BACKEND-MODULE-LIFECYCLE-001：Builder 不应解析 Register 才暴露的跨模块服务

- Status: active
- Level: L2
- Applies to:
  - `server/modules/*/descriptor.go`
  - `server/modules/*/module.go`
  - `server/internal/moduleapi/**`
  - 跨模块 capability 的 provider / consumer wiring
- Source:
  - 2026-06-09 notification 启动失败 `resolve rbac access service: service not registered: *moduleapi.RBACAccessService`
- Problem:
  `notification` 在 descriptor builder 阶段解析 `moduleapi.RBACAccessService`，但该 capability 由 `rbac.Register` 注册。runtime 会先构造所有模块实例，再执行各模块 `Register`；即使 `notification` 声明了 `DependsOn: ["rbac"]`，builder 仍看不到 `rbac.Register` 才注册的服务，导致模块构建期失败。
- Correct pattern:
  Builder 只解析 core/runtime 已经预注册的基础设施服务，或构造模块自有 repository/service。消费其它模块在 `Register` 阶段暴露的 capability 时，模块必须声明对应 `Dependencies`，并在自身 `Register` 或 `Boot` 的窄 wiring 边界解析同一个 `moduleapi` key 后注入本模块对象。
- Anti-pattern:
  认为 `ModuleSpec.Dependencies` 会让被依赖模块的 `Register` 在当前模块 builder 前执行，进而在 descriptor builder 中硬解析其它模块 `RegisterSingleton` 暴露的服务。
- Enforcement:
  对新增跨模块服务消费增加 descriptor build 测试，至少覆盖“只注册 core 基础设施、不注册被依赖模块 capability 时 builder 仍能成功”；再用 module lifecycle 测试覆盖 `Register` 或 `Boot` 阶段使用完全一致的 `(*moduleapi.Interface)(nil)` key 完成解析与注入。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `server/modules/notification/descriptor.go`
  - `server/modules/notification/module.go`
  - `server/modules/rbac/module_registration.go`
  - `server/internal/moduleregistry/registry.go`
- Updated at:
  2026-06-09

## LESSON-BACKEND-HTTPX-CONTEXT-001：守卫发布安全审计前必须先写回增强后的请求上下文

- Status: active
- Level: L1
- Applies to:
  - `server/internal/httpx/**`
  - 任何会在 HTTP guard / middleware 中发布 audit、security event、app log 或其它 side effect 的路径
- Source:
  - 2026-06-04 access-log closeout / security-event bridge regression tests
- Problem:
  HTTP guard 先构造了包含认证主体的 `context.Context`，但在权限拒绝分支发布 security audit event 前没有把该上下文写回 `gin.Context.Request`。发布器从旧请求上下文读取用户信息，导致 `auth.permission.denied` 安全事件缺少 operator。
- Correct pattern:
  当 guard 或 middleware 生成了更完整的请求上下文，且后续失败分支会发布 side effect 时，必须先执行 `ctx.Request = ctx.Request.WithContext(enrichedCtx)`，再调用发布器、日志器或错误响应分支。
- Anti-pattern:
  只把增强上下文传给授权器或下游 handler，却让同一 guard 内的拒绝/错误分支继续读取旧的 `ctx.Request.Context()`。
- Enforcement:
  为发布 side effect 的拒绝分支增加直接测试，断言 payload 中的 operator、request id、route、method、status 和 metadata 来自增强后的请求上下文。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `server/internal/httpx/authz.go`
  - `server/internal/httpx/authz_test.go`
- Updated at:
  2026-06-04

## LESSON-BACKEND-AGENT-EXECUTION-AUTHORITY-001：特权 Agent 执行必须同时冻结意图、世代能力与瞬时材料边界

- Status: active
- Level: L2
- Applies to:
  - Task-owned external execution lease
  - Runtime Target Agent capability binding 与证书轮换
  - 需要宿主机路径、凭据引用或其它敏感执行材料的独立 Agent
- Source:
  - 2026-08-21 Docker Runtime Agent Batch 4 Application/Container 迁移
- Problem:
  只把业务操作名放进 lease 不能安全完成特权执行：若能力只绑定稳定 Agent 身份，旧证书可能在新世代扩权后继承能力；若把宿主机路径或凭据直接写进 Stage input，Task、日志和数据库又会持久化不该拥有的 provider material。
- Correct pattern:
  领域模块只提交 provider-neutral intent；Task Runtime 冻结 Stage、lease、fence、provider、capability 与版本；Runtime Target 对每次 claim 和 post-claim 请求按活动证书世代重新授权；Agent 在持有有效 fence 后通过瞬时 resolver 取得执行材料，且 material 不进入 Task、日志或 receipt。
- Anti-pattern:
  只在 claim 时检查稳定 identity；让未过期 lease 绕过新世代能力缩减；把 endpoint、凭据、宿主机路径或命令复制进 Task input/result；或保留 server-local fallback 以掩盖协议缺口。
- Enforcement:
  行为测试必须覆盖版本不匹配、能力缺失、旧证书扩权拒绝、重连恢复、post-claim 再授权、material fence/expiry 校验、持久化负面检查和旧执行路径负面扫描；迁移完成后删除对应本地 adapter 与兼容别名。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md`
  - `ai-plan/design/architecture/credential-vault-and-runtime-target-agent-protocol.md`
  - `ai-plan/design/architecture/任务执行运行时设计.md`
  - `server/internal/moduleapi/task.go`
  - `server/modules/runtime-target/**`
- Updated at:
  2026-08-21
