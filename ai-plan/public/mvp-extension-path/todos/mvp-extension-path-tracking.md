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
- 当前仓库级 startup governance 最小迁移切片已落地到文档真值：根 `AGENTS.md` 现在独占 startup preflight、
  最小 receipt、resume/restart 重验与 subagent 最小继承包定义；`graft-boot`、`graft-multi-agent-batch` 与
  `ai-plan` 文档不再各自维护第二套 boot 链。
- `server` 当前已形成最小 runtime 闭环，并新增受保护的 `/api/auth/bootstrap` 真实契约，统一返回当前用户、权限码、按权限过滤的菜单和 locale 快照。
- `web` 当前已把登录主路径收敛到真实 `login/refresh/bootstrap` 契约，并开始用 bootstrap 菜单快照驱动最小动态路由，而不是继续依赖 starter mock 登录与静态 demo 菜单。
- 当前 cross-boundary 切片已把 `/users` 从“真实菜单路径 + demo 页面内容”的假闭环，收敛为真实 `GET /api/users` 契约加最小只读列表页；`/user/index` 静态个人中心残留入口已移除。
- 当前 auth / RBAC 最小响应收敛切片也已完成第一轮跨边界对齐：`server` 已冻结 `AUTH_*` 失败 code、统一 envelope 与 request-id 透传，`web` 已冻结“仅 `AUTH_TOKEN_EXPIRED` refresh，`INVALID/MISSING` 单一退出”的请求层行为，并通过显式 session bridge 消除了构建 warning。
- 当前默认管理员与首次登录强制改密切片的跨边界口径已冻结：`graft-admin` 只允许作为默认管理员初始化例外密码写入；首次改密状态必须由后端持久化并通过 `login/bootstrap` 返回；当前 MVP 不为全部业务接口增加全局后端拦截，而由 `web` 登录后受限态负责阻断，后续如需更强安全再补服务端全局中间件。
- 当前仓库真值还新增冻结了一条 shared backend governance baseline：`server` 完成态必须收敛到 `graft validate backend`，
  固定使用 `golangci-lint v2.12.2`，并把 backend lint issue 默认视为阻断项；如需暂留，只能登记到 active
  tracking 文档中的 controlled exception。
- 当前 `server-lint` CI 路径已修正为“只安装 pinned `golangci-lint`，再由 `graft validate backend --stage lint` 执行统一入口”；生产代码 lint backlog 已清零，但测试代码 lint backlog 仍阻断后端完成态。
- 较早的拆分前历史保留在 `archive/`，具体实现轨迹保留在各自 `trace` 文件。

## Shared Milestones

- `mvp-extension-path` 已稳定为父主题 + `server` / `web` 子主题的恢复结构。
- 后端已具备最小可运行的 plugin-oriented runtime、迁移链路、认证与 RBAC 基线。
- 前端已具备可继续收敛的 starter 壳层，并已重新挂回真实认证与最小动态菜单入口，host Windows Bun `bun run check` 当前再次通过零 warning 完成门槛。
- 当前主题阶段已从“后端先闭环”推进到“后端稳定 bootstrap 契约，前端按该契约同步收敛”。
- 当前主题阶段还额外完成了 auth 响应面与 refresh 单出口的第一轮跨边界收口，后续可以把主线重心放回真实页面接线而不是继续补请求层控制流分支。
- 后端 lint 治理的仓库真值已经先行冻结：后续 `server` 与 cross-boundary 收尾都必须以同一个 backend quality entrypoint 和同一套 lint 口径完成，而不是继续依赖散落的临时命令。

## Shared Risks

- 如果 `web` 回到页面扩张或长期保留 mock/demo 依赖，前后端契约会再次漂移。
- `auth`、`menu`、`permission`、`locale` 等共享契约若在后端收敛期内继续无边界扩张，会放大 `web` 接线返工成本。
- 如果后续 skill、README、或 topic 恢复文档再次复制完整 boot 规则，当前 startup governance 收口会重新退化成
  双轨并存状态，`resume` 与 subagent 入口会最先失真。
