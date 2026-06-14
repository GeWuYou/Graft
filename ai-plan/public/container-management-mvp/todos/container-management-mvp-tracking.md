# Container Management MVP Tracking

## Topic

Container Management MVP

## Scope

实现“运维管理 -> 容器管理”模块，第一版通过本机容器运行时 socket 支持容器列表、详情、日志读取和
`start` / `stop` / `restart`，并接入权限、审计、系统配置与前端管理页面。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/模块与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/系统配置模型与渲染设计.md`
- `ai-plan/design/TDesign-MCP-辅助开发规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Authority Discovery

- `server/modules/container/**` 是后端业务模块 authority。
- `openapi/**` 是 wire contract authority。
- `web/src/modules/container/**` 是前端业务模块 authority。
- 权限、菜单、错误 key、route fragment、message key 属于模块 contract。
- `DockerRuntime` 只是第一版 runtime adapter，不是业务命名前缀。
- 第一版不建数据库表；容器状态来自本地容器运行时 API，操作记录复用 audit log。

## Current Recovery Point

- Phase 0 已完成：设计文档与 public topic 持久化。
- Phase 1 已完成：OpenAPI source 新增容器管理 tag、六个 `/api/ops/containers` 路径、容器响应 schema，并同步
  backend / web OpenAPI derived artifacts。
- Phase 2 已完成：新增 `server/modules/container` 后端模块骨架，注册 `运维管理 -> 容器管理` 菜单、六个 MVP
  `ops.container.*` 权限、容器错误/message key、六个系统配置定义，并接入 compile-time module registry。
- Phase 3 已完成：新增后端容器 runtime/service/route API、`DockerRuntime` adapter、权限保护、错误映射、日志 guardrail
  和 start/stop/restart 审计事件。
- 当前分支：`feat/container-management-mvp`。
- 下一批次：Phase 4 前端容器列表、详情 Drawer、日志 Drawer/Dialog、高危确认。
- 下一批次不得修改后端 runtime/API，除非发现 OpenAPI 或后端 authority 阻断前端消费。

## Task Checklist

- [x] Phase 0：设计和 topic 持久化
- [x] Phase 1：OpenAPI 与 contract source
- [x] Phase 2：后端模块骨架、菜单、权限、i18n、系统配置定义
- [x] Phase 3：后端 `DockerRuntime` adapter、API、权限、错误映射、审计
- [ ] Phase 4：前端容器列表、详情 Drawer、日志 Drawer/Dialog、高危确认
- [ ] Phase 5：测试、i18n、治理收尾、归档准备

## Risks

- 本地容器运行时 socket 权限依赖部署用户和宿主机配置。
- Phase 3 引入官方 Docker Go SDK `github.com/docker/docker v28.5.2+incompatible`；license 为 Apache-2.0，兼容仓库
  Apache-2.0 授权。依赖只服务 `DockerRuntime` adapter，不改变产品命名或 core runtime。
- `dangerous_actions_enabled` 默认关闭，写操作必须被显式启用。
- 当前 core 配置快照尚未提供读取任意 `ops.container.*` 系统配置值的稳定 resolver；Phase 3 不扩展 resolver/UI surface，
  因此后端 runtime 继续使用模块默认值，API 默认返回 `runtimeDisabled`。后续若要启用运行时，应先走系统配置 resolver
  authority，而不是在容器模块私自读取环境变量。
- container id/name path 参数必须严格 encode/unescape/validate。
- 日志读取必须限制 `tail <= 2000` 并设置超时。
- OpenAPI source、generated Go、generated TypeScript 和前端消费层容易 drift，必须按 authority-first 顺序推进。
- 不得泄露敏感 env、secret、token 或 Docker inspect 原始敏感字段。

## Last Validation

- Phase 3 修改 `server/modules/container/**`、`server/go.mod`、`server/go.sum` 和 public recovery 文档。
- 直接验证：
  - `cd server && go test ./modules/container/...`
  - `cd server && go test ./internal/contract/openapi/... ./modules/container/...`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `git diff --check`
- Phase 3 未运行 web 完成态校验；本批次未修改 `web/**` 或 OpenAPI source。
- 命名约束：产品、权限、菜单、OpenAPI route 常量保持 `container` / `ops.container` 语义；`docker` 仅出现在
  `DockerRuntime` adapter、官方 SDK dependency 和 `ops.container.docker.endpoint` 配置键语义。
