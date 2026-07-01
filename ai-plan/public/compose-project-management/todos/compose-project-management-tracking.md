# Compose Project Management Tracking

## Topic

Compose Project Management

## Scope

在 `Graft` 新增 Docker Compose Project 管理能力，保持 `Project` 作为管理与聚合层，保持 `Container` 作为运行时 authority，并按 Phase 1-3 分阶段完成导入、项目注册、生命周期、配置只读、活动聚合与后续扩展。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/domains/compose/Compose项目管理设计.md`
- `ai-plan/design/domains/container/容器管理设计.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/architecture/前端架构设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md`
- `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Current Recovery Point

- Phase 0 已完成：
  - Compose Project authority 文档已落地
  - active topic / tracking / trace / startup prompt 已落地
- Phase 1 Batch 1 已完成：
  - `openapi/**` 已建立 `/api/ops/projects/**` route space、import / lifecycle / readonly configuration contract source。
  - `server/modules/project/**` 已建立 module-owned typed contract、数据模型与 SQL migration baseline。
  - `server/internal/moduleapi/container_project.go` 已建立后续项目服务聚合所需的最小稳定 container shared boundary。
  - Compose 设计 authority 已同步 Batch 1 的 canonical owner 落点。
- Phase 1 Batch 2 已完成：
  - `server/modules/project/**` 已建立 module skeleton、repository、Compose import validate/import/register/refresh 服务与 route wiring。
  - `server/internal/moduleregistry/generated.go` 已同步 compile-time registry 派生产物。
  - `server/internal/moduleregistry/registry_test.go` 已完成最小上游 authority repair，使 project migration baseline 纳入 owner-aligned registry 预期。
- Phase 1 Batch 3 已完成：
  - `server/modules/project/**` 已建立 `up/down/restart/unregister/destroy` 生命周期路径、ownership guard、services/runtime summary 映射和 soft-delete repository 能力。
  - `server/modules/container/**` 已提供最小稳定 `ContainerProjectRuntimeReader` 实现，供 project 聚合 runtime member/counts 使用。
  - `container` 仍保持 runtime authority，未引入 project-level logs/events backend aggregation。
- Phase 1 Batch 4 已完成：
  - `web/src/modules/project/**` 已建立 project module registration、typed API consumer、locale owner，以及 list/detail 页面。
  - `Overview`、`Services`、`Configuration`、`Activity` 四个页签已按 design authority 落地，且未把 Overview 做成 runtime dashboard。
  - `Configuration` 继续保持只读三段式消费；`Activity` 继续只做前端 fan-out，复用现有 container logs/events。
- Phase 1 Batch 5 已完成：
  - Phase 1 validation chain 已重新跑通，包含 OpenAPI bundle、project migration SQL 校验、backend entrypoint 与 web entrypoint。
  - Compose 设计 authority 已同步 batch 4 前端 owner 落点。
  - Phase 1 acceptance conditions 已满足，主题继续推进到 Phase 2，而不是停在 Phase 1 closeout。
- Phase 2 Batch 1 已完成：
  - `project` 模块已拥有 managed root 系统配置键、managed create 权限与路由合同，以及 OpenAPI create/create-validate/managed-root 合同源。
  - 本批只修上游 authority owner，没有越界实现实际文件写入、editor、diff、validate UI 或 deploy。
- Phase 2 Batch 2 已完成：
  - `project` 模块已拥有 managed create 的服务端 file-write path：在 managed root 下创建 working directory、写 compose/env 文件、解析配置、持久化 registry 与 snapshot bootstrap。
  - create flow 在 registry 失败时会清理本轮新建目录和文件，避免留下无主目录。
  - 本批同步修正 create request/response 的 canonical OpenAPI authority，使其与真实同步创建语义一致。
- Phase 2 Batch 3 已完成：
  - `web/src/modules/project/**` 已建立 managed create route、create 表单流和 Compose/Env editor surface。
  - 本批继续使用 Phase 2 的 create authority，没有越界进入 diff/deploy flow、remote host 或 backend runtime-state persistence。
  - TDesign MCP preflight 已执行并用于 create form、editor surface、tabs 与 summary card 设计落地。
