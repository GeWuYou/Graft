# 后端主导的 MVP 闭环收敛计划 Server 跟踪

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `server`
- Scope: `server/core`、plugin lifecycle、registries、Ent/Atlas、event bus、audit、scheduler、auth/RBAC 与 backend contract stabilization

## Goal

- 让 `server` 先完成 MVP 必要闭环，再把当前 `web` 所需的真实契约稳定下来，避免继续把主线投入到会话治理宽度扩张。

## Current Recovery Point

- 当前 RBAC MVP 第一波实施已启动并冻结为“只读管理面 + 真值补强”切片：`rbac` 插件开始从“仅提供
  `pluginapi.Authorizer` 的授权插件”扩展为最小只读管理插件，范围仅限角色/权限 canonical contract、
  只读路由、只读仓储接口与 focused tests；本轮不进入角色写操作、用户禁用、用户分配角色或
  `super_admin` bypass。
- 当前 RBAC MVP 第二波最小写 contract 已在 `server/plugins/rbac` 内收口到稳定插件边界：角色创建、角色更新、
  角色权限替换与用户角色替换四类最小写 API 现由 `plugin_write_routes.go`、`write_service.go` 与
  `contract/{route,permission}.go` 共同持有；范围继续明确限制在 replace 语义与稳定 `permission_ids` / `role_ids`
  DTO，不扩展删除角色、禁用用户或 `super_admin` bypass。
- 当前 `server/plugins/rbac` 已补齐“目标用户当前已分配角色”的稳定 HTTP 读契约：`GET /api/users/:id/roles`
  现在由 `rbac` 插件持有，配套新增 `user.role.read` permission、`role_ids` 稳定 DTO、显式目标用户存在性校验与
  focused tests；该契约保持最小只读范围，不复用 `GET /api/roles` 的角色详情所有权，也不引入 `super_admin` bypass。
- 当前 `server` 侧 RBAC 真值同步也已开始进入持久化层：`roles.builtin` 与 `permissions.category`
  已进入 Ent schema / Atlas migration 设计面，`bootstrap` 最小快照开始补齐 `roles`，让后续 `web`
  不再依赖空 roles 数组或本地猜测角色状态。
