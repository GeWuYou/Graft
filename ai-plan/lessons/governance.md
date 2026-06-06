# Governance Lessons

## LESSON-GOVERNANCE-BROWSER-BACKEND-001：浏览器验收需要真实后端时不要停在 mock 登录页

- Status: active
- Level: L1
- Applies to:
  - `web` 页面浏览器验收
  - 需要登录、bootstrap、动态菜单或真实 API 契约的 Playwright 检查
  - 使用 `graft-web-browser-agent` 或本地 Playwright 脚本做 UI 交互验证的任务
- Source:
  - Scheduled Task 配置弹窗 UX 重构浏览器验收
  - `web` 以 `dev:mock` 启动后，登录页可渲染但 `/api/auth/login` 未形成可用会话，导致目标页面无法进入
- Problem:
  `web` 的 mock dev server 不一定覆盖当前页面所需的认证、bootstrap、动态菜单和业务 API。浏览器验收如果只停留在
  mock 登录页，会把真实 UI 检查误判为“需要继续找选择器”或“页面不可达”，浪费调试时间。
- Correct pattern:
  当目标页面依赖登录态或后端动态契约时，直接启动真实后端：
  `cd server && go run ./cmd/graft dev`。确认后端已注册目标 API 后，再通过 `web/.env.development`
  的 `VITE_API_TARGET` 代理访问页面。若当前目录的 `temp` 下提供本地凭据，可用于本机浏览器验收，但不要把凭据写入代码、
  文档正文或测试 fixture。
- Anti-pattern:
  在 `dev:mock` 登录页反复尝试账号密码、改前端选择器或伪造业务结论，而没有先确认认证接口、bootstrap 和目标业务 API
  是否由 mock server 真实承载。
- Enforcement:
  浏览器验收前先检查页面是否需要认证或真实后端数据；若需要，确认 `server` 已启动并能看到目标路由，例如
  `/api/auth/login`、`/api/auth/bootstrap` 和相关业务 API。若登录仍失败，优先检查后端启动状态和代理配置，而不是继续调整
  UI 选择器。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `.agents/skills/graft-web-browser-agent/SKILL.md`
  - `README.md`
  - `web/.env.development`
- Updated at:
  2026-06-07
