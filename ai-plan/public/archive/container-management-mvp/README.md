# Container Management MVP

## 当前状态摘要

- 当前主题目标是在 `Graft` 增加“运维管理 -> 容器管理”能力。
- 状态：`archived`。
- 任务分类为 `cross-boundary`，涉及 OpenAPI、server module、permission/menu/audit/system-config、web module 和 i18n。
- Canonical design：`ai-plan/design/domains/container/容器管理设计.md`。
- 已完成 Phase 0 设计与恢复入口、Phase 1 OpenAPI contract source、Phase 2 后端模块骨架、Phase 3 后端运行时
  API / `DockerRuntime` adapter、Phase 4 前端容器管理页面、Phase 5 完成态验证与治理收尾。
- 本主题已从 active topic index 移入 `ai-plan/public/archive/container-management-mvp/`。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：`ai-plan/design/domains/container/容器管理设计.md` + OpenAPI source + `server/modules/container` module contract/descriptor + `web/src/modules/container` bootstrap routes + permission/menu/audit/system-config/i18n governance docs

## Historical Owned Scope

实现期允许修改：

- `ai-plan/design/domains/container/容器管理设计.md`
- `ai-plan/public/container-management-mvp/**`
- `ai-plan/public/archive/container-management-mvp/**`
- `ai-plan/public/README.md`
- `openapi/**`
- `server/modules/container/**`
- `server/internal/moduleregistry/generated.go`
- 必要的 backend generated OpenAPI contract 文件
- `web/src/modules/container/**`
- 必要的 `web` OpenAPI generated schema、module aggregation、route/menu/i18n 接入文件

禁止误触：

- 不得把产品能力命名为特定容器运行时管理。
- 不得使用 `ops.docker` 作为权限、菜单、OpenAPI tag 或前端业务前缀。
- 下列限制是本 topic 早期设计基线，不代表最终 archived 完成态；最终 MVP authority 以
  `ai-plan/design/domains/container/容器管理设计.md` 为准。该基线当时排除了 exec、服务器终端、文件编辑、删除、prune、
  镜像构建/拉取/推送、容器创建、多节点、远程 Docker Host 和 Kubernetes。
- 不得新增单独操作日志表；操作记录复用 audit log。
- 不得把容器管理挂入既有“服务器管理”菜单。

## Phase Plan

- Phase 0：设计和 public topic 持久化。
- Phase 1：OpenAPI 与 contract source。
- Phase 2：后端模块骨架、菜单、权限、i18n、系统配置定义。
- Phase 3：后端 `DockerRuntime` adapter、API、权限、错误映射、审计。
- Phase 4：前端容器列表、详情 Drawer、日志 Drawer/Dialog、高危确认。
- Phase 5：测试、i18n、治理收尾、归档准备。

## Current Recovery Point

- 分支为 `feat/container-management-mvp`。
- 设计 authority 已落到 `ai-plan/design/domains/container/容器管理设计.md`。
- archived topic 已落到 `ai-plan/public/archive/container-management-mvp/`。
- Phase 0 已完成：设计文档与 public topic 持久化。
- Phase 1 已完成：OpenAPI 与 generated contract artifacts。
- Phase 2 已完成：`server/modules/container` 后端模块骨架、菜单、权限、i18n、系统配置定义和 compile-time registry 接入。
- Phase 3 已完成：后端 `DockerRuntime` adapter、API、权限、错误映射、日志 guardrail 和 start/stop/restart 审计。
- Phase 4 已完成：前端容器列表、详情 Drawer、日志 Drawer 和 start/stop/restart 高危确认。
- Phase 5 已完成：OpenAPI/generated drift、backend、web、命名、安全边界与 topic archive closeout 均通过最终核对。
- 当前下一批次：无；主题处于 archive-ready / archived 状态。

## Validation Targets

```bash
git diff --check
cd server && go test ./internal/contract/openapi/...
node scripts/openapi-bundle.mjs
cd web && bun run openapi:types:check
cd web && bun run check
cd server && go run ./cmd/graft validate backend
```

最终 Phase 5 已执行完成态验证。`cd web && bun run check` 第一次运行在一个既有 monitor dependency-page Vitest
用例上出现一次未复现失败；随后 focused monitor test、container frontend tests 和完整 `bun run check` 均通过。
