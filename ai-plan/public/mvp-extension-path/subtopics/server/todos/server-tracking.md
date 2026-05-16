# 后端主导的 MVP 闭环收敛计划 Server 跟踪

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `server`
- Scope: `server/core`、plugin lifecycle、registries、Ent/Atlas、event bus、audit、scheduler、auth/RBAC 与 backend contract stabilization

## Goal

- 让 `server` 先完成 MVP 必要闭环，再把当前 `web` 所需的真实契约稳定下来，避免继续把主线投入到会话治理宽度扩张。

## Current Recovery Point

- PR #11 当前一轮 review follow-up 已核对并收敛当前 HEAD 上仍然成立的问题：`user` 插件现在把默认管理员初始化从 `Register` 挪到 `Boot`，`bootstrap` 读模型补齐了 `auth` 仓储空值防御，`EnsureUserCredential` 与 RBAC 幂等写路径补上了唯一约束冲突后的重查/视为已存在语义，避免并发启动或重放时偶发失败。
- 同一轮 follow-up 还补齐了 `must_change_password` 字段注释，并让 `web` 强制改密弹窗不再把 `graft-admin` 常量编进前端 bundle；当前默认管理员密码禁止规则仍保留在 `passwordPolicy` 中，由后端 `AUTH_PASSWORD_REUSE_FORBIDDEN` 作为权威约束。
- `server` 已具备最小可运行 runtime、显式 plugin 注册、Ent/Atlas 迁移链路、基础 auth/RBAC、`graft migrate up` / `graft serve` / `graft validate smoke` 校验入口。
- 现有 session/auth 路径已经足够支撑当前 MVP 收敛阶段；下一阶段重点不再是继续增加 revoke/filter/list 变体，而是补齐 event bus、audit、scheduler 和跨插件稳定契约。
- PR #8 新一轮 AI review 跟进已补强 `RequirePermission` fail-closed 回归断言、登录用户名枚举时序收敛、登录失败日志最小化，以及 user plugin 测试仓储 receiver / session seed 稳定性问题。
- PR #8 当前一轮 review follow-up 已修正 refresh session 轮换的提交/条件更新语义，并补齐 smoke validate 并发测试与 auth/RBAC 回归覆盖，当前恢复点无需再为已消费 refresh cookie 的重复使用行为继续返工。
- 同一轮 review 补查确认 `httpx.RequirePermission(..., \"\")` 不应隐式依赖 RBAC 插件，基础 `Login()` 不应签发未绑定服务端 session 的孤儿 access token，以及 `revoke-others` 需要对并发已失效 session 保持幂等；这三处已进入当前跟进范围并已在本地修正。
- `server/internal/eventbus` 最小进程内事件总线切片已落地：`Runtime` 持有并通过 `plugin.Context` 与 `container` 注入统一 `eventbus.Bus`，当前仅暴露 `Subscribe / Publish`、顺序派发、panic recover 与错误日志记录。
- 最小 `audit` 闭环已接到当前 `eventbus.Bus`：`audit` 插件同时挂载请求级自动审计中间件与主动审计事件订阅，Ent/store 边界只新增稳定写入能力，未提前暴露检索 DSL。
- 最小 `scheduler` 闭环已接入运行时：`cron registry` 声明现在通过独立 `scheduler` 封装装配到 `robfig/cron/v3`，启动与停止语义仍收敛在插件生命周期边界内。
- `user` 插件已新增受保护的 `GET /api/auth/bootstrap` 最小契约：当前登录用户、当前权限码列表、按权限过滤后的菜单列表，以及 locale 配置快照现在可以通过一条真实后端接口返回，供 `web` 后续壳层接线直接消费。
- `user` 插件现已补齐最小 `GET /api/users` 只读列表契约，继续保持在现有 plugin/store 边界内，不提前扩展分页、筛选和写操作，只为 `web` 当前 `/users` 真实接线提供稳定落点。
- 当前 `auth / RBAC` 最小响应收敛切片已经进入实施准备：只允许修改 `server/internal/httpx` 与 `server/plugins/user` 现有链路，目标是稳定 HTTP status 语义、稳定业务 `code`、稳定 auth/bootstrap envelope，并让 `web` 后续只基于 `HTTP status + code` 处理认证分支。
- 当前下一步认证治理切片边界已冻结：默认管理员账号固定为 `graft`；`graft-admin` 是仅允许在初始化路径写入的例外密码；首次改密状态必须由后端持久化并通过 `login/bootstrap` 返回；当前 MVP 不在本切片内给全部业务接口追加全局“已改密”中间件，而是把登录后受限态阻断交给 `web`，后续如需更强安全再评估服务端全局 hardening。
- 默认管理员不能成为只能登录的空账号；若当前不存在可复用的 `admin` 角色/权限 seed，本切片内必须补最小管理员角色/权限绑定。
- 本轮响应收敛明确冻结以下后端契约：成功响应固定保留 `success / code / message / traceId / data`，错误响应固定保留 `success / code / message / traceId`，`messageKey / locale` 仅作为可选扩展口，禁止继续保留 `error + message` 双字段重复设计。
- 本轮认证失败语义明确收口为：`AUTH_INVALID_CREDENTIALS -> 400`、`AUTH_TOKEN_MISSING -> 401`、`AUTH_TOKEN_EXPIRED -> 401`、`AUTH_TOKEN_INVALID -> 401`、`AUTH_FORBIDDEN -> 403`；其中 `CurrentUser` 在 token 可解析但服务端 session 已失效时必须归入 `AUTH_TOKEN_INVALID`，不能再退化成笼统“未登录”。
- 本轮最小链路补充 `request-id` 约束：后端只做最小 UUID request-id，中间件优先读取 `X-Request-Id` 与 `X-Trace-Id`，缺失时生成并写回 `X-Request-Id`，所有 auth/RBAC envelope 的 `traceId` 都必须取该值，不扩展到 OpenTelemetry 或其它 tracing 基础设施。
- 本轮前后端边界约束同步固定 refresh 行为：只有 `AUTH_TOKEN_EXPIRED` 允许触发一次 refresh；`AUTH_TOKEN_INVALID`、`AUTH_TOKEN_MISSING`、`AUTH_FORBIDDEN` 都禁止 refresh；refresh 失败后必须走单一出口，统一清理本地登录态与缓存并跳转登录页，不能继续 retry 或形成递归。
- 当前 auth / RBAC 响应收敛切片已经完成第一轮直接落地：`server/internal/httpx` 现已稳定输出 `AUTH_TOKEN_EXPIRED`、统一 success/error envelope 与最小 request-id 透传，并补齐对应 direct tests；`server/plugins/user` 的关键写接口成功路径也已补上 envelope 回归断言，避免后续回退成裸 `200`。
- PR #9 当前一轮 AI review 已确认并落地的 `server` 跟进包括：统一审计 `Action` trim 一致性、主动审计事件同时兼容值/指针 payload、bootstrap locale fallback 去重，以及 `pluginapi.AuditEvent`、scheduler 生命周期文档补强。
- PR #9 当前剩余的 greptile `server` 评论已核对到本地 HEAD：`scheduler` 插件尾部未使用的 `logJobFailure` 确认为死代码，`audit` 请求级自动审计已改为把 `ResourceType` 从稳定路由中拆解为资源域，避免继续与 `RequestPath` 重复。
- PR #9 最新 CodeRabbit nitpick 已在本地核对并收敛：`plugin.Context` 现已显式承载 `LifecycleContext`，runtime 会在 `Shutdown` 阶段注入独立有界关闭上下文，`scheduler` 不再绕过宿主生命周期直接使用 `context.Background()`。
- `pluginapi`、registries、store factory 与当前 auth/menu/permission/i18n 返回面，已经成为 `web` 真实契约收敛前必须谨慎冻结的后端边界。
- 当前本地启动修复已补齐 `server/.env.example` 的显式 auth 密钥示例、README 最小启动步骤与 GoLand working directory 提示，并用隔离环境测试锁定“缺少 `GRAFT_AUTH_JWT_SECRET` 与 `GRAFT_AUTH_SIGNING_KEY` 时严格失败”的配置行为；未引入任何 dev-only 默认密钥或 auth 语义变更。
- `server` 当前已补充两个独立开发辅助程序：`cmd/graft-jwt-secret` 与 `cmd/graft-signing-key`，用于生成可直接写入 `.env` 的随机 auth 密钥文本；该能力只辅助配置准备，不参与运行时加载或 token 语义。
- `server` 当前已把模块工具链基线提升到 `Go 1.26.x`，并将 `go.uber.org/zap` 收敛到 `v1.28.0`；`go test ./...` 与 `go build ./cmd/...` 在本地 `go1.26.1` 下通过，未触发 Ent/Atlas regeneration。
- 当前 backend governance 文档真值已经冻结：`server` 完成态必须统一走 `graft validate backend`，固定质量顺序为 `golangci-lint run -> go test (smallest direct scope) -> go build ./cmd/graft -> graft validate smoke when needed`，并固定 pin `golangci-lint v2.12.2`。
- 当前 `AGENTS.md` 已新增独立 `Go 代码组织与命名规范` 章节，统一冻结 `server` 手写 Go 代码的文件/包/类型命名、Context 传播、API/DTO、config、runtime wiring、事务、Ent、并发、日志、安全与 AI 生成代码约束；相关完成态与注释/验证章节仅再做引用式收敛，不再重复堆叠第二套真值。
- 当前 CI `server-lint` job 已改为只安装固定版 `golangci-lint` 二进制，再回到 `graft validate backend --stage lint` 统一执行入口；不再让 `golangci-lint-action` 在仓库根目录误跑 `./...`。
- 当前生产代码 lint backlog 已在本地清零，`go run ./cmd/graft validate backend --stage lint` 的生产配置可返回 `0 issues`；完成态仍被 `server/.golangci.test.yml` 暴露的既有测试代码 backlog 阻断，当前量级为 117 条，不属于本次 workflow 与生产代码修复引入的回归。
- backend lint issue 默认按阻断项处理；如果当前切片无法立即清理，只能登记到本文件的 `Backend Lint Controlled Exceptions`，并记录来源、影响、保留原因和下一步清理动作。
- 详细实现历史保留在 `subtopics/server/traces/server-trace.md`。