- 如果默认管理员、首次改密真值来源、或登录后阻断责任在 `server` 与 `web` 之间重新漂移，后续实现会重新引入猜测式前端逻辑或半生效的安全边界。
- disposable PostgreSQL / Redis 校验仍依赖手工准备环境，后续恢复时必须显式说明当前可用的校验入口。
- 如果本地、agent 和 CI 在 `server` 完成态上继续各自维护不同的 lint 命令或参数，新的 backend 治理基线会很快再次失真，无法稳定阻断可维护性回退。

## Shared Validation Summary

- 本次跨边界直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run check`
- 本次 `/users` 真实列表切片直接校验：
  - `cd server && go test ./plugins/user ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- 本次 topic 级同步通过 `sed`、`rg`、`git diff -- ai-plan/public/mvp-extension-path` 与对应直接校验结果完成一致性检查。
- 本次 startup governance 最小迁移切片一致性检查：
  - `rg -n "Startup Governance|startup preflight|startup receipt|recovery index|subagent" AGENTS.md .agents/skills/graft-boot/SKILL.md .agents/skills/graft-multi-agent-batch/SKILL.md ai-plan/README.md ai-plan/public/README.md ai-plan/design/AI任务追踪与恢复设计.md ai-plan/public/mvp-extension-path`
  - `git diff -- AGENTS.md .agents/skills/graft-boot/SKILL.md .agents/skills/graft-multi-agent-batch/SKILL.md ai-plan/README.md ai-plan/public/README.md ai-plan/design/AI任务追踪与恢复设计.md ai-plan/public/mvp-extension-path`
- 本次 auth / RBAC 响应收敛切片直接校验：
  - `cd server && go test ./internal/httpx ./plugins/user`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run test:run -- src/utils/request.test.ts src/store/modules/user.test.ts src/utils/route/bootstrap.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- 本次默认管理员/首次改密文档与跟踪同步一致性检查：
  - `rg -n "graft-admin|must_change_password|change-password|bootstrap|受限态" ai-plan/design/项目设计.md server/plugins/user/README.md ai-plan/public/mvp-extension-path`
  - `git diff -- ai-plan/design/项目设计.md server/plugins/user/README.md ai-plan/public/mvp-extension-path`
- 本次 backend lint 治理文档切片一致性检查：
  - `rg -n "golangci-lint|graft validate backend|controlled exception|revive|stylecheck" AGENTS.md README.md ai-plan/design/项目设计.md ai-plan/design/代码注释与模块文档规范.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
  - `git diff -- AGENTS.md README.md ai-plan/design/项目设计.md ai-plan/design/代码注释与模块文档规范.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
- 本次 workflow/lint 直接校验：
  - `cd server && go test ./internal/httpx ./internal/store/entstore ./plugins/user`
  - `cd server && golangci-lint run --config .golangci.yml ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./internal/app ./internal/cli ./internal/config ./internal/database ./internal/httpx ./internal/i18n ./internal/plugin ./internal/redisx ./internal/store/entstore ./plugins/audit ./plugins/rbac ./plugins/scheduler ./plugins/user`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - 结果：生产配置通过；测试配置仍被既有 test-lint backlog 阻断。

## Immediate Next Step

- 在当前 backend governance 真值已经落到 CI 安装/执行分离后，下一步单独治理 `server/.golangci.test.yml` 暴露的历史测试 lint backlog；不要让生产代码已清零但测试 lint 长期阻断完成态。
- 保持当前 `/api/auth/bootstrap` 契约稳定，只在必要范围内收敛 DTO 和菜单/权限语义。
- 在该 bootstrap 与 auth 响应契约上继续实施默认管理员与首次登录强制改密闭环：先由 `server` 增加持久化状态、初始化例外密码和最小管理员绑定，再由 `web` 落登录后受限态与强制改密弹窗，而不是扩大安全范围或回到猜测式控制流。
