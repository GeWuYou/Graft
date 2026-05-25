# OpenAPI Docs MVP

## Topic

- Topic: `openapi-docs-mvp`
- Task class: `cross-boundary`
- Branch: `feat/wt-oapi-codegen-types-only-spike`
- Recovery source: current repository state + `ai-plan/public/openapi-codegen-governance-audit`

## Goal

为当前 Graft server 提供最小 OpenAPI 文档闭环：

- `GET /openapi.json`
- `GET /openapi.yaml`
- `GET /docs`
- dev/test 默认开启
- prod 默认关闭，允许显式配置开启
- 不接入 `web` 菜单，只提供浏览器直达入口和右上角按钮快捷入口

## Implementation Path

- `server/internal/config`
  - 增加 docs 暴露策略配置
- `server/internal/app`
  - 在 core runtime 启动时预加载嵌入的 OpenAPI spec
  - 在 `registerCoreRoutes()` 挂载 `/openapi.json`、`/openapi.yaml`、`/docs`
- `web/src/layouts/components/Header.vue`
  - 在 GitHub 按钮旁增加文档按钮
  - GitHub 链接改为 `https://github.com/GeWuYou/Graft`
- `web/src/modules/auth/pages/components/Header.vue`
  - 同步增加登录页文档按钮

## Exposed Routes

- `GET /openapi.json`
  - 返回基于 `openapi/openapi.yaml` 解析后的 JSON 文档
- `GET /openapi.yaml`
  - 返回仓库根 `openapi/openapi.yaml` 的嵌入 YAML 内容
- `GET /docs`
  - 返回最小 HTML 页面
  - 页面通过 Scalar CDN 渲染 `/openapi.json`

## Exposure Strategy

- `local` / `development` / `dev` / `test`
  - 默认开启
- `prod` / `production`
  - 默认关闭
- 显式覆盖
  - `GRAFT_DOCS_ENABLED=true|false`

## UI Choice

- Docs UI: `Scalar` CDN
- Docs button icon: `book-open`
- GitHub button target: `https://github.com/GeWuYou/Graft`

## Final Route Choice

- 本切片同时提供：
  - `/openapi.json`
  - `/openapi.yaml`
- `/docs` 默认读取 `/openapi.json`
- 这样既满足约定路由名，也保留 YAML 直出调试入口

## Out Of Scope / Deferred

- 不接入 `web` 菜单
- 不新增 Swagger UI / Scalar / Redoc 包依赖
- 不修改业务 plugin handler / service / repository
- 不修改数据库 schema / migration
- 不调整 generated 代码

## Next Suggestions

1. 为 docs 页面增加最小 prod 提示文案或受限访问策略，而不是仅靠开关。
2. 将 docs 暴露策略补进示例环境配置说明。
3. 后续如需更强 Try-it-out 行为，再评估 Swagger UI 与更细粒度鉴权。
