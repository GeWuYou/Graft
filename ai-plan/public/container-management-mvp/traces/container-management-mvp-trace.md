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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-topic-persistence",
    "phase-1-openapi-contract-source",
    "phase-2-server-module-foundation",
    "phase-3-server-runtime-api-audit"
  ],
  "pending_batches": [
    "phase-4-web-container-management-ui",
    "phase-5-validation-governance-closeout"
  ],
  "current_batch": "phase-3-server-runtime-api-audit",
  "next_batch": "phase-4-web-container-management-ui",
  "closeout_status": "active"
}
```