## Backend Lint Controlled Exceptions

- Source：`cd server && go run ./cmd/graft validate backend --stage lint` 第二阶段 `golangci-lint run --config .golangci.test.yml`，当前集中暴露 `cyclop`、`dupl`、`gosec`、`revive`、`staticcheck`，主要分布在 `internal/*_test.go`、`plugins/*_test.go`。
  Impact：阻断 `server` 完成态统一入口，导致本次修复虽然已清空生产代码 lint 并修正 CI 根目录误跑问题，仍不能宣告 `graft validate backend` 全链路通过。
  Retention Reason：本次切片目标是修复 `golangci-lint-action` 在 monorepo 根目录误跑与当前生产代码 backlog；继续一并治理 117 条测试 lint 会把范围从 workflow/quality-gate 修复扩张为仓库级测试代码清扫，超出当前切片。
  Next Cleanup Action：下一切片按 `server/.golangci.test.yml` 单独治理测试 lint backlog，优先处理 `internal/httpx/*_test.go`、`internal/config/*_test.go`、`plugins/user/*_test.go` 与测试中的 `gosec`/`revive` 高密度文件，然后重跑 `graft validate backend`。
- 后续如需暂留 backend lint issue，只能在本节追加受控例外，并至少记录以下字段：
  - Source：linter 名称，以及对应的 package / file / command。
  - Impact：用户可见影响或工程可维护性影响。
  - Retention Reason：为什么当前切片不能安全清理。
  - Next Cleanup Action：下一步清理动作或计划承接切片。

