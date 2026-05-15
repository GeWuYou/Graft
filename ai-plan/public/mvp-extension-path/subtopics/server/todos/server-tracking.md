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
- PR #8 当前一轮 review follow-up 已修正 refresh session 轮换的提交/条件更新语义，并补齐 smoke validate 并发测试与 auth/RBAC 回归覆盖，当前恢复点无需再为已消费 refresh cookie 的重复使用行为继续返工。
- 同一轮 review 补查确认 `httpx.RequirePermission(..., \"\")` 不应隐式依赖 RBAC 插件，且基础 `Login()` 不应签发未绑定服务端 session 的孤儿 access token；这两处已进入当前跟进范围。
- `pluginapi`、registries、store factory 与当前 auth/menu/permission/i18n 返回面，已经成为 `web` 真实契约收敛前必须谨慎冻结的后端边界。
- 详细实现历史保留在 `subtopics/server/traces/server-trace.md`。

## Active Risks

- event bus、audit、scheduler 仍未形成最小闭环，说明“后端主导的 MVP 闭环收敛”还没有真正完成。
- 如果在下一阶段继续无边界扩张 session-governance 细节，会挤占当前最关键的后端闭环资源。
- 若 `pluginapi`、store DTO 或权限/菜单契约在收敛期内继续频繁漂移，`web` 对真实契约的接线成本会快速上升。
- disposable PostgreSQL / Redis 仍需手工准备；恢复执行时必须确认当前可用的 smoke 环境。

## Latest Validation

- 本次 PR #8 review follow-up 直接校验：
  - `cd server && go test ./internal/cli ./internal/store/entstore ./plugins/user ./plugins/rbac`
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
  1. 稳定 event bus 边界与最小事件发布/订阅能力。
  2. 基于事件链路补齐最小 audit plugin 闭环。
  3. 补齐 scheduler plugin、cron 注册与启动/关闭链路。
  4. 冻结当前 `web` 需要消费的 `auth + menu + permission + locale` 契约面，并只在必要范围内收敛 DTO。