- Phase 2 Batch 4 已完成：
  - `openapi/**`、`server/modules/project/**`、`web/src/modules/project/**` 已共同落地 managed compose project 的 `diff / validate / deploy` 流程。
  - `Project` 仍只拥有配置草稿、差异、校验和部署编排，没有引入项目级 runtime 持久化，也没有越界到 container 私有实现或后端 project logs/events 聚合。
  - 前端仍在 `project detail` 的 `list-form-detail` 页型中承接 Configuration tab 下的编辑、diff、validate、deploy 流程。
- Phase 2 Batch 5 已完成：
  - Phase 2 managed create/edit/diff/validate/deploy slice 的验证链已重新跑通，未发现 contract、generated、validation 或 governance drift 需要额外实现修补。
  - Phase 2 archive-readiness check 已通过：`Project` 继续只拥有 project registry、draft editor、静态 diff/validate 与 deploy orchestration，没有引入 project-level runtime persistence 或 backend project logs/events aggregation。
  - 原有过宽的 `phase-3-discovery-git-template-and-remote-host` 已重切为安全 bounded batches，并把下一步前移到 `phase-3-batch-1-git-template-source-contract-and-boundary`。
- 当前 authority 决议：
  - `Project` 不得持久化容器运行时信息。
  - `Project` 不得新增自己的 container detail。
  - Phase 1 Activity 继续复用 container logs/events，并由前端 fan-out 聚合。
  - Phase 1 Configuration 只读。
  - `Canonical Project Name` 与 `Display Name` 必须分离。
  - `Unregister` 是安全默认；`Destroy` 是显式高危动作。

## Task Checklist

- [x] Phase 0：Compose Project 设计 authority
- [x] Phase 0：public topic recovery materials
- [x] Phase 0：`$graft-multi-agent-loop` startup prompt
- [x] phase-1-batch-1：project contract、route space、data model、migration plan
- [x] phase-1-batch-2：server project module skeleton、repository、import validate/import/register/refresh
- [x] phase-1-batch-3：lifecycle executor、ownership guard、container aggregation shared boundary
- [x] phase-1-batch-4：web project module list/detail/overview/services/configuration/activity
- [x] phase-1-batch-5：Phase 1 validation、drift guard、docs sync、Phase 1 archive-readiness check
- [x] phase-2-batch-1：managed root、create contract、system config / permission / route authority
- [x] phase-2-batch-2：server managed create、compose/env file write path、snapshot bootstrap
- [x] phase-2-batch-3：web managed create、compose/env editors
- [x] phase-2-batch-4：diff、validate、deploy flow
- [x] phase-2-batch-5：Phase 2 validation、drift guard、docs sync、Phase 2 archive-readiness check
- [x] phase-3-batch-1：git/template source contract、metadata boundary、route/permission owner
- [x] phase-3-batch-2：directory scan、candidate model、auto discovery bounded authority
- [ ] phase-3-batch-3：remote host boundary、project activity authority decision
- [ ] drift-repair：恢复 Phase 1 import 主入口、托管创建次入口、source selector 边界定位，以及 topic truth

## Phase 1 Acceptance Conditions

- 可以导入本机现有 Compose Project
- 可以保存 working directory、compose files、env files、canonical name、display name、snapshot 与 drift metadata
- 可以查看项目列表与详情
- Overview 保持 summary，不复制 runtime dashboard
- Services 以静态定义加容器计数方式工作，并可跳转到现有 Container Detail
- Configuration 只读，支持 file list、preview、download
- Activity 继续通过前端 fan-out 使用现有 container logs/events
- 支持 `refresh/up/down/restart/unregister/destroy`
- 销毁逻辑有 ownership proof guard

## Phase 2 Acceptance Conditions

