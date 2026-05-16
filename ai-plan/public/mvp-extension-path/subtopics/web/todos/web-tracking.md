# 后端主导的 MVP 闭环收敛计划 Web 跟踪

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `web`
- Scope: starter 壳层、真实后端 `menu + route + page + api + permission` 契约挂接、i18n UI surface 与 frontend validation governance

## Goal

- 在后端主导的 MVP 收敛阶段，把 `web` 约束在真实 `web/` 工程内的 starter 壳层收敛与真实契约接线范围内，不再把新增页面、并行 runtime baseline 或新的前端壳层广度作为近期目标。

## Current Recovery Point

- 当前 RBAC MVP 第一波前端实施已冻结为“共享权限消费基础 + 现有 `/users` 真入口接线增强”切片：只允许修改
  `permission` store、permission directive、`user` store 的 bootstrap roles/permissions 消费，以及当前
  `/users` 页面上的权限显隐；本轮不新增 `/roles` 页面、不新增第二批动态菜单、不做完整 CRUD UI。
- 当前 `web` 侧 RBAC 真值继续保持单一路径：只消费后端 `bootstrap` 快照中的 `roles`、`permissions`、`menus`，
  不允许本地自造 permission code，也不允许把按钮/菜单显隐升级为真实安全边界。
- 当前与 RBAC MVP 第二波方向的跨边界协同也已冻结：`web` 暂不把最小写 API 视为可消费完成态，不新增 `/roles`
  写操作页面，也不提前把按钮显隐升级成提交流程；要等 `server` 对最小写接口完成主代理验证后，再决定下一轮前端接线范围。

- `web` 现阶段以真实 `web/` 工程作为唯一运行面；starter 只保留可继续收敛的壳层风格、页面样板和治理参考，不再把 starter 全量工程视为运行基线。
- 当前主线不是页面扩张，也不是继续深化独立前端工作台能力；任何 shell 级调整都应服务于真实契约挂接和 mock/demo 清理。
- `web/ai-libs/tdesign-vue-next-starter` 继续只作为本地参考源存在，不是当前工程的独立 Git root，也不应在 IDE 里登记为第二个仓库。
- `signals` 已收敛为文档级候选方案：`Pinia` 继续作为唯一正式共享状态层，当前不进入 `setting/theme` 局部试点，只保留未来最小 POC 的准入与退出规则。
- 前端命令真值保持不变：WSL 场景下继续使用 host Windows Bun，完成态仍以 `bun run check` 零 warning 为门槛。
- docs/automation 第一波治理收口已同步到 `web` 边界：README、validation skill、CI workflow 和前端设计文档现在都只把 starter 视为参考源，并把 `bun run check` / host Windows Bun 规则回指到同一套仓库真值。
- PR #9 当前一轮 AI review 已确认并落地的 `web` 跟进包括：登出失败时仍强制跳转登录页、动态路由装配去除双重断言、locale header 全量下划线替换，以及 route guard / bootstrap / token 持久化的中文契约注释补强。
- `/users` 当前已不再复用 starter 个人中心 demo 页，而是直接消费真实 `GET /api/users` 最小只读契约；对应的 `/user/index` 静态入口与 header 残留跳转也已移除，避免双轨导航。
- 开发环境下的请求策略已收敛为“前端统一请求相对 `/api` 路径，由 Vite proxy 转发到 `VITE_API_TARGET`”；只有显式关闭代理时，Axios 才会直连后端绝对地址。
- 前端环境文件治理当前已与 `server` 对齐：提交 `web/.env.example` 作为共享模板，忽略真实 `web/.env.*` 本地开发配置，不再把机器专属开发地址直接提交进仓库。
- 当前 auth 响应收敛切片已经完成第一轮前端落地：请求层只对 `AUTH_TOKEN_EXPIRED` 触发一次 refresh，`AUTH_TOKEN_INVALID` / `AUTH_TOKEN_MISSING` 统一走单一清理出口并跳转登录；请求层与 `user` store 之间的登录态同步已收敛为显式 session bridge，避免动态导入 store 带来的构建 warning 与双源状态漂移。
- 当前下一步认证治理切片的 `web` 边界已冻结：首次改密真值只能来自后端 `login/bootstrap`，前端不得通过用户名 `graft` 或默认密码猜测；当前 MVP 的业务阻断由登录后受限态与强制改密弹窗完成，不新增独立安全插件、不改 refresh 单出口，也不把控制流建立在中文 `message` 上。
- 当前首次改密受限态切片已落地到 `web`：`must_change_password=true` 现在明确表示“已认证但受限”，路由守卫会把业务路由统一拦到静态 `/auth/restricted-session` 入口并保留 token；改密成功后必须按 `change-password -> bootstrap(true) -> rebuild routes -> restore navigation` 顺序恢复，不本地直接把 `must_change_password` 改成 `false`。
- 当前运行面治理已冻结单一路径：`bootstrap -> module registry -> route -> page`。主运行面不得再保留 demo route、playground page、独立 mock page、feature 自带 runtime 或绕过 bootstrap/menu 的入口。
- PR #10 的最近一次 review follow-up 已补齐用户页版权年份、用户页列表页文案 i18n、接口说明 `<code>` 展示、用户页样式深度选择器兼容写法，以及 `request` / `user` store 在 refresh 失败路径上的重复清理与重复 refresh 防护；相关 `web` 完成态校验继续以 host Windows Bun `bun run check` 为准。
- 详细前端实现历史保留在 `subtopics/web/traces/web-trace.md`。

