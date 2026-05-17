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
- 当前第一波 AGENTS 合规整治实施切片已冻结为“治理真值收口 + `web` 主运行面收口”，不扩展业务范围，不进入新的页面广度，也不新增第二套 runtime、第二套 lifecycle 或第二套 validation 入口。
- 当前仓库级 startup governance 最小迁移切片已落地到文档真值：根 `AGENTS.md` 现在独占 startup preflight、
  最小 receipt、resume/restart 重验与 subagent 最小继承包定义；`graft-boot`、`graft-multi-agent-batch` 与
  `ai-plan` 文档不再各自维护第二套 boot 链。
- `server` 当前已形成最小 runtime 闭环，并新增受保护的 `/api/auth/bootstrap` 真实契约，统一返回当前用户、权限码、按权限过滤的菜单和 locale 快照。
- `web` 当前已把登录主路径收敛到真实 `login/refresh/bootstrap` 契约，并开始用 bootstrap 菜单快照驱动最小动态路由，而不是继续依赖 starter mock 登录与静态 demo 菜单。
- 当前 cross-boundary 切片已把 `/users` 从“真实菜单路径 + demo 页面内容”的假闭环，收敛为真实 `GET /api/users` 契约加最小只读列表页；`/user/index` 静态个人中心残留入口已移除。
- 当前 auth / RBAC 最小响应收敛切片也已完成第一轮跨边界对齐：`server` 已冻结 `AUTH_*` 失败 code、统一 envelope 与 request-id 透传，`web` 已冻结“仅 `AUTH_TOKEN_EXPIRED` refresh，`INVALID/MISSING` 单一退出”的请求层行为，并通过显式 session bridge 消除了构建 warning。
- 当前默认管理员与首次登录强制改密切片的跨边界口径已冻结：`graft-admin` 只允许作为默认管理员初始化例外密码写入；首次改密状态必须由后端持久化并通过 `login/bootstrap` 返回；当前 MVP 不为全部业务接口增加全局后端拦截，而由 `web` 登录后受限态负责阻断，后续如需更强安全再补服务端全局中间件。
- 当前默认管理员首次改密切片已进一步落地：`must_change_password=true` 现在明确表示“已认证但受限”，`web` 通过受限态路由守卫阻断业务入口但保留 token；`server` 仅在“默认管理员 + 受限态 + 当前仍是初始化例外密码”时允许 `change-password` 省略 `current_password`，改密成功后必须重新 `bootstrap` 恢复正常导航。
- 当前仓库真值还新增冻结了一条 shared backend governance baseline：`server` 完成态必须收敛到 `graft validate backend`，
  固定使用 `golangci-lint v2.12.2`，并由 `graft validate backend --stage lint` 作为唯一 blocking lint gate。
- backend blocking lint gate is changed-file scoped against the resolved base branch.
- changed-file scoped means whole-file enforcement on files changed relative to the resolved base branch via
  `--new-from-rev=<merge-base> --whole-files`.
- untouched files are not blocking gate failures.
- full-repo lint is audit-only backlog scanning.
- touching a historical hotspot file means the touched file must satisfy the current lint gate, unless a narrowly
  documented temporary `nolint` is justified.
- new code must not expand the lint backlog.
- 当前 `server-lint` CI 路径已修正为“只安装 pinned `golangci-lint`，再由 `graft validate backend --stage lint` 执行统一入口”；full-repo `golangci-lint run` 仅保留为 audit-only backlog 扫描与 artifact 输出。
- 当前已知 audit backlog hotspots：
  `internal/ent/schema/user_role.go`、`internal/ent/schema/role_permission.go`、`internal/i18n/service.go`、
  `plugins/rbac/plugin_routes.go`、`plugins/rbac/plugin_write_routes.go`、`plugins/user/plugin_routes.go`。
- 当前 `server authz/rbac wiring convergence` 也已落地：`user` 路由守卫不再维护本地 `routeAuthorizer` 语义副本，而是在
  `Boot` 阶段绑定 `rbac` 插件公开的共享 `pluginapi.Authorizer`，请求热路径不再解析容器服务。
- 当前恢复主线已进入 RBAC MVP 第二波方向同步：`server/plugins/rbac` 的最小写接口能力已进入活动实现范围，当前文档真值按
  “角色写接口 + 角色权限分配 + 用户角色分配的最小闭环”登记方向，但本主题文档不会把该方向表述成已完成，也不会替主代理声明
  尚未重新执行的 backend / cross-boundary 验证通过。
