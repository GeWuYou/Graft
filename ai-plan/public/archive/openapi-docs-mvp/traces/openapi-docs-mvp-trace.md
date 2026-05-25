# OpenAPI Docs MVP Trace

## 2026-05-24

- 按 root `AGENTS.md` 完成 startup preflight，任务最终按 `cross-boundary` 落地：
  - `server` 增加核心 docs/spec 暴露路由
  - `web` 仅增加壳层快捷按钮，不接菜单
- 选择在 `server/internal/app/registerCoreRoutes()` 挂载文档路由，而不是放进业务 plugin 或 `httpx`。
- 复用现有 `kin-openapi` 依赖，把嵌入的 `openapi/openapi.yaml` 解析为 JSON 输出到 `/openapi.json`。
- 同时保留 `/openapi.yaml` 便于核对原始 spec。
- `GET /docs` 返回最小 Scalar CDN HTML，不引入新的 Go/Bun 依赖。
- docs 暴露策略采用：
  - `local/dev/development/test` 默认开启
  - `prod/production` 默认关闭
  - `GRAFT_DOCS_ENABLED` 显式覆盖
- `web` 壳层在 GitHub 按钮旁增加了文档按钮，登录页头部也同步补齐。
- GitHub 链接已从 starter 仓库替换为 `https://github.com/GeWuYou/Graft`。

## 2026-05-24 Follow-up

- 确认 MVP 首版 `/openapi.json` 实际暴露的是 unbundled root spec，而不是 Scalar 可直接消费的 bundled 文档。
- 根文档保留了 `./paths/**` 与 `./components/**` 外部 `$ref`，导致 `/docs` 只能显示 tags / models，无法展开具体 operation。
- follow-up 修复把 `/openapi.json` 改为读取 `openapi/dist/openapi.bundle.json`，同时保留 `/openapi.yaml` 作为源文档入口。
- 后续仍需补充 bundle stale gate，防止 checked-in artifact 漂移。

## Validation Intent

- `git diff --check`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check`

## Known Risks

- docs 开关当前仍是全局开关，不区分更细粒度的网段、认证或角色。
- `/docs` 依赖外部 CDN；离线环境下页面不可用，但 `/openapi.json` 与 `/openapi.yaml` 仍可访问。
