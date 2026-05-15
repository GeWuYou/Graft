# 后端主导的 MVP 闭环收敛计划跟踪

## Topic

- Topic: `mvp-extension-path`
- Branch: `feat/mvp-extension-path`
- Scope: 以 `server` 主导完成 MVP 闭环，并推动 `web` 收敛到真实后端契约

## Goal

- 保持 `mvp-extension-path` 作为默认恢复入口，把当前阶段聚焦到后端闭环收敛和前端真实契约对接，而不是继续扩大会话治理或页面广度。

## Repository Truth

- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/代码注释与模块文档规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Topic Roadmap

- `ai-plan/public/mvp-extension-path/roadmap/backend-mvp-closure-plan.md`

## Subtopics

- `server`
  - Tracking: `ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/subtopics/server/traces/server-trace.md`
  - Use for: backend runtime、plugin lifecycle、registries、Ent/Atlas、event bus、audit、scheduler、auth/RBAC 与跨插件契约稳定化。
- `web`
  - Tracking: `ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
  - Trace: `ai-plan/public/mvp-extension-path/subtopics/web/traces/web-trace.md`
  - Use for: starter 壳层收敛、真实后端 `auth + menu + permission + locale` 契约挂接，以及 mock/demo 清理。

## Parent-Scope Rules

- 父主题只保留跨边界方向、共享风险、共享验证摘要和子主题入口指引。
- 纯 `server` 推进写入 `server` 子主题；纯 `web` 推进写入 `web` 子主题。
- 任何会改变共享契约、校验口径或阶段主线的变更，都要同时更新父主题与相关子主题。

## Current Recovery Point

- 当前阶段正式收敛到“后端主导的 MVP 闭环收敛计划”：先补齐 `server` 的 event bus、audit、scheduler 和稳定插件契约，再让 `web` 挂接这些真实契约。
- `server` 当前已形成最小 runtime 闭环，并新增受保护的 `/api/auth/bootstrap` 真实契约，统一返回当前用户、权限码、按权限过滤的菜单和 locale 快照。
- `web` 当前已把登录主路径收敛到真实 `login/refresh/bootstrap` 契约，并开始用 bootstrap 菜单快照驱动最小动态路由，而不是继续依赖 starter mock 登录与静态 demo 菜单。
- 较早的拆分前历史保留在 `archive/`，具体实现轨迹保留在各自 `trace` 文件。

## Shared Milestones

- `mvp-extension-path` 已稳定为父主题 + `server` / `web` 子主题的恢复结构。
- 后端已具备最小可运行的 plugin-oriented runtime、迁移链路、认证与 RBAC 基线。
- 前端已具备可继续收敛的 starter 壳层，并已重新挂回真实认证与最小动态菜单入口，host Windows Bun `bun run check` 当前再次通过零 warning 完成门槛。
- 当前主题阶段已从“后端先闭环”推进到“后端稳定 bootstrap 契约，前端按该契约同步收敛”。

## Shared Risks

- 如果 `web` 回到页面扩张或长期保留 mock/demo 依赖，前后端契约会再次漂移。
- `auth`、`menu`、`permission`、`locale` 等共享契约若在后端收敛期内继续无边界扩张，会放大 `web` 接线返工成本。
- disposable PostgreSQL / Redis 校验仍依赖手工准备环境，后续恢复时必须显式说明当前可用的校验入口。

## Shared Validation Summary

- 本次跨边界直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
  - `cd web && "/mnt/c/Users/gewuyou/.bun/bin/bun.exe" run check`
- 本次 topic 级同步通过 `sed`、`rg`、`git diff -- ai-plan/public/mvp-extension-path` 与对应直接校验结果完成一致性检查。

## Immediate Next Step

- 保持当前 `/api/auth/bootstrap` 契约稳定，只在必要范围内收敛 DTO 和菜单/权限语义。
- 在该 bootstrap 契约上继续扩展 `web` 真实页面接线，优先补齐当前用户、用户管理和权限可见性链路，而不是回到 starter demo 页面扩张。