- 当前 `web /user-role minimal read wiring` 阶段阻断已由 `server/plugins/rbac` 本轮最小收口解除：后端现已补齐
  `GET /api/users/:id/roles` 读契约、`user.role.read` permission、`role_ids` 稳定 DTO、显式目标用户存在性校验与
  focused tests；因此下一轮可以评估 `web` 是否进入 user-role UI 最小接线，但仍不得借机扩完整角色中心或把写接口包装成假闭环。
- 当前第一波治理切片要求把以下仓库级约束写回真值并开始进入执行层：`bootstrap -> module registry -> route -> page` 单一运行面、`register -> boot -> dispose(optional)` 单一生命周期、resolver 只允许存在于 composition root wiring、`web/src/modules/<name>` 与 `server/plugins/<name>` 作为默认 feature boundary、CI 只作为治理执行层而不是第二真值来源。
- docs/automation 第一波治理收口已同步完成：根 `AGENTS.md` 进一步冻结 runtime surface、module lifecycle、
  service locator、feature boundary、AI architecture preservation 与 validation governance；前端设计文档不再把
  starter 全量工程写成临时运行基线；README、validation skill、CI workflow 与环境说明已统一改为引用仓库入口，
  并明确 split stage 只是执行层，不是第二真值。
- 当前 contract governance / magic-value governance 的 phase-1 底座也已开始进入执行层：根 `AGENTS.md`、
  `项目设计`、`前端架构设计`、`插件与依赖注入设计` 与新建的 `契约治理与魔法值治理规范` 已统一冻结 canonical
  ownership、typed contract、lifecycle 与 compatibility 规则；本地 hooks、CI workflow 与仓库扫描脚本开始用同一套
  phase-1 规则阻断新增高风险裸字面量，而不是继续依赖人工记忆。
- 当前第一批真实 auth contract 收口也已完成：`server/internal/contract/**` 与 `web/src/contracts/**` 现在开始承载
  auth error code、message key、header、auth scheme、auth API path、restricted-session route 与 storage key 的
  canonical ownership；`httpx`、`i18n`、`request`、`router`、`user store` 与相关恢复链路不再各自散落维护同义字面量。
- 当前这批 auth contract 收口同时修正了 phase-1 scanner 的真值入口：definition-context 已识别 `server/internal/contract/**`
  与 `web/src/contracts/**`，error-code / message-key drift report 也已改为从 canonical contract 文件读取，而不是继续依赖
  旧的消费侧字面量位置。
- 当前 `server/plugins/user` 的 permission / message key / auth route 热点与 shared `common.conjunction`、
  `common.copyright` drift 也已完成本轮治理：`server/plugins/user/contract` 现在承载插件内 canonical permission
  与 auth-route path；`plugin.go` / `plugin_routes.go` 已改为消费 typed contract 与平台 `message.Key`；phase-1
  scanner report 不再对这些运行时文件报出本轮 targeted findings。
- 当前 Phase 2 title/message follow-up 也已开始从 design direction 进入运行时落地：`server/internal/menu` 与
  `GET /api/auth/bootstrap` 菜单快照现在新增 `title_key` 并保留 `title` fallback，`user` / `rbac` 插件在
  `Register` 阶段把内建菜单标题注册到统一 `server/internal/i18n` facade，而不是继续把菜单本地化文案停留在裸标题常量。
- 较早的拆分前历史保留在 `archive/`，具体实现轨迹保留在各自 `trace` 文件。

## Shared Milestones

- `mvp-extension-path` 已稳定为父主题 + `server` / `web` 子主题的恢复结构。
- 后端已具备最小可运行的 plugin-oriented runtime、迁移链路、认证与 RBAC 基线。
- 前端已具备可继续收敛的 starter 壳层，并已重新挂回真实认证与最小动态菜单入口，host Windows Bun `bun run check` 当前再次通过零 warning 完成门槛。
- 当前主题阶段已从“后端先闭环”推进到“后端稳定 bootstrap 契约，前端按该契约同步收敛”。
- 当前主题阶段还额外完成了 auth 响应面与 refresh 单出口的第一轮跨边界收口，后续可以把主线重心放回真实页面接线而不是继续补请求层控制流分支。
- 后端 lint 治理的仓库真值已经先行冻结：后续 `server` 与 cross-boundary 收尾都必须以同一个 backend quality entrypoint 和同一套 lint 口径完成，而不是继续依赖散落的临时命令。
- docs/automation 侧的第一波治理收口也已冻结：starter 参考源、统一验证入口、host Windows Bun 例外规则和 CI
  stage 语义现在都回指同一套仓库真值，不再保留“临时运行基线”或“workflow 自有验收口径”的文档空间。