- 本轮边界明确保持不变：`role/permission` 管理归 `rbac` 插件，`user` 插件继续持有用户与认证链路；
  不允许把角色/权限管理继续堆回 `user` 插件形成第二套插件职责。

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
- 当前 i18n 收敛边界保持不变：`server/internal/i18n` 仍是唯一平台 facade；本轮设计真值只要求先补 registry / namespace / duplicate-key / freeze 语义，不提前声称已经进入 `go-i18n` 接入阶段。
- 插件 i18n 生命周期边界已冻结为：插件可在 `Register` 阶段注册 message bundles / message keys，runtime 必须在进入 `Boot` 前冻结 i18n 注册面；`Boot` 之后新增注册不属于当前允许语义。
- 当前 locale 范围继续只收敛到 `zh-CN` / `en-US`；菜单本地化 contract 已开始实际落地为 `title_key` 优先、`title` 回退：`user` / `rbac` 内建菜单标题现由插件在 `Register` 阶段注册到 `server/internal/i18n`，`bootstrap` 菜单快照同步暴露 `title_key`，供 `web` 后续按同一 contract 收敛。
- `user` 插件现已补齐最小 `GET /api/users` 只读列表契约，继续保持在现有 plugin/store 边界内，不提前扩展分页、筛选和写操作，只为 `web` 当前 `/users` 真实接线提供稳定落点。
- 当前 `auth / RBAC` 最小响应收敛切片已经进入实施准备：只允许修改 `server/internal/httpx` 与 `server/plugins/user` 现有链路，目标是稳定 HTTP status 语义、稳定业务 `code`、稳定 auth/bootstrap envelope，并让 `web` 后续只基于 `HTTP status + code` 处理认证分支。
- 当前下一步认证治理切片边界已冻结：默认管理员账号固定为 `graft`；`graft-admin` 是仅允许在初始化路径写入的例外密码；首次改密状态必须由后端持久化并通过 `login/bootstrap` 返回；当前 MVP 不在本切片内给全部业务接口追加全局“已改密”中间件，而是把登录后受限态阻断交给 `web`，后续如需更强安全再评估服务端全局 hardening。
- 默认管理员不能成为只能登录的空账号；若当前不存在可复用的 `admin` 角色/权限 seed，本切片内必须补最小管理员角色/权限绑定。
- 本轮响应收敛明确冻结以下后端契约：成功响应固定保留 `success / code / message / traceId / data`，错误响应固定保留 `success / code / message / traceId`，`messageKey / locale` 仅作为可选扩展口，禁止继续保留 `error + message` 双字段重复设计。
- 本轮认证失败语义明确收口为：`AUTH_INVALID_CREDENTIALS -> 400`、`AUTH_TOKEN_MISSING -> 401`、`AUTH_TOKEN_EXPIRED -> 401`、`AUTH_TOKEN_INVALID -> 401`、`AUTH_FORBIDDEN -> 403`；其中 `CurrentUser` 在 token 可解析但服务端 session 已失效时必须归入 `AUTH_TOKEN_INVALID`，不能再退化成笼统“未登录”。
- 本轮最小链路补充 `request-id` 约束：后端只做最小 UUID request-id，中间件优先读取 `X-Request-Id` 与 `X-Trace-Id`，缺失时生成并写回 `X-Request-Id`，所有 auth/RBAC envelope 的 `traceId` 都必须取该值，不扩展到 OpenTelemetry 或其它 tracing 基础设施。
- 本轮前后端边界约束同步固定 refresh 行为：只有 `AUTH_TOKEN_EXPIRED` 允许触发一次 refresh；`AUTH_TOKEN_INVALID`、`AUTH_TOKEN_MISSING`、`AUTH_FORBIDDEN` 都禁止 refresh；refresh 失败后必须走单一出口，统一清理本地登录态与缓存并跳转登录页，不能继续 retry 或形成递归。
- 当前 auth / RBAC 响应收敛切片已经完成第一轮直接落地：`server/internal/httpx` 现已稳定输出 `AUTH_TOKEN_EXPIRED`、统一 success/error envelope 与最小 request-id 透传，并补齐对应 direct tests；`server/plugins/user` 的关键写接口成功路径也已补上 envelope 回归断言，避免后续回退成裸 `200`。
- 当前默认管理员首次改密补丁也已落地到 `server/plugins/user`：`change-password` 路由不再在 HTTP 层硬编码要求 `current_password` 非空，而是由 service 基于“默认管理员 + `must_change_password=true` + 当前散列仍匹配 `graft-admin`”判定是否允许空原密码；其它用户仍保持 `400 common.invalid_argument` + `field=current_password` 契约。
- PR #9 当前一轮 AI review 已确认并落地的 `server` 跟进包括：统一审计 `Action` trim 一致性、主动审计事件同时兼容值/指针 payload、bootstrap locale fallback 去重，以及 `pluginapi.AuditEvent`、scheduler 生命周期文档补强。
- PR #9 当前剩余的 greptile `server` 评论已核对到本地 HEAD：`scheduler` 插件尾部未使用的 `logJobFailure` 确认为死代码，`audit` 请求级自动审计已改为把 `ResourceType` 从稳定路由中拆解为资源域，避免继续与 `RequestPath` 重复。
- PR #9 最新 CodeRabbit nitpick 已在本地核对并收敛：`plugin.Context` 现已显式承载 `LifecycleContext`，runtime 会在 `Shutdown` 阶段注入独立有界关闭上下文，`scheduler` 不再绕过宿主生命周期直接使用 `context.Background()`。
- 当前一轮 `server` 架构治理补强切片已落地到代码：`httpx.RequirePermission` 不再在请求热路径依赖 container resolver；`user` 插件路由守卫改为显式 typed wiring，`GET /api/users` 不再在 handler 内直接现取 `store factory`；`scheduler` 运行时改为由 `Boot` 显式绑定生命周期上下文并在 `Shutdown` 收敛取消；`entstore.NewFactory` 改为显式返回错误而不是 `panic`。
- 当前 backend completion 入口已重新回到全绿：`go run ./cmd/graft validate backend` 本地通过，先前记录的 test-lint controlled exception 已清空，不再保留活动例外。
- 当前 `server authz/rbac wiring convergence` 切片也已落地：`user` 路由在 `Register` 阶段只持有延迟绑定的
  `pluginapi.Authorizer` 句柄，并在 `Boot` 阶段解析绑定 `rbac` 插件公开的共享授权器；请求热路径不再 `Resolve`
  服务，也不再保留第二套本地 RBAC 判定逻辑。
