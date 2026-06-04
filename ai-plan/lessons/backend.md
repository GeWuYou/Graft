# Backend Lessons

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
