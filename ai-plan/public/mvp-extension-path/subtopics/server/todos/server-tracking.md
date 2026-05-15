# 后端主导的 MVP 闭环收敛计划 Server 跟踪

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `server`
- Scope: `server/core`、plugin lifecycle、registries、Ent/Atlas、event bus、audit、scheduler、auth/RBAC 与 backend contract stabilization

## Goal

- 让 `server` 先完成 MVP 必要闭环，再把当前 `web` 所需的真实契约稳定下来，避免继续把主线投入到会话治理宽度扩张。

## Current Recovery Point

- `server` 已具备最小可运行 runtime、显式 plugin 注册、Ent/Atlas 迁移链路、基础 auth/RBAC、`graft migrate up` / `graft serve` / `graft validate smoke` 校验入口。
- 现有 session/auth 路径已经足够支撑当前 MVP 收敛阶段；下一阶段重点不再是继续增加 revoke/filter/list 变体，而是补齐 event bus、audit、scheduler 和跨插件稳定契约。
- PR #8 新一轮 AI review 跟进已补强 `RequirePermission` fail-closed 回归断言、登录用户名枚举时序收敛、登录失败日志最小化，以及 user plugin 测试仓储 receiver / session seed 稳定性问题。
- PR #8 当前一轮 review follow-up 已修正 refresh session 轮换的提交/条件更新语义，并补齐 smoke validate 并发测试与 auth/RBAC 回归覆盖，当前恢复点无需再为已消费 refresh cookie 的重复使用行为继续返工。
- 同一轮 review 补查确认 `httpx.RequirePermission(..., \"\")` 不应隐式依赖 RBAC 插件，基础 `Login()` 不应签发未绑定服务端 session 的孤儿 access token，以及 `revoke-others` 需要对并发已失效 session 保持幂等；这三处已进入当前跟进范围并已在本地修正。
- `server/internal/eventbus` 最小进程内事件总线切片已落地：`Runtime` 持有并通过 `plugin.Context` 与 `container` 注入统一 `eventbus.Bus`，当前仅暴露 `Subscribe / Publish`、顺序派发、panic recover 与错误日志记录。
- 最小 `audit` 闭环已接到当前 `eventbus.Bus`：`audit` 插件同时挂载请求级自动审计中间件与主动审计事件订阅，Ent/store 边界只新增稳定写入能力，未提前暴露检索 DSL。
- 最小 `scheduler` 闭环已接入运行时：`cron registry` 声明现在通过独立 `scheduler` 封装装配到 `robfig/cron/v3`，启动与停止语义仍收敛在插件生命周期边界内。
- `user` 插件已新增受保护的 `GET /api/auth/bootstrap` 最小契约：当前登录用户、当前权限码列表、按权限过滤后的菜单列表，以及 locale 配置快照现在可以通过一条真实后端接口返回，供 `web` 后续壳层接线直接消费。
- PR #9 当前一轮 AI review 已确认并落地的 `server` 跟进包括：统一审计 `Action` trim 一致性、主动审计事件同时兼容值/指针 payload、bootstrap locale fallback 去重，以及 `pluginapi.AuditEvent`、scheduler 生命周期文档补强。
- PR #9 当前剩余的 greptile `server` 评论已核对到本地 HEAD：`scheduler` 插件尾部未使用的 `logJobFailure` 确认为死代码，`audit` 请求级自动审计已改为把 `ResourceType` 从稳定路由中拆解为资源域，避免继续与 `RequestPath` 重复。
- PR #9 最新 CodeRabbit nitpick 已在本地核对并收敛：`plugin.Context` 现已显式承载 `LifecycleContext`，runtime 会在 `Shutdown` 阶段注入独立有界关闭上下文，`scheduler` 不再绕过宿主生命周期直接使用 `context.Background()`。
- `pluginapi`、registries、store factory 与当前 auth/menu/permission/i18n 返回面，已经成为 `web` 真实契约收敛前必须谨慎冻结的后端边界。
- 详细实现历史保留在 `subtopics/server/traces/server-trace.md`。

## Active Risks

- 当前最大的剩余风险已经从 runtime 闭环转向共享契约漂移；若后端在 `auth + menu + permission + locale` 返回面上继续频繁变动，`web` 接线会反复返工。
- 如果在下一阶段继续无边界扩张 session-governance 细节，会挤占当前最关键的后端闭环资源。
- 若 `pluginapi`、store DTO 或权限/菜单契约在收敛期内继续频繁漂移，`web` 对真实契约的接线成本会快速上升。
- disposable PostgreSQL / Redis 仍需手工准备；恢复执行时必须确认当前可用的 smoke 环境。

## Latest Validation

- 本次 PR #8 AI review 跟进直接校验：
  - `cd server && go test ./internal/httpx ./plugins/user`
  - `cd server && go vet ./plugins/user`
  - `cd server && go build ./cmd/graft`
- 本次 event bus 切片预期直接校验：
  - `cd server && go test ./internal/eventbus ./internal/app`
  - `cd server && go build ./cmd/graft`
- 本次 audit 切片直接校验：
  - `cd server && go test ./internal/app ./internal/audit ./plugins/audit ./internal/store/entstore ./internal/httpx ./plugins/user`
  - `cd server && go build ./cmd/graft`
- 本次 scheduler 切片直接校验：
  - `cd server && go test ./internal/scheduler ./plugins/scheduler ./internal/cli`
  - `cd server && go build ./cmd/graft`
- 本次 PR #8 review follow-up 补丁直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
- 本次 PR #8 review follow-up 直接校验：
  - `cd server && go test ./internal/cli ./internal/store/entstore ./plugins/user ./plugins/rbac`
  - `cd server && go build ./cmd/graft`
- 本次 bootstrap 契约切片直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
- 本次 PR #9 review follow-up 预期直接校验：
  - `cd server && go test ./plugins/audit ./plugins/user ./internal/scheduler`
  - `cd server && go build ./cmd/graft`
- 本次 PR #9 greptile follow-up 预期直接校验：
  - `cd server && go test ./plugins/audit ./plugins/scheduler`
  - `cd server && go build ./cmd/graft`
- 本次 PR #9 scheduler shutdown context follow-up 预期直接校验：
  - `cd server && go test ./internal/app ./plugins/scheduler`
  - `cd server && go build ./cmd/graft`
- 当前后端恢复基线沿用最近一次 focused backend validation：
  - `cd server && go test ./internal/cli ./internal/app ./internal/store ./internal/store/entstore ./plugins/user ./plugins/rbac`
  - `cd server && go build ./cmd/graft`
- 当前 live-validation 基线沿用最近一次 disposable PostgreSQL / Redis 验证：
  - `graft migrate up`
  - `atlas migrate status`
  - `graft serve` + `/healthz`
- `graft validate smoke` 已经作为下一次最小闭环验证入口存在；本次文档同步没有新增运行时校验。

## Immediate Next Step

- 停止继续扩大会话治理宽度，按以下顺序推进 backend MVP closure：
  1. 在不破坏当前 `/api/auth/bootstrap` 返回面的前提下，只做必要 DTO 收敛，避免 `web` 接线面再次漂移。
  2. 与 `web` 同步推进真实登录态、当前用户、动态菜单与权限守卫接线。