## Active Risks

- 当前最大的剩余风险已经从 runtime 闭环转向共享契约漂移；若后端在 `auth + menu + permission + locale` 返回面上继续频繁变动，`web` 接线会反复返工。
- 如果 auth/RBAC 收敛期间继续混用 “401 + 各类认证失败” 与非稳定 envelope，`web` 将无法只凭 `HTTP status + code` 做出稳定登录态分支，refresh 死循环风险也会重新出现。
- 如果在下一阶段继续无边界扩张 session-governance 细节，会挤占当前最关键的后端闭环资源。
- 如果把首次改密阻断直接扩展成全局后端接口治理，会偏离当前 MVP 范围并放大本轮 auth/RBAC 收敛面的回归风险。
- 若 `pluginapi`、store DTO 或权限/菜单契约在收敛期内继续频繁漂移，`web` 对真实契约的接线成本会快速上升。
- disposable PostgreSQL / Redis 仍需手工准备；恢复执行时必须确认当前可用的 smoke 环境。
- 如果 `server` 本地完成态、agent 完成态与 CI 阻断继续各自维护不同的 lint 命令或参数，backend quality gate 会在实现落地后迅速重新分叉。

## Latest Validation

- 本次 workflow 根目录误跑修复与生产代码 lint 治理直接校验：
  - `cd server && go test ./internal/httpx ./internal/store/entstore ./plugins/user`
  - `cd server && golangci-lint run --config .golangci.yml ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./internal/app ./internal/cli ./internal/config ./internal/database ./internal/httpx ./internal/i18n ./internal/plugin ./internal/redisx ./internal/store/entstore ./plugins/audit ./plugins/rbac ./plugins/scheduler ./plugins/user`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - 结果：生产配置 `0 issues`；测试配置仍被既有 117 条 test-lint backlog 阻断。

