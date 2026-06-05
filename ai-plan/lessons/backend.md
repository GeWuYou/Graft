# Backend Lessons

## LESSON-BACKEND-MIGRATION-VERSION-001：已执行 Atlas migration 版本不能追加新 DDL

- Status: active
- Level: L1
- Applies to:
  - `server/modules/*/migrations/**`
  - `server/internal/*/migrations/**`
  - 任何已经被本地、CI 或协作者数据库执行过的 Atlas versioned migration
- Source:
  - 2026-06-05 scheduler 启动缺少 `scheduled_tasks` 表的修复
- Problem:
  `202606050001_scheduler_task_runs.sql` 先被执行并记录到 Atlas revision，后来同一个 version 文件又追加了 `scheduled_tasks` 表 DDL。数据库 revision 已经推进到该 version，Atlas 显示无 pending migration，但实际 schema 没有新追加的表，导致 scheduler Boot seed 内置任务时报 `relation "scheduled_tasks" does not exist`。
- Correct pattern:
  一旦某个 Atlas migration version 可能已经执行，后续 schema 增量必须新增更高 version 的补丁 migration；补丁 migration 可使用 `IF NOT EXISTS` 修复当前缺口，但不得依赖 Atlas 重放旧 version。
- Anti-pattern:
  在已经执行过的 migration version 文件里追加表、列、索引或注释，然后只更新 `atlas.sum`，期待已有数据库自动补齐新增 DDL。
- Enforcement:
  修复缺失 schema 时先用 `atlas migrate status` 和 `atlas schema inspect` 区分 revision 状态与实际结构；若 revision 已到目标 version 但结构缺失，必须新增后续 migration，并在验证中应用迁移、检查结构、启动对应模块。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `server/modules/scheduler/migrations/202606050002_scheduler_scheduled_tasks.sql`
  - `server/modules/scheduler/migrations/atlas.sum`
- Updated at:
  2026-06-05

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
