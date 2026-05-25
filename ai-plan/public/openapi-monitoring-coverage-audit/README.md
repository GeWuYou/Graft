# OpenAPI Monitoring Coverage Audit

## Topic

- Topic: `openapi-monitoring-coverage-audit`
- Task class: `cross-boundary`
- Branch: `feat/wt-oapi-codegen-types-only-spike`
- Recovery source: current repository state + `ai-plan/public/openapi-docs-mvp` + `ai-plan/public/openapi-docs-bundled-spec-fix`

## Startup Receipt

- Governance source: root `AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: `subtopic`
- Repository root: current worktree
- Owned scope:
  - `ai-plan/public/openapi-monitoring-coverage-audit/**`
  - `openapi/**` read-only unless final conclusion is pure contract backfill
  - `server/**` read-only
  - `web/**` read-only
  - `scripts/**` read-only
- Forbidden scope:
  - `server/internal/ent/**`
  - database migrations
  - business service/repository behavior changes
  - frontend page refactor
  - broad generated-file churn
  - unrelated formatting

## Audit Result

结论先行：

- 当前项目并非缺少监控后端接口。
- 已存在真实后端 route：`GET /api/monitor/server-status`。
- `web` 监控模块三页已经消费该接口。
- 当前缺口是 OpenAPI 未登记 `monitor` tag 和 `/api/monitor/server-status` path。
- 本轮未补契约，只完成审计与最小补齐方案定义。

## Confirmed Facts

### Backend

- `server/internal/app/runtime.go:166` 将插件路由挂到 `/api` 组。
- `server/plugins/monitor/plugin.go:398-404` 注册 `ctx.Router.Group("/monitor").GET("/server-status", ...)`。
- 因此真实 HTTP 路径为 `GET /api/monitor/server-status`。

### Frontend

- `web/src/modules/monitor/contract/paths.ts:8-10` 定义 `MONITOR_API_PATH.SERVER_STATUS = "/api/monitor/server-status"`。
- `web/src/modules/monitor/api/server-status.ts:7-13` 使用该路径发起 `request.get`。
- `web/src/modules/monitor/pages/overview/index.vue` 直接调用 `getServerStatus(...)`。
- `web/src/modules/monitor/shared/server-status-snapshot.ts:32-52` 在共享 snapshot 层调用 `getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES)`，被 `runtime` 和 `dependencies` 页面复用。

### OpenAPI

- `openapi/openapi.yaml:13-18` 当前仅有 `health`、`auth`、`users`、`rbac` tags。
- `openapi/openapi.yaml:18-44` 当前没有 `/api/monitor/server-status` path。
- `openapi/paths/**` 也没有 `monitor` 相关 path 文件。

## Minimal Backfill Plan

仅在下一轮满足“已有 route、已有使用、请求/响应结构明确、不改业务行为”时执行：

1. 在 `openapi/openapi.yaml` 增加 `monitor` tag。
2. 新增 `openapi/paths/monitor.server-status.yaml`。
3. 为 `trend_range` query 参数补契约，值域以当前稳定值为准：`10m`、`30m`、`1h`。
4. 为响应体补最小 schema：
   - `status`
   - `observed_at`
   - `server`
   - `runtime`
   - `dependencies`
   - `summary`
   - `trend`
   - `plugins`
5. 保持只补 OpenAPI 契约，不改 `server/plugins/monitor/**`、`web/src/modules/monitor/**` 的真实行为。

## Out Of Scope

- 不新增监控业务能力
- 不重构监控页面
- 不迁移 DTO
- 不改 Scalar 样式
- 不移动 docs 路由
- 不修改无关模块
