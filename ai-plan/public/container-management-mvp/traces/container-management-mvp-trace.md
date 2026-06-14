# Container Management MVP Trace

## 2026-06-14

- 启动任务并确认 task class 为 `cross-boundary`。
- 当前分支从 `feat/frontend-vibe-toolchain-ui-fixes` 重命名为 `feat/container-management-mvp`。
- 建立容器管理设计 authority：`ai-plan/design/容器管理设计.md`。
- 建立 public recovery topic：`ai-plan/public/container-management-mvp/README.md`。
- 明确菜单 IA 为 `运维管理 / 容器管理`，不复用既有 `服务器管理`。
- 明确业务命名使用 `container` / `ops.container`，特定运行时只作为第一版 adapter 名称 `DockerRuntime` 或配置键
  `ops.container.docker.endpoint`。
- 明确 MVP 不做 exec、终端、文件编辑、删除、prune、镜像构建/拉取/推送、容器创建、远程 Docker Host、
  Kubernetes、长周期资源采集或单独操作日志表。
- Phase 0 仅持久化设计与恢复入口，不修改 OpenAPI source、后端模块或前端页面。
- Phase 1 新增 OpenAPI `container` tag（展示语义“容器管理”）和六个固定 API path：
  `GET /api/ops/containers`、`GET /api/ops/containers/{id}`、`GET /api/ops/containers/{id}/logs`、
  `POST /api/ops/containers/{id}/start`、`POST /api/ops/containers/{id}/stop`、
  `POST /api/ops/containers/{id}/restart`。
- Phase 1 新增 `ContainerSummary`、`ContainerDetail`、`ContainerPort`、`ContainerMount`、`ContainerNetwork`、
  `ContainerLogResponse`、`ContainerActionResponse`、`ContainerRuntimeInfo` 等 OpenAPI schema，并声明详情响应不暴露敏感环境变量。
- Phase 1 同步 `openapi/dist/openapi.bundle.json`、`server/internal/contract/openapi/generated/types.gen.go`、
  `server/internal/contract/openapi/container/zz_generated.container.go` 和
  `web/src/contracts/openapi/generated/schema.ts`。
- Phase 1 未实现 runtime handlers、DockerRuntime adapter 业务逻辑或前端页面。
- Phase 1 验证通过：`cd server && go test ./internal/contract/openapi/...`、`node scripts/openapi-bundle.mjs`、
  `cd web && bun run openapi:types`、`cd web && bun run openapi:types:check`、`git diff --check`。
- Phase 2 新增 `server/modules/container` 后端模块骨架，并通过 `NewModuleSpec()` 接入
  `server/internal/moduleregistry/generated.go`。
- Phase 2 注册菜单 IA 为 `运维管理 -> 容器管理`，菜单路径 `/ops/containers`，未挂入“服务器管理”。
- Phase 2 注册 MVP 权限：`ops.container.view`、`ops.container.detail`、`ops.container.logs`、
  `ops.container.start`、`ops.container.stop`、`ops.container.restart`；未新增 `ops.docker`。
- Phase 2 注册容器菜单、错误 key/message、route/config contract 常量和六个系统配置定义：
  `ops.container.enabled`、`ops.container.runtime`、`ops.container.docker.endpoint`、
  `ops.container.logs.default_tail`、`ops.container.logs.max_tail`、`ops.container.dangerous_actions_enabled`。
- Phase 2 将 `ops.container.docker.endpoint` 声明为 `RestartRequired`；系统配置 schema 作为系统配置 UI 渲染 authority。
- Phase 2 未实现 `DockerRuntime` adapter 业务逻辑、runtime API handlers、审计生产 service 行为或前端页面。
- Phase 2 验证通过：`cd server && go generate ./internal/moduleregistry`、
  `cd server && go test ./modules/container/...`、`cd server && go test ./internal/moduleregistry/... ./modules/container/...`、
  `cd server && go run ./cmd/graft validate backend --stage lint`、`git diff --check`。
- Phase 3 新增 `server/modules/container` runtime/service/route API，注册六个 `/ops/containers` 后端路由并通过
  `httpx.RequirePermission` 使用 Phase 2 已注册的 MVP 权限。