- 当前 `server/plugins/user` contract-governance follow-up 也已落地：插件内新增 `contract` 包承载 canonical
  permission 与 auth-route path，`plugin.go` / `plugin_routes.go` 已改为消费平台 `message.Key` 与插件内 typed
  contract，shared `common.conjunction` / `common.copyright` 也已补回 `server` canonical message contract 与
  i18n catalog；本轮 targeted scanner findings 已从运行时文件清零。
- `pluginapi`、registries、store factory 与当前 auth/menu/permission/i18n 返回面，已经成为 `web` 真实契约收敛前必须谨慎冻结的后端边界。
- 当前 i18n 相关设计治理只冻结 facade、注册期与菜单 contract 方向；凡未在设计文档明确的底层实现细节，当前 tracking 不得提前当作已完成实现事实记录。
- 当前本地启动修复已补齐 `server/.env.example` 的显式 auth 密钥示例、README 最小启动步骤与 GoLand working directory 提示，并用隔离环境测试锁定“缺少 `GRAFT_AUTH_JWT_SECRET` 与 `GRAFT_AUTH_SIGNING_KEY` 时严格失败”的配置行为；未引入任何 dev-only 默认密钥或 auth 语义变更。
- `server` 当前已补充两个独立开发辅助程序：`cmd/graft-jwt-secret` 与 `cmd/graft-signing-key`，用于生成可直接写入 `.env` 的随机 auth 密钥文本；该能力只辅助配置准备，不参与运行时加载或 token 语义。
- `server` 当前已把模块工具链基线提升到 `Go 1.26.x`，并将 `go.uber.org/zap` 收敛到 `v1.28.0`；`go test ./...` 与 `go build ./cmd/...` 在本地 `go1.26.1` 下通过，未触发 Ent/Atlas regeneration。
- 当前 backend governance 文档真值已经冻结：`server` 完成态必须统一走 `graft validate backend`，固定质量顺序为 `graft validate backend --stage lint -> go test (smallest direct scope) -> go build ./cmd/graft -> graft validate smoke when needed`，并固定 pin `golangci-lint v2.12.2`。
- 当前 `AGENTS.md` 已新增独立 `Go 代码组织与命名规范` 章节，统一冻结 `server` 手写 Go 代码的文件/包/类型命名、Context 传播、API/DTO、config、runtime wiring、事务、Ent、并发、日志、安全与 AI 生成代码约束；相关完成态与注释/验证章节仅再做引用式收敛，不再重复堆叠第二套真值。
- 当前 CI `server-lint` job 已改为只安装固定版 `golangci-lint` 二进制，再回到 `graft validate backend --stage lint` 统一执行入口；不再让 `golangci-lint-action` 在仓库根目录误跑 `./...`。
- backend blocking lint gate is changed-file scoped against the resolved base branch.
- changed-file scoped means whole-file enforcement on files changed relative to the resolved base branch via
  `--new-from-rev=<merge-base> --whole-files`.
- untouched files are not blocking gate failures.
- full-repo lint is audit-only backlog scanning.
- touching a historical hotspot file means the touched file must satisfy the current lint gate, unless a narrowly
  documented temporary `nolint` is justified.
