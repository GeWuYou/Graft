# OpenAPI Monitoring Coverage Audit Trace

## Backend Route Inventory

| Method | Path | 所属模块 | Handler 文件 | 是否已在 OpenAPI | 备注 |
| --- | --- | --- | --- | --- | --- |
| GET | `/healthz` | core runtime | `server/internal/app/runtime.go` | Present | 非监控业务接口，但属于状态探针 |
| GET | `/openapi.json` | core runtime | `server/internal/app/runtime.go` | N/A | docs 暴露面，不纳入业务 OpenAPI 覆盖审计 |
| GET | `/openapi.yaml` | core runtime | `server/internal/app/runtime.go` | N/A | docs 暴露面，不纳入业务 OpenAPI 覆盖审计 |
| GET | `/docs` | core runtime | `server/internal/app/runtime.go` | N/A | docs UI，不纳入业务 OpenAPI 覆盖审计 |
| GET | `/api/monitor/server-status` | `monitor` plugin | `server/plugins/monitor/plugin.go` | Missing | 真实监控接口，前端已接入 |

## Backend Evidence

- `server/internal/app/runtime.go:166`
  - `plugin.Context.Router = r.server.Engine().Group("/api")`
- `server/internal/app/runtime.go:230-255`
  - core runtime 直接注册 `/healthz`、`/openapi.json`、`/openapi.yaml`、`/docs`
- `server/plugins/monitor/contract/route.go:9-34`
  - `MonitorGroup = "/monitor"`
  - `ServerStatusRoute = "/server-status"`
- `server/plugins/monitor/plugin.go:398-404`
  - `ctx.Router.Group("/monitor").GET("/server-status", ...)`
- `server/plugins/monitor/plugin.go:409-420`
  - handler 读取 `trend_range`，返回 `buildServerStatusResponse(...)`

## Frontend Monitoring Call Inventory

| 页面/模块 | API 路径 | 方法 | 文件 | 是否后端存在 | 是否 OpenAPI 存在 | 备注 |
| --- | --- | --- | --- | --- | --- | --- |
| `monitor/overview` | `/api/monitor/server-status` | GET | `web/src/modules/monitor/pages/overview/index.vue` | Yes | No | 直接调用 `getServerStatus(requestedTrendRange)` |
| `monitor/runtime` | `/api/monitor/server-status` | GET | `web/src/modules/monitor/shared/server-status-snapshot.ts` | Yes | No | 页面复用共享 snapshot，不直接写 request |
| `monitor/dependencies` | `/api/monitor/server-status` | GET | `web/src/modules/monitor/shared/server-status-snapshot.ts` | Yes | No | 页面复用共享 snapshot，不直接写 request |
| `monitor/dependencies` future entry 卡片 | None | None | `web/src/modules/monitor/pages/dependencies/index.vue` | N/A | N/A | 静态占位，不是未接入接口 |
| `monitor/runtime` 某些展示字段 | None | None | `web/src/modules/monitor/pages/runtime/index.vue` | N/A | N/A | `notReported` 展示，不是缺少 route |
| `access-control/overview` | `/api/users`、`/api/roles`、`/api/permissions` 等间接调用 | mixed | `web/src/modules/access-control/pages/overview/index.vue` | Yes | Mostly present | 不是 monitor 模块，不计入本轮缺口 |

## Frontend Evidence

- `web/src/modules/monitor/bootstrap-routes.ts:5-20`
  - 已注册 `overview`、`runtime`、`dependencies` 三个监控页面
- `web/src/modules/monitor/contract/paths.ts:1-10`
  - 路由常量与 API 常量都已固定
- `web/src/modules/monitor/api/server-status.ts:7-13`
  - `request.get({ url: "/api/monitor/server-status", params: { trend_range } })`
- `web/src/modules/monitor/pages/overview/index.vue:379` 与 `:945`
  - overview 页直接调用 `getServerStatus(...)`
- `web/src/modules/monitor/shared/server-status-snapshot.ts:32-52`
  - runtime/dependencies 共用 snapshot 刷新逻辑，请求固定走 `getServerStatus(MONITOR_TREND_RANGE.TEN_MINUTES)`
- `web/src/modules/monitor/pages/dependencies/index.vue`
  - `futureEntry` 区块是静态 UI，不对应未建后端接口

## Current OpenAPI Coverage

### Tags

当前 `openapi/openapi.yaml` 只有以下 tag：

- `health`
- `auth`
- `users`
- `rbac`

### Path Inventory

| 路径 | 来源 | OpenAPI 状态 | 建议 |
| --- | --- | --- | --- |
| `/healthz` | server route | Present | 无需处理 |
| `/api/monitor/server-status` | server route + web call | Missing | 补契约 |
| `/api/auth/login` | server route | Present | 无需处理 |
| `/api/auth/refresh` | server route | Present | 无需处理 |
| `/api/auth/logout` | server route | Present | 无需处理 |
| `/api/auth/bootstrap` | server route | Present | 无需处理 |
| `/api/users` | server route + web call | Present | 无需处理 |
| `/api/roles` | server route + web call | Present | 无需处理 |
| `/api/permissions` | server route + web call | Present | 无需处理 |

## Coverage Judgment

1. 是否确实少了监控接口
   - 少的是 OpenAPI 覆盖，不是后端 route。
2. 少的是后端 route 还是 OpenAPI path
   - 缺 `openapi/openapi.yaml` path 注册与 `monitor` tag。
3. 是否有前端调用了未登记接口
   - 有，`GET /api/monitor/server-status`。
4. 是否有 OpenAPI 写了但后端没有的孤儿接口
   - 本轮监控范围内未发现。

## Contract Clarity Assessment

当前监控接口属于“可最小补齐”的契约缺口，而不是“信息不足候选”：

- query 参数稳定：`trend_range`
- 值域稳定：`10m`、`30m`、`1h`
- 响应结构可从以下来源稳定推导：
  - `server/plugins/monitor/plugin.go`
  - `server/plugins/monitor/plugin_test.go`
  - `web/src/modules/monitor/types/server-status.ts`

因此若下一轮进入补契约，可不改业务行为，只补 OpenAPI path 与 schema。