## Active Risks

- 如果 `web` 回到页面扩张、长期保留 starter demo/mock 流程，或重新把 starter 全量工程写成运行基线，前端会再次偏离“后端主导的 MVP 闭环收敛”主线。
- 如果主路由树继续保留 starter homepage/result/demo 入口、`permission-fe` 旁路或默认 mock runtime，前端会重新形成假闭环并继续绕开后端契约。
- 如果后端共享契约在收敛期内继续频繁漂移，starter 壳层的真实接线会产生反复返工。
- 如果 `web` 重新通过用户名、默认密码或 message 文案猜测首次改密状态，后续改密弹窗、路由受限态和 bootstrap 恢复都会失去稳定真值。
- 混用 WSL Bun 与 host Windows Bun 仍可能破坏当前工作树的前端依赖与 IDE 运行稳定性。
- 如果 IDE 把 `web/ai-libs/tdesign-vue-next-starter` 重新登记成额外 Git root，仓库视图会混入参考目录历史与标签，影响当前主仓提交判断。
- 如果 README、skill 或 workflow 再把 CI stage / Linux runner 语义写成独立前端验收规则，当前 frontend validation governance 会重新分叉。
- 如果 `web` 在后端最小写 API 仍处于 in-progress 时就提前扩展角色管理写页面，前端会再次基于未冻结契约形成假闭环。

## Latest Validation

- 当前前端恢复基线沿用最近一次 host Windows Bun 完成态校验：
  - `bun run check`
- 该完成态基线要求 `format:check`、`typecheck`、`lint`、`stylelint`、`test:run`、`build` 全部通过且无未处理 warning。
- 本次 web/topic 文档同步未新增任何基于 RBAC 第二波写接口的前端通过声明；相关接线仍保持待后端验证后的 planned 状态。
- 本次 PR #9 review follow-up 预期直接校验：
  - `cd web && bun run check`
- 本次 `/users` 真实列表页切片直接校验：
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- 本次 Git 边界修复直接校验：
  - `git rev-parse --show-toplevel`
  - `git status --short --branch`
  - `git tag --list`
  - `git worktree list --porcelain`
- 本次文档同步通过 `rg`、`sed` 与 `git diff -- ai-plan/design/前端架构设计.md ai-plan/public/mvp-extension-path/subtopics/web` 进行一致性检查。
- 本次登录页控制台报错修复预期直接校验：
  - `cd web && bun run typecheck`
  - `cd web && bun run build`
- 本次前端环境文件治理修复预期直接校验：
  - `git check-ignore -v web/.env.development web/.env.local`
  - `git ls-files web/.env.example web/.env.development`
- 本次 auth 响应收敛切片实际直接校验：
  - `cd web && bun run test:run -- src/utils/request.test.ts src/store/modules/user.test.ts src/utils/route/bootstrap.test.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`
- 本次首次改密受限态切片直接校验：
  - `cd web && bun run test:run -- src/store/modules/user.test.ts src/utils/request.test.ts src/layouts/components/force-password-change.test.ts src/permission.test.ts`
  - `cd web && bun run typecheck`
- 本次 PR #10 review follow-up 实际直接校验：
  - `cd web && bun run check`