- contract governance / magic-value governance 的第一波 phase-1 底座已经冻结：高风险 contract 的 canonical
  ownership、typed boundary、lifecycle、baseline/allowlist 与 drift-report 目标现已同时写入设计真值、仓库守卫和
  automation 入口。
- 第一批真实 auth contract 也已从“文档与守卫基线”推进到“运行时消费面收口”：`server` 与 `web` 的 auth/request/route/storage
  热路径现在开始复用 canonical contract，而不是继续各自散落定义。

## Shared Risks

- 如果 `web` 回到页面扩张或长期保留 mock/demo 依赖，前后端契约会再次漂移。
- 如果治理文档、workflow、skill 与实际入口同时保留“统一入口 + 手工分步链路”的等价表述，仓库会重新退化成双轨治理，AI 与后续开发会继续选择成本最低的旧路径。
- 如果后续 README、skill、workflow、环境说明或 tracking 文档再次把 starter 全量工程写成运行基线，或把 split
  stage / CI job 写成独立验收规则，docs/automation 真值会重新分叉。
- `auth`、`menu`、`permission`、`locale` 等共享契约若在后端收敛期内继续无边界扩张，会放大 `web` 接线返工成本。
- 如果后续 skill、README、或 topic 恢复文档再次复制完整 boot 规则，当前 startup governance 收口会重新退化成
  双轨并存状态，`resume` 与 subagent 入口会最先失真。
- 如果默认管理员、首次改密真值来源、或登录后阻断责任在 `server` 与 `web` 之间重新漂移，后续实现会重新引入猜测式前端逻辑或半生效的安全边界。
- disposable PostgreSQL / Redis 校验仍依赖手工准备环境，后续恢复时必须显式说明当前可用的校验入口。
- 如果本地、agent 和 CI 在 `server` 完成态上继续各自维护不同的 lint 命令或参数，新的 backend 治理基线会很快再次失真，无法稳定阻断可维护性回退。
- 如果 contract scanner、allowlist/baseline 元数据与设计文档的 canonical ownership/lifecycle 口径重新分叉，
  CI 和本地 hook 会很快退化成“有工具但没有同一套 contract 真值”的伪治理状态。
- 如果接下来把 permission、message key、auth route 等后续热点继续直接堆回 `plugin_routes.go`、`request.ts`、
  `router/index.ts` 或其它消费点，当前刚建立的 canonical contract surface 会很快再次失真。
- 如果任务交接继续只写“next step”而不附带下一任务 startup prompt，或者在交接前跳过对已验证切片的 scoped commit 判断，
  当前 startup governance 与 git workflow 会重新退回隐式上下文驱动状态。