- 支持在 managed root 下创建项目
- 支持 Compose / Env 编辑
- 支持 diff / validate / deploy

2026-07-01 archive-readiness check：

- 通过完整验证链：`git diff --check`、`node scripts/openapi-bundle.mjs`、`python3 scripts/validate_sql_migrations.py --paths server/modules/project/migrations/202606300002_project_registry_baseline.sql`、`cd server && go run ./cmd/graft validate backend`、`cd web && bun run check`
- `Project` 与 `Container` authority 边界保持稳定，没有引入 project-level runtime persistence、project-owned container detail 或 backend project logs/events aggregation
- Topic 未进入 `archive-ready`，因为安全重切后的 Phase 3 bounded work 仍明确存在

## Phase 3 Acceptance Conditions

- 支持 git/template/scan/discovery/remote-host 扩展路径
- 支持后端 project activity aggregation authority

当前 batch-1 已完成的前置条件：

- source catalog authority 已固定到 `openapi/** + server/modules/project/** + web/src/modules/project/**`
- source selector 与 `managed/git/template` route boundary 已固定
- git/template 仍保持 planned，不存在 runtime persistence、directory scan、remote host 或 backend activity aggregation 越界

当前 batch-2 已完成的前置条件：

- `openapi/**` 已固定 discovery candidate 只读 contract 与 `/api/ops/projects/discovery-candidates` route owner
- `server/modules/project/**` 已把 managed root 收口为 bounded local scan authority，只返回 candidate preview，不自动注册 project
- `web/src/modules/project/**` 已在 source selector 下提供 hidden discovery preview surface，不越界到 remote host 或 backend activity aggregation

当前 batch-3 的当前状态：

- `openapi/**` 已把 `remote-host` 固定为 planned source entry，并为 project list/detail 固定 `activity_authority` contract
- `server/modules/project/**` 已把 remote-host 收口为 source catalog / route / metadata owner，不引入 remote execution、credential persistence 或 backend activity aggregation
- `web/src/modules/project/**` 已提供 `/ops/projects/create/remote-host` planned boundary，并在 detail 页面显式提示当前 activity authority

## Topic Archive-Readiness Check

- Phase 1 acceptance conditions：已满足
- Phase 2 acceptance conditions：已满足
- Phase 3 acceptance conditions：未满足
  - git/template/scan/discovery/remote-host 扩展路径目前只完成了部分 authority boundary
  - 后端 project activity aggregation authority 仍停留在 `backend-planned`
- 当前 topic 不是 `archive-ready`：
  - 主入口 IA 已从 Phase 1/2 设计偏移到 Phase 3 boundary surface，必须先修复入口 truth
  - `compose-project-management` 的 recovery docs 与实际可用性需要重新对齐

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-authority-and-topic-persistence",
    "phase-1-batch-1-project-contract-and-data-model",
    "phase-1-batch-2-server-project-module-import-and-refresh",
    "phase-1-batch-3-server-lifecycle-and-container-aggregation-boundary",
    "phase-1-batch-4-web-project-list-detail-and-readonly-configuration",
    "phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync",
    "phase-2-batch-1-managed-root-and-create-contracts",
    "phase-2-batch-2-server-managed-create-and-file-write-path",
    "phase-2-batch-3-web-managed-create-and-editors",
    "phase-2-batch-4-diff-validate-and-deploy-flow",
    "phase-2-batch-5-phase-2-validation-drift-guard-and-governance-sync",
    "phase-3-batch-1-git-template-source-contract-and-boundary",
    "phase-3-batch-2-directory-scan-and-auto-discovery-candidates"
  ],
  "pending_batches": [
    "drift-repair-import-primary-entry-and-topic-truth",
    "phase-3-batch-3-remote-host-boundary-and-activity-authority"
  ],
  "current_batch": "drift-repair-import-primary-entry-and-topic-truth",
  "next_batch": "phase-3-batch-3-remote-host-boundary-and-activity-authority",
  "closeout_status": "drift-repair-in-progress"
}
```