- new code must not expand the lint backlog.
- backend lint issue 如当前切片无法立即清理，只能登记到本文件的 `Backend Lint Controlled Exceptions`，并记录来源、
  影响、保留原因、负责人或上下文与下一步清理条件。
- 详细实现历史保留在 `subtopics/server/traces/server-trace.md`。

## Backend Lint Controlled Exceptions

- 当前无活动中的 backend lint controlled exception；若后续再次出现无法在当前切片内清理的 backend lint 阻断，只能在本节追加登记。
- 当前 audit backlog hotspots：
  `internal/ent/schema/user_role.go`、`internal/ent/schema/role_permission.go`、`internal/i18n/service.go`、
  `plugins/rbac/plugin_routes.go`、`plugins/rbac/plugin_write_routes.go`。
- 后续如需暂留 backend lint issue，只能在本节追加受控例外，并至少记录以下字段：
  - Source：linter 名称，以及对应的 package / file / command。
  - Impact：用户可见影响或工程可维护性影响。
  - Retention Reason：为什么当前切片不能安全清理。
  - Owner / Context：谁拥有该暂留语义，或者该注释所绑定的 slice / file 上下文。
  - Next Cleanup Action：下一步清理动作或计划承接切片。
  - Cleanup Condition：何时必须移除该临时 `nolint` 或 controlled exception。

## Active Risks

- 当前最大的剩余风险已经从 runtime 闭环转向共享契约漂移；若后端在 `auth + menu + permission + locale` 返回面上继续频繁变动，`web` 接线会反复返工。
- 如果 i18n 在 `server/internal/i18n` facade 之外继续分叉第二套注册/查找入口，或在 freeze 语义未稳定前提前切换底层实现，当前 `server` 与 `web` 契约会再次失去单一真值。
- 如果 auth/RBAC 收敛期间继续混用 “401 + 各类认证失败” 与非稳定 envelope，`web` 将无法只凭 `HTTP status + code` 做出稳定登录态分支，refresh 死循环风险也会重新出现。
- 如果在下一阶段继续无边界扩张 session-governance 细节，会挤占当前最关键的后端闭环资源。
- 如果把首次改密阻断直接扩展成全局后端接口治理，会偏离当前 MVP 范围并放大本轮 auth/RBAC 收敛面的回归风险。
- 若 `pluginapi`、store DTO 或权限/菜单契约在收敛期内继续频繁漂移，`web` 对真实契约的接线成本会快速上升。
- disposable PostgreSQL / Redis 仍需手工准备；恢复执行时必须确认当前可用的 smoke 环境。
- 如果 `server` 本地完成态、agent 完成态与 CI 阻断继续各自维护不同的 lint 命令或参数，backend quality gate 会在实现落地后迅速重新分叉。
- 如果 `server` 继续只有用户角色写接口而没有读接口，`web` 一旦开始接入用户角色分配 UI，就只能靠空初始值或本地猜测
  驱动表单，重新形成假闭环。

## Latest Validation

- 本次 `server/plugins/rbac` 第二波最小写 contract/README/tracking 收口直接校验：
  - `cd server && go test ./plugins/rbac`
  - 结果：角色创建、builtin 角色重命名保护、角色权限 replace、用户角色 replace、目标用户未命中与 TOCTOU ID 漂移映射等 focused route/service tests 通过。