- Phase 3 新增 `DockerRuntime` adapter，使用官方 Docker Go SDK `github.com/docker/docker v28.5.2+incompatible`
  访问本地 `unix:///var/run/docker.sock`；该依赖为 Apache-2.0 license，兼容仓库授权。
- Phase 3 对 container id/name path 参数执行 `PathUnescape`，并拒绝空值、斜杠和控制字符。
- Phase 3 日志读取支持 `tail`、`since`、`timestamps`、`stdout`、`stderr`，默认 tail 为 200，最大 2000。
- Phase 3 详情和日志响应不暴露环境变量、secret、token、authorization header 或 raw Docker inspect payload。
- Phase 3 在 service 层为 start/stop/restart 成功和失败发布 `moduleapi.AuditEvent`，复用 audit event path，
  未新增单独操作日志表。
- Phase 3 使用 `ops.container.dangerous_actions_enabled` 语义守住 start/stop/restart；当前后端配置读取仍使用模块默认值，
  因 core 配置快照尚未提供任意系统配置 resolver，API 默认保持 disabled，未在模块内私自读取环境变量或扩展 resolver surface。
- Phase 3 验证通过：`cd server && go test ./modules/container/...`、
  `cd server && go test ./internal/contract/openapi/... ./modules/container/...`、
  `cd server && go run ./cmd/graft validate backend --stage lint`、`git diff --check`。
- Phase 4 新增 `web/src/modules/container` 前端模块，采用 `list-form-detail` 页面类型，注册 bootstrap route 到
  `/ops/containers`，页面语义保持 `运维管理 -> 容器管理`。
- Phase 4 API 层从 `@/contracts/openapi/generated/schema` 提取 OpenAPI operation 类型，未手写第二套 DTO 或 runtime client。
- Phase 4 页面实现容器列表、status/keyword filters、refresh、状态 Tag、详情 Drawer、日志 Drawer（tail/since/timestamps/stdout/stderr、
  refresh、copy）和 start/stop/restart Popconfirm 高危确认。
- Phase 4 表格列覆盖 status、name、image、ports、created/started time、restart policy 和 detail/logs/start/stop/restart 操作；
  未展示 CPU/memory 空列。
- Phase 4 接入 `v-permission` 语义使用 Phase 2 `ops.container.*` 权限；未新增 `ops.docker` 或前端私有授权语义。
- Phase 4 补齐 RBAC permission catalog 与 system-config ops/container locale keys；前端可见 copy 位于模块/既有 locale 边界。
- Phase 4 发现 `web` i18n 完成态校验会扫描 `server/modules/container/config.go`，其本地 helper 字段名
  `title` / `description` 被识别为 fallback-only copy。已做窄范围 backend authority repair：改为 `fallbackTitle` /
  `fallbackDescription`，不改变注册的 `TitleKey`、`DescriptionKey`、配置 key 或 runtime API。
- Phase 4 TDesign MCP preflight 使用 `vue-next` 查询组件列表、Table、Tag、Drawer、Dialog、Popconfirm、InputNumber、Select、Button、
  Space、Tooltip、Form、Input、Empty、Descriptions、Alert、Loading、Checkbox 文档，并查询 Table、Drawer、Dialog、Form、Select、
  Popconfirm DOM 结构。
- Phase 4 验证通过：`cd web && bun run check`、`cd server && go test ./modules/container/...`、
  `cd server && go run ./cmd/graft validate backend --stage lint`、`git diff --check`。
- Phase 4 未启动浏览器 dev server 做浏览器 QA；完成态验证以 `bun run check` 的 typecheck/governance/lint/style/test/build 为准。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-topic-persistence",
    "phase-1-openapi-contract-source",
    "phase-2-server-module-foundation",
    "phase-3-server-runtime-api-audit",
    "phase-4-web-container-management-ui"
  ],
  "pending_batches": [
    "phase-5-validation-governance-closeout"
  ],
  "current_batch": "phase-4-web-container-management-ui",
  "next_batch": "phase-5-validation-governance-closeout",
  "closeout_status": "active"
}
```
