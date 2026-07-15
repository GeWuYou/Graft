# Backend Lessons

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

## LESSON-BACKEND-MIGRATION-VERSION-001：已执行 Atlas migration 版本不能追加新 DDL

- Status: active
- Level: L2
- Applies to:
  - `server/modules/*/migrations/**`
  - `server/internal/*/migrations/**`
  - 任何已经被本地、CI 或协作者数据库执行过的 Atlas versioned migration
- Source:
  - 2026-06-05 scheduler 启动缺少 `scheduled_tasks` 表的修复
  - 2026-06-11 用户指出不应修改已执行的 `202606050002_scheduler_scheduled_tasks.sql`，应通过新 migration 修复
  - 2026-07-15 access-log WebSocket 回填修复：撤回对已提交 `202607150001` 的改写，改为追加 `202607150002`
- Problem:
  `202606050001_scheduler_task_runs.sql` 先被执行并记录到 Atlas revision，后来同一个 version 文件又追加了 `scheduled_tasks` 表 DDL。数据库 revision 已经推进到该 version，Atlas 显示无 pending migration，但实际 schema 没有新追加的表，导致 scheduler Boot seed 内置任务时报 `relation "scheduled_tasks" does not exist`。
- Correct pattern:
  已提交、共享或进入 CI 的 versioned migration 默认视为可能已执行，后续 DDL 或数据修复必须新增更高 version 的补丁 migration；补丁 migration 可使用 `IF NOT EXISTS` 修复当前缺口，但不得依赖 Atlas 重放旧 version。仅在先通过配置的本地数据库核验 Atlas revision、并能证明本轮所有相关环境均未执行该 version 时，才允许改写该未执行文件。
- Anti-pattern:
  在已经执行过的 migration version 文件里追加表、列、索引或注释，然后只更新 `atlas.sum`，期待已有数据库自动补齐新增 DDL。
- Enforcement:
  修改历史 migration 前先使用配置的本地数据库执行 `atlas migrate status`，并检查 revision 与实际 schema；本地未执行不能单独推翻共享 migration 可能已执行的默认假设。若不能证明所有相关环境均未执行，必须新增后续 migration。提交前必须检查 `git diff -- server/**/migrations/*.sql`，确认没有修改已可能执行的历史 migration 文件。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `server/modules/scheduler/migrations/202606050002_scheduler_scheduled_tasks.sql`
  - `server/modules/scheduler/migrations/atlas.sum`
- Updated at:
  2026-07-15

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