- 本次 `server` 架构治理补强切片直接校验：
  - `cd server && go test ./internal/httpx ./plugins/user ./plugins/audit ./internal/scheduler ./plugins/scheduler ./internal/store/entstore ./internal/app`
  - `cd server && go build ./cmd/graft`
  - `cd server && go run ./cmd/graft validate backend`
  - 结果：统一 backend completion 入口通过；当前无活动中的 backend lint controlled exception。

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
- 本次默认管理员首次改密后端切片直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go test ./plugins/user -run 'TestChangeCurrentUserPassword|TestChangePasswordRoute'`
  - `cd server && go run ./cmd/graft validate backend`
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
- 本次 `server/plugins/user` contract-governance follow-up 直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go test ./internal/i18n ./internal/contract/...`
  - `python3 scripts/magic_value/check_magic_values.py --mode report --output-json /tmp/graft-magic-report-next.json`
  - 结果：`server/plugins/user/plugin.go`、`server/plugins/user/plugin_routes.go` 与
    `server/internal/contract/message/key.go` 不再出现本轮 targeted permission/message-key/auth-route 或
    `common.conjunction` / `common.copyright` drift findings。
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
- 本次 `server/plugins/user/plugin_routes.go` hotspot reduction 直接校验：
  - `cd server && golangci-lint run --config .golangci.yml ./plugins/user --new-from-rev "$(git merge-base HEAD origin/main)" --whole-files`
  - `cd server && go test ./plugins/user`
  - `cd server && go test ./internal/i18n`
  - `cd server && go build ./cmd/graft`
  - 结果：`plugins/user/plugin_routes.go` 的 changed-file scoped lint gate 已回到 `0 issues`。
- 本次 `server/plugins/user/plugin_routes.go` hotspot reduction completion rerun：
  - `cd server && go test ./internal/ent`
  - `cd server && go test ./internal/store/entstore`
  - `cd server && go run ./cmd/graft validate backend`
  - 结果：`internal/ent/runtime.go:150` 的先前 panic 在当前工作树未复现；`internal/ent`、`internal/store/entstore` 与统一 backend completion 入口均已通过，因此这次 hotspot-reduction 切片现在可以按 `server` 完成态认定验证充分。
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
- 本次 `server/plugins/rbac` user-role minimal read contract 直接校验：
  - `cd server && go test ./plugins/rbac ./internal/store/entstore`
  - 结果：`GET /api/users/:id/roles`、`user.role.read` permission 注册、`USER_NOT_FOUND` 未命中语义与稳定 `role_ids`
    快照的 focused tests 通过。

## Immediate Next Step

- 当前 `internal/ent/schema/{user_role.go,role_permission.go}` 的配对重复清理已回到 `0 issues`；若继续执行本轮 audit-backlog reduction，优先转到 `internal/i18n/service.go`，并把 `plugins/rbac/plugin_routes.go` 与 `plugins/rbac/plugin_write_routes.go` 保持为后续同类热点。
- 当前 `server/plugins/rbac` 的第二波最小写 contract、README 与 tracking 真值已经收齐；下一步如果继续推进 `server`
  侧 RBAC，优先决定是否需要补更强的 completion-state backend validation，再决定是否进入更高风险的用户禁用、删除或
  `super_admin` bypass。
- 当前 `server` 在 user-role 管理面上的最小阻断已解除；下一步由 `web` 基于 `GET /api/users/:id/roles` 与
  `POST /api/users/:id/roles/assign` 评估是否进入最小 UI 接线，同时继续把范围限制在 `/users` 模块内的最小角色查看/分配。
- 保持 `bootstrap.roles`、`roles.builtin`、`permissions.category` 的真值收口，后续 `web` 只消费这批后端契约，不得再本地推导角色类别或权限分组。
- 保持当前共享 `pluginapi.Authorizer` wiring 与 `server/plugins/user/contract` 稳定；后续新增 `server`
  受保护路由时，继续复用 typed permission/route contract 与 `rbac` 插件公开服务，不再在 `user` 或其它插件本地复制实现。
- 当前纯 `server` 的 runtime auth/authz contract 热点已经完成一轮清扫；后续若继续做 `server` 治理，优先收敛
  `plugin_test.go` 与其它测试侧仍残留的 auth/shared 字面量，否则跨边界主线回到父主题与 `web` 子主题继续推进主运行面清理。
- 当前 `plugin_routes.go` hotspot-reduction 切片的 backend completion gate 已恢复充分；RBAC 第二波最小写接口的
  contract/README/tracking 收口也已完成，后续如继续推进 `server` 治理，避免重新扩大到无关运行时热点。