- 如果 `web` 在缺少“目标用户当前角色”读契约时提前接入用户角色分配 UI，前后端会再次形成“可提交但不可核对初始状态”的
  假闭环，并把后续 server contract 返工压力转嫁给页面层。

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
- 本次默认管理员首次改密受限态切片直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run test:run -- src/store/modules/user.test.ts src/utils/request.test.ts src/layouts/components/force-password-change.test.ts src/permission.test.ts`
  - `cd web && bun run typecheck`
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
  - 结果：生产配置通过；测试配置已回到统一验证口径，不再被既有 test-lint backlog 阻断。
- 本次 docs/automation 治理收口一致性检查：
  - `rg -n "runtime surface|module lifecycle|service locator|feature boundary|第二真值|bun run check|host Windows Bun|execution-layer|临时运行基线" AGENTS.md README.md ai-plan/design/前端架构设计.md .agents/skills/graft-validation-runner/SKILL.md .github/workflows/pull-request-validation.yml .ai/environment/README.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
  - `python3 -c "import pathlib, yaml; yaml.safe_load(pathlib.Path('.github/workflows/pull-request-validation.yml').read_text())"`
  - `git diff -- AGENTS.md README.md ai-plan/design/前端架构设计.md .agents/skills/graft-validation-runner/SKILL.md .github/workflows/pull-request-validation.yml .ai/environment/README.md ai-plan/public/mvp-extension-path/todos/mvp-extension-path-tracking.md ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
- 本次 contract governance / magic-value governance 底座切片直接校验：
  - `python3 -m py_compile scripts/magic_value/check_magic_values.py`
  - `python3 scripts/magic_value/check_magic_values.py --mode changed`
  - `python3 scripts/magic_value/check_magic_values.py --mode report --output-json /tmp/contract-governance-report.json`
  - `python3 -c "import pathlib, yaml; yaml.safe_load(pathlib.Path('.github/workflows/pull-request-validation.yml').read_text())"`
  - `sh -n .husky/pre-commit .husky/pre-push`
  - `git diff -- AGENTS.md .gitignore .husky/pre-commit .husky/pre-push .github/workflows/pull-request-validation.yml ai-plan/design ai-plan/public/mvp-extension-path scripts/magic_value`
- 本次第一批 auth contract 收口切片直接校验：
  - `cd server && go test ./internal/httpx ./internal/i18n ./internal/contract/...`
  - `python3 scripts/magic_value/check_magic_values.py --mode changed`
  - `python3 scripts/magic_value/check_magic_values.py --mode report --output-json /tmp/contract-governance-report-next.json`
  - `cd web && bun run test:run -- src/utils/request.test.ts src/store/modules/user.test.ts src/permission.test.ts src/layouts/components/force-password-change.test.ts`
  - `cd web && bun run typecheck`
- 本次 `server/plugins/user` contract-governance follow-up 直接校验：
  - `cd server && go test ./plugins/user`
  - `cd server && go test ./internal/i18n ./internal/contract/...`
  - `python3 scripts/magic_value/check_magic_values.py --mode report --output-json /tmp/graft-magic-report-next.json`
  - 结果：`server/plugins/user/plugin.go`、`server/plugins/user/plugin_routes.go` 与
    `server/internal/contract/message/key.go` 不再出现本轮 targeted permission/message-key/auth-route 或
    `common.conjunction` / `common.copyright` drift findings。

## Immediate Next Step

- 保持新的交接治理真值：当当前切片结束并移交下一任务时，先按 `graft-commit` 风格判断是否可以安全提交当前已验证范围，再在交接文本中附带下一任务 startup prompt，避免下一轮从隐式上下文继续。
- 在 RBAC MVP 第二波方向上，继续把焦点放在 `server/plugins/rbac` 的最小写接口与 shared contract 稳定化；若主代理尚未重新跑通 `graft validate backend`，或在 cross-boundary 收口时尚未同时跑通 backend completion entrypoint 与 host Windows Bun `bun run check`，相关 tracking 继续保持 in-progress 语气，且该切片不得标记为 done。
- 保持 docs/automation 侧新收口的真值稳定，不要再把 starter 全量工程、split stage 或环境例外规则复制成新的并行治理文本。
- 在 phase-1 底座提交后，继续把魔法值治理推进到真实 contract surface：优先从 `server/internal/httpx`、`server/internal/pluginapi`、
  `server/plugins/user/contract` 与 `web/src/contracts` / `web/src/modules/*/contract` 建立首批 canonical typed
  contract，而不是先做全仓“零字面量”清扫。
- 在 `server/plugins/user` runtime hotspots 与 shared `common.*` drift 已收口后，下一步优先处理 `server` / `web`
  tests 与其它消费侧仍残留的 auth/shared route/message 字面量，不要回到全仓泛扫或页面广度扩张。
- 在当前 backend completion 与 shared authorizer wiring 都已收口后，把跨边界主线切回 `web` 主运行面清理：继续移除 starter demo 入口、默认 mock runtime 与前端权限旁路，让主运行面只服务真实 bootstrap 菜单和已注册页面。
- 保持当前 `/api/auth/bootstrap`、`AUTH_*` code 与共享 permission 契约冻结，不要回退到第二套授权实现、中文 `message` 分支或 refresh 多出口。
- RBAC 下一步把主线切到 `web`：基于已稳定的 `GET /api/users/:id/roles` 最小读契约，评估是否进入 user-role UI
  最小接线；范围继续保持在 `/users` 页或其模块内的最小角色查看/分配，不扩完整角色中心、不新增 `super_admin`
  bypass，也不把一次性写入表单包装成完成态。
