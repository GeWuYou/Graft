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
- `ai-plan/design/domains/container/容器管理设计.md`
- `ai-plan/design/architecture/项目设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/platform/系统配置模型与渲染设计.md`
- `ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md`
- `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`

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
- Phase 4 已完成：新增 `web/src/modules/container` 前端模块，接入 `运维管理 -> 容器管理` bootstrap route、OpenAPI
  schema 类型消费、容器列表筛选、详情 Drawer、日志 Drawer 和 start/stop/restart 高危确认。
- Phase 5 已完成：完成态 validation/governance closeout 已通过，topic 已从 active index 移入
  `ai-plan/public/archive/container-management-mvp/`。
- 当前分支：`feat/container-management-mvp`。
- 下一批次：无；主题 archive-ready / archived。

## Task Checklist

- [x] Phase 0：设计和 topic 持久化
- [x] Phase 1：OpenAPI 与 contract source
- [x] Phase 2：后端模块骨架、菜单、权限、i18n、系统配置定义
- [x] Phase 3：后端 `DockerRuntime` adapter、API、权限、错误映射、审计
- [x] Phase 4：前端容器列表、详情 Drawer、日志 Drawer/Dialog、高危确认
- [x] Phase 5：测试、i18n、治理收尾、归档准备

## Risks

- 本地容器运行时 socket 权限依赖部署用户和宿主机配置。
- Phase 3 引入官方 Docker Go SDK `github.com/docker/docker v28.5.2+incompatible`；其上游许可与仓库当前
  `AGPL-3.0-only` 授权兼容。依赖只服务 `DockerRuntime` adapter，不改变产品命名或 core runtime。
- `ops.container.actions.dangerous_enabled` 默认关闭，写操作必须被显式启用。
- 当前 core 配置快照尚未提供读取任意 `ops.container.*` 系统配置值的稳定 resolver；Phase 3 不扩展 resolver/UI surface，
  因此后端 runtime 继续使用模块默认值，API 默认返回 `runtimeDisabled`。后续若要启用运行时，应先走系统配置 resolver
  authority，而不是在容器模块私自读取环境变量。
- container id/name path 参数必须严格 encode/unescape/validate。
- 日志读取必须限制 `tail <= 2000` 并设置超时。
- OpenAPI source、generated Go、generated TypeScript 和前端消费层容易 drift，必须按 authority-first 顺序推进。
- 不得泄露敏感 env、secret、token 或 Docker inspect 原始敏感字段。

## Last Validation

- Phase 5 修改范围只包含 recovery/archive governance 文档和 active-topic index；未修改容器 feature code。
- 完成态验证：
  - `node scripts/openapi-bundle.mjs`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
  - `cd server && go test ./internal/contract/openapi/... ./modules/container/...`
  - `cd server && go run ./cmd/graft validate backend`
  - `git diff --check`
  - `rg "ops\\.docker|Docker 管理" ai-plan openapi server web --glob '!web/node_modules/**' --glob '!server/.cache/**'`
  - `rg "服务器管理" ai-plan openapi server web --glob '!web/node_modules/**' --glob '!server/.cache/**'`
- 结果：
  - OpenAPI source/generated drift clean；`openapi-bundle` 未产生 tracked diff，web generated schema check 通过。
  - Backend 完成态验证通过；`graft validate backend` 包含 migration version gate、OpenAPI generated freshness、
    backend DTO boundary check、Go test/build 路径。命令输出含既有 OpenAPI 3.1 / oapi-codegen warning，但未失败。
  - Web 完成态验证通过；首次 `bun run check` 在既有 monitor dependency-page Vitest 用例上出现一次未复现失败，
    focused monitor test 和 container frontend tests 随后通过，完整 `bun run check` rerun 通过 107 test files / 623 tests
    并完成 release build。
  - 命名约束通过：产品、权限、菜单、OpenAPI route 常量保持 `container` / `ops.container` 语义；`ops.docker` 仅出现于
    禁止项说明和测试断言，`docker` 仅作为 `DockerRuntime` adapter、官方 SDK dependency 和
    `ops.container.docker.endpoint` 配置键语义。
  - 菜单 IA 保持 `运维管理 -> 容器管理`；`服务器管理` 命中只来自容器设计/恢复文档中的禁止挂载说明、既有 monitor 模块
    和历史 archive evidence。
  - 未发现容器 API/UI/docs/code 暴露敏感 env、secret、token、authorization header 或 raw Docker inspect payload。
  - 未实现 MVP 禁止范围：exec、终端、文件编辑、删除、prune、镜像构建/拉取/推送、容器创建、远程 Docker Host、
    Kubernetes 或单独操作日志表。
  - Phase 4 范围扩展已确认是 authority/i18n repair：`server/modules/container/config.go` 本地 helper 字段名从
    `title` / `description` 调整为 `fallbackTitle` / `fallbackDescription`，并补齐 RBAC/system-config locale keys；
    该修复未改变配置 key、TitleKey、DescriptionKey 或 runtime API。