- 本次默认管理员/首次改密 web 跟踪同步一致性检查：
  - `rg -n "graft-admin|must_change_password|change-password|受限态|bootstrap" ai-plan/design/项目设计.md server/plugins/user/README.md ai-plan/public/mvp-extension-path/subtopics/web`
  - `git diff -- ai-plan/public/mvp-extension-path/subtopics/web ai-plan/design/项目设计.md server/plugins/user/README.md`
- 本次 docs/automation 治理收口同步一致性检查：
  - `rg -n "starter|运行基线|bun run check|host Windows Bun|第二真值|execution-layer" ai-plan/design/前端架构设计.md README.md .agents/skills/graft-validation-runner/SKILL.md .github/workflows/pull-request-validation.yml .ai/environment/README.md ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
  - `git diff -- ai-plan/design/前端架构设计.md README.md .agents/skills/graft-validation-runner/SKILL.md .github/workflows/pull-request-validation.yml .ai/environment/README.md ai-plan/public/mvp-extension-path/subtopics/web/todos/web-tracking.md`
- 本次 web 主运行面收口预期直接校验：
  - `rg -n "dashboard/base|vite-plugin-mock|permission-fe|tabs-router" web/src web/package.json web/vite.config.ts`
  - `cd web && bun run typecheck`
  - `cd web && bun run check`

## Immediate Next Step

- 先把 permission helper / directive 和 `/users` 页最小权限显隐做实，再决定是否进入 `/roles` 新页面；不要在当前切片里同时扩展第二批动态菜单映射。
- 在后端最小写 API 仍未完成主代理验证前，保持 `web` 对 RBAC 第二波的状态为“等待稳定契约 + 等待验证结果”，不要提前开启角色写操作 UI。
- 保持 `bootstrap.roles` 和 `bootstrap.permissions` 作为唯一前端 RBAC 快照来源，不要回到基于页面本地常量或角色名字符串的条件分支。
- 继续在真实 `web/` 工程里把 starter 壳层风格挂接到真实后端 `auth + current user + menu + permission + locale` 契约。
- 先清理主路由树里的 starter demo 入口、默认 mock runtime 与前端权限旁路，让主运行面重新只服务真实 bootstrap 菜单和已注册页面。
- 快速隔离或移除当前阶段不再需要的 mock/demo 入口，避免形成前端自洽假闭环。
- 在当前 auth 契约与刷新单出口已经稳定后，优先把首次登录强制改密受限态接进现有 `login -> refresh -> bootstrap` 恢复链路，确保刷新页面后仍能恢复弹窗与阻断，而不是再回到请求层分支治理或视觉扩张。
- 保持当前受限态入口与恢复链路稳定，不要再把 `must_change_password` 回退成“未登录”清理路径，也不要在前端本地伪造改密完成状态。

## 2026-05-15 真实 auth/bootstrap 接线恢复点

- `web` 认证主路径已从本地 mock 收敛为真实 `POST /api/auth/login -> GET /api/auth/bootstrap`。
- 当存在 access token 但 bootstrap 失败时，前端会先尝试 `POST /api/auth/refresh`，成功后再重新执行 bootstrap。
- 当本地没有 access token 时，前端也会先静默尝试 `POST /api/auth/refresh -> GET /api/auth/bootstrap`，仅在刷新失败后才回退到登录页。
- `permission` store 当前只消费 bootstrap 返回的真实 `menus` 快照，并按当前已存在的页面实现生成最小动态路由；本轮只接入 `/users`。
- 当前最小动态菜单策略是“菜单展示只保留首页和后端返回且前端已经具备页面实现的菜单项”，避免再次回退到 starter demo 菜单树。
- `/users` 页面当前已替换为最小真实用户列表页，不再依赖本地 profile、图表或团队成员 demo 数据。
- 下一轮首次改密切片必须复用同一 bootstrap 恢复链路：`must_change_password=true` 由后端返回，前端以受限态和不可绕过弹窗阻断业务使用，不通过用户名 `graft` 或默认密码做任何前端猜测。
- 登录页当前不再把 `http://127.0.0.1:3000` 暴露为代理模式下的浏览器请求主机；控制台里看到的 API URL 应保持为相对 `/api/...`，由 Vite 开发代理负责转发。
- `web` 本地开发配置当前应从 `web/.env.example` 派生，并保持真实 `web/.env.development` 未跟踪，避免继续把个人开发地址或临时联调配置写入 Git 历史。