- 本次 PR #11 review follow-up 直接校验：
  - `cd server && go test ./plugins/user ./internal/store/entstore`
  - `cd server && go build ./cmd/graft`
  - `cd server && go run ./cmd/graft validate backend --test-target ./plugins/user --test-target ./internal/store/entstore`
  - `cd web && /mnt/c/Users/gewuyou/.bun/bin/bun.exe run typecheck`
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
- 本次最小用户列表契约切片直接校验：
  - `cd server && go test ./plugins/user ./internal/store/entstore`
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
- 本次 auth / RBAC 最小响应收敛切片预期直接校验：
  - `cd server && go test ./internal/httpx ./plugins/user`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run check`
- 本次 auth / RBAC 最小响应收敛切片实际直接校验：
  - `cd server && go test ./internal/httpx ./plugins/user`
  - `cd server && go build ./cmd/graft`
  - `cd web && bun run test:run -- src/utils/request.test.ts src/store/modules/user.test.ts src/utils/route/bootstrap.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- 本次本地启动 auth 配置引导修复直接校验：
  - `cd server && go test ./...`
  - `cd server && go build ./cmd/graft`
- 本次本地 auth 密钥生成工具切片直接校验：
  - `cd server && go test ./internal/keygen ./...`
  - `cd server && go build ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key`
- 本次 Go 1.26.x / Zap 1.28.0 最小升级切片直接校验：
  - `cd server && go test ./...`
  - `cd server && go build ./cmd/...`
  - `cd server && PATH=/tmp/codex-bin:$PATH go run ./cmd/graft validate backend`
- 本次 PR #11 review follow-up 的 focused `go test`、`go build ./cmd/graft` 与 `web` `typecheck` 已通过；`graft validate backend --test-target ./plugins/user --test-target ./internal/store/entstore` 仍被既有仓库级 lint backlog 阻断，当前输出包含历史 `cyclop`、`dupl`、`revive`、`gosec`、`staticcheck` 等问题，不属于本次 follow-up 新增回归。
- 当前后端恢复基线沿用最近一次 focused backend validation：
  - `cd server && go test ./internal/cli ./internal/app ./internal/store ./internal/store/entstore ./plugins/user ./plugins/rbac`
  - `cd server && go build ./cmd/graft`
- 当前 live-validation 基线沿用最近一次 disposable PostgreSQL / Redis 验证：
  - `graft migrate up`
  - `atlas migrate status`
  - `graft serve` + `/healthz`
- `graft validate smoke` 已经作为下一次最小闭环验证入口存在；本次文档同步没有新增运行时校验。
- 本次默认管理员/首次改密 server 跟踪同步一致性检查：
  - `rg -n "graft-admin|must_change_password|change-password|全局.*拦截|受限态" ai-plan/design/项目设计.md server/plugins/user/README.md ai-plan/public/mvp-extension-path/subtopics/server`
  - `git diff -- server/plugins/user/README.md ai-plan/public/mvp-extension-path/subtopics/server ai-plan/design/项目设计.md`
- 本次 backend lint 治理文档切片一致性检查：
  - `rg -n "golangci-lint|graft validate backend|controlled exception|revive|stylecheck" AGENTS.md README.md ai-plan/design/项目设计.md ai-plan/design/代码注释与模块文档规范.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
  - `git diff -- AGENTS.md README.md ai-plan/design/项目设计.md ai-plan/design/代码注释与模块文档规范.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md`
- 本次 `AGENTS.md` Go 治理章节扩展一致性检查：
  - `rg -n "Go 代码组织与命名规范|Context 规范|API 与 DTO 规范|Runtime Wiring 与依赖注入规范|安全与鉴权规范|AI 生成代码约束" AGENTS.md`
  - `git diff -- AGENTS.md ai-plan/public/mvp-extension-path/subtopics/server/todos/server-tracking.md ai-plan/public/mvp-extension-path/subtopics/server/traces/server-trace.md`

## Immediate Next Step

- 单开一个 `server test-lint backlog` 切片，专门治理 `server/.golangci.test.yml` 暴露的 117 条历史测试 lint；不要把这批测试清扫再与 workflow 修复或新的功能切片混做一刀。
- 停止继续扩大会话治理宽度，按以下顺序推进 backend MVP closure：
  1. 保持当前 `AUTH_*` code、success/error envelope 与 request-id 契约冻结，不再无边界扩张 auth 响应面。
  2. 在 `user` 插件内实现默认管理员、首次改密持久化状态、`login/bootstrap` 扩展、`change-password` 与最小管理员绑定，但不扩展到 OAuth / SSO / MFA / 密码历史 / 配置化策略 / 全局后端拦截。
  3. 在该稳定契约上继续推动 `web` 的真实页面接线与登录后受限态阻断，不要回退到中文 `message` 分支或重新引入 refresh 多出口。
