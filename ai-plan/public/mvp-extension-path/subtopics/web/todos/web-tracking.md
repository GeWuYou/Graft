# 后端主导的 MVP 闭环收敛计划 Web 跟踪

## Subtopic

- Parent Topic: `mvp-extension-path`
- Subtopic: `web`
- Scope: starter 壳层、真实后端 `menu + route + page + api + permission` 契约挂接、i18n UI surface 与 frontend validation governance

## Goal

- 在后端主导的 MVP 收敛阶段，把 `web` 约束在 starter 壳层收敛与真实契约接线范围内，不再把新增页面或新的前端壳层广度作为近期目标。

## Current Recovery Point

- `web` 现阶段只保留一个可继续收敛的 starter 壳层基线，用于接入真实后端 `auth`、动态菜单、权限门禁和 locale 契约。
- 当前主线不是页面扩张，也不是继续深化独立前端工作台能力；任何 shell 级调整都应服务于真实契约挂接和 mock/demo 清理。
- `web/ai-libs/tdesign-vue-next-starter` 继续只作为本地参考源存在，不是当前工程的独立 Git root，也不应在 IDE 里登记为第二个仓库。
- `signals` 已收敛为文档级候选方案：`Pinia` 继续作为唯一正式共享状态层，当前不进入 `setting/theme` 局部试点，只保留未来最小 POC 的准入与退出规则。
- 前端命令真值保持不变：WSL 场景下继续使用 host Windows Bun，完成态仍以 `bun run check` 零 warning 为门槛。
- PR #9 当前一轮 AI review 已确认并落地的 `web` 跟进包括：登出失败时仍强制跳转登录页、动态路由装配去除双重断言、locale header 全量下划线替换，以及 route guard / bootstrap / token 持久化的中文契约注释补强。
- `/users` 当前已不再复用 starter 个人中心 demo 页，而是直接消费真实 `GET /api/users` 最小只读契约；对应的 `/user/index` 静态入口与 header 残留跳转也已移除，避免双轨导航。
- 开发环境下的请求策略已收敛为“前端统一请求相对 `/api` 路径，由 Vite proxy 转发到 `VITE_API_TARGET`”；只有显式关闭代理时，Axios 才会直连后端绝对地址。
- 前端环境文件治理当前已与 `server` 对齐：提交 `web/.env.example` 作为共享模板，忽略真实 `web/.env.*` 本地开发配置，不再把机器专属开发地址直接提交进仓库。
- 当前 auth 响应收敛切片已经完成第一轮前端落地：请求层只对 `AUTH_TOKEN_EXPIRED` 触发一次 refresh，`AUTH_TOKEN_INVALID` / `AUTH_TOKEN_MISSING` 统一走单一清理出口并跳转登录；请求层与 `user` store 之间的登录态同步已收敛为显式 session bridge，避免动态导入 store 带来的构建 warning 与双源状态漂移。
- PR #10 的最近一次 review follow-up 已补齐用户页版权年份、用户页样式深度选择器兼容写法，以及 `AUTH_TOKEN_INVALID / AUTH_TOKEN_MISSING` 分支不触发 refresh 的请求序列断言；相关 `web` 完成态校验继续以 host Windows Bun `bun run check` 为准。
- 详细前端实现历史保留在 `subtopics/web/traces/web-trace.md`。

## Active Risks

- 如果 `web` 回到页面扩张、长期保留 starter demo/mock 流程，前端会再次偏离“后端主导的 MVP 闭环收敛”主线。
- 如果后端共享契约在收敛期内继续频繁漂移，starter 壳层的真实接线会产生反复返工。
- 混用 WSL Bun 与 host Windows Bun 仍可能破坏当前工作树的前端依赖与 IDE 运行稳定性。
- 如果 IDE 把 `web/ai-libs/tdesign-vue-next-starter` 重新登记成额外 Git root，仓库视图会混入参考目录历史与标签，影响当前主仓提交判断。

## Latest Validation

- 当前前端恢复基线沿用最近一次 host Windows Bun 完成态校验：
  - `bun run check`
- 该完成态基线要求 `format:check`、`typecheck`、`lint`、`stylelint`、`test:run`、`build` 全部通过且无未处理 warning。
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
- 本次 PR #10 review follow-up 实际直接校验：
  - `cd web && bun run check`

## Immediate Next Step

- 继续把 starter 壳层挂接到真实后端 `auth + current user + menu + permission + locale` 契约。
- 快速隔离或移除当前阶段不再需要的 mock/demo 入口，避免形成前端自洽假闭环。
- 在当前 auth 契约与刷新单出口已经稳定后，优先沿现有用户列表落点继续接用户详情与会话治理页面，而不是再回到请求层分支治理或视觉扩张。

## 2026-05-15 真实 auth/bootstrap 接线恢复点

- `web` 认证主路径已从本地 mock 收敛为真实 `POST /api/auth/login -> GET /api/auth/bootstrap`。
- 当存在 access token 但 bootstrap 失败时，前端会先尝试 `POST /api/auth/refresh`，成功后再重新执行 bootstrap。
- 当本地没有 access token 时，前端也会先静默尝试 `POST /api/auth/refresh -> GET /api/auth/bootstrap`，仅在刷新失败后才回退到登录页。
- `permission` store 当前只消费 bootstrap 返回的真实 `menus` 快照，并按当前已存在的页面实现生成最小动态路由；本轮只接入 `/users`。
- 当前最小动态菜单策略是“菜单展示只保留首页和后端返回且前端已经具备页面实现的菜单项”，避免再次回退到 starter demo 菜单树。
- `/users` 页面当前已替换为最小真实用户列表页，不再依赖本地 profile、图表或团队成员 demo 数据。
- 登录页当前不再把 `http://127.0.0.1:3000` 暴露为代理模式下的浏览器请求主机；控制台里看到的 API URL 应保持为相对 `/api/...`，由 Vite 开发代理负责转发。
- `web` 本地开发配置当前应从 `web/.env.example` 派生，并保持真实 `web/.env.development` 未跟踪，避免继续把个人开发地址或临时联调配置写入 Git 历史。
