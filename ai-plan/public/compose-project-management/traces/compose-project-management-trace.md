# Compose Project Management Trace

## 2026-06-28 Phase 0 authority and recovery persistence

- 建立 Compose Project 设计 authority：`ai-plan/design/domains/compose/Compose项目管理设计.md`。
- 建立 active topic：`ai-plan/public/compose-project-management/README.md`。
- 建立 tracking：`ai-plan/public/compose-project-management/todos/compose-project-management-tracking.md`。
- 建立 trace：`ai-plan/public/compose-project-management/traces/compose-project-management-trace.md`。
- 建立 `$graft-multi-agent-loop` 启动提示：`ai-plan/public/compose-project-management/startup-prompt.md`。
- 将 active topic 注册进 `ai-plan/public/README.md`。

## 2026-06-28 Locked decisions

- `Project` 是 Compose Project 的管理与聚合层，不是新的 Runtime。
- `Container` 继续是 Runtime Authority。
- 当前仓库只有 Compose labels 识别能力，没有 Compose Project registry、解析、生命周期执行或项目级持久化。
- 推荐新增独立 `project` module，而不是让 `container` module 吞并项目注册。
- 推荐静态解析使用 `compose-go`，生命周期执行使用 `docker compose` CLI。
- 推荐 persistence 使用模块自有 `database/sql + migrations` 模式。
- 推荐为 `project` 新增最小的 container runtime read shared boundary，而不是直接 import container private service。
- Phase 1 Activity 继续复用现有 container logs/events，由前端 fan-out 聚合。
- Phase 1 Configuration 保持只读。

## 2026-06-28 Initial implementation direction

- Phase 1 优先级固定为：
  - Import Existing Project
  - Project Registry
  - Overview
  - Services
  - Configuration Read Only
  - Activity frontend aggregation
  - `up/down/restart`
  - Refresh
  - Unregister
  - Destroy guard
- Phase 2 再进入 managed create、editor、diff、validate、deploy。
- Phase 3 再进入 git/template/discovery/remote-host/backend aggregation。

## 2026-06-30 Phase 1 Batch 1 project contract and data model

- 落地 `openapi/**` authority owner：新增 `/api/ops/projects/**` route space，以及 import、detail、services、readonly configuration、refresh、up/down/restart、unregister、destroy 的 canonical contract source。
- 落地 `server/modules/project/**` authority owner：新增 module-owned route contract、message key、typed model，以及 `202606300002_project_registry_baseline.sql` migration baseline。
- 落地 `server/internal/moduleapi/container_project.go`：定义后续 `project -> container` 只读聚合所需的最小稳定 shared boundary，避免直接依赖 `server/modules/container/**` 私有实现。
- 同步 `ai-plan/design/domains/compose/Compose项目管理设计.md`，把 Batch 1 authority owner 和批次边界写回设计文档，避免 topic 设计与实现漂移。
- 本批验证通过：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `python3 scripts/validate_sql_migrations.py --paths server/modules/project/migrations/202606300002_project_registry_baseline.sql`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`8c23dd2e` `feat(project): define phase 1 project contract and data model`

## 2026-06-30 Phase 1 Batch 2 server project module import and refresh

- 落地 `server/modules/project/**` authority owner：建立 module skeleton、SQL repository、Compose loader、import validate/import/register/refresh 服务与 route wiring。
- 同步 `server/internal/moduleregistry/generated.go`，把 project module 纳入 compile-time registry 派生产物。
- 在 retry round 中修复 `server/internal/moduleregistry/registry_test.go` 的最小上游 authority drift，把 `modules/project/migrations` 纳入 owner-aligned migration baseline 预期，使 required backend validation 恢复通过。
- 本批验证通过：
  - `git diff --check`
  - `cd server && go test ./modules/project/...`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`608a5815` `feat(project): add project import and refresh module`

## 2026-06-30 Phase 1 Batch 3 server lifecycle and runtime aggregation boundary

- 落地 `server/modules/project/**` authority owner：建立 `up/down/restart/unregister/destroy` 生命周期路径、ownership guard、service/runtime summary 映射，以及 repository soft-delete 能力。
- 落地 `server/modules/container/**` authority owner：新增 `ContainerProjectRuntimeReader` 的最小稳定实现，只暴露 project 聚合所需的 runtime members/counts，保持 container 作为 runtime authority。
- 继续复用 `server/internal/moduleapi/container_project.go` 作为跨模块稳定边界，没有把 detail/logs/events/stats/shell 私有实现泄漏给 `project` module。
- 本批验证通过：
  - `git diff --check`
  - `cd server && go test ./modules/project/... ./modules/container/...`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`f03e4c78` `feat(project): add lifecycle and runtime aggregation boundary`

## 2026-06-30 Phase 1 Batch 4 web project list detail and readonly configuration

- 落地 `web/src/modules/project/**` authority owner：建立 project module registration、typed API consumer、locale owner，以及 `list/detail` 页面。
- `list` 页承载 project registry list、filters、summary tags、lifecycle actions 与 detail tab 导航。
- `detail` 页承载 `Overview`、`Services`、`Configuration`、`Activity` 四个页签：
  - `Overview` 保持 summary，不复制 runtime dashboard。
  - `Services` 只展示静态服务定义与 container member/count 聚合，并回跳现有 Container Detail。
  - `Configuration` 保持只读 metadata、preview、single-file content 三段消费。
  - `Activity` 继续只做前端 fan-out，复用现有 container logs/events API。
- 在 batch 内修复了 web governance blockers：
  - 删除未使用 helper。
  - 抽出 module-local shared helpers 以通过 duplicate-code gate。
  - 把 fixed spacing 改为 density tokens。
  - 补齐 ownership mode i18n key，避免可见文案硬编码。
- 本批验证通过：
  - `git diff --check`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`5c593f9f` `feat(project): add phase 1 project web module`

## 2026-06-30 Phase 1 Batch 5 validation drift guard and governance sync

- 重新运行 Phase 1 closeout validation chain，确认当前 authority owner 与 generated/runtime consumers 无 drift：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `python3 scripts/validate_sql_migrations.py --paths server/modules/project/migrations/202606300002_project_registry_baseline.sql`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 同步 `ai-plan/design/domains/compose/Compose项目管理设计.md`，补齐 batch 4 前端 authority 落点，避免 design 与运行面漂移。
- 完成 Phase 1 archive-readiness check：
  - local import / registry / snapshot / lifecycle / readonly configuration / frontend activity fan-out 路径都已落地。
  - `Project` 与 `Container` authority 边界保持稳定，没有引入 project-level runtime persistence 或 backend logs/events aggregation。
- Topic 未进入 `archive-ready`，因为 `Phase 2` 与 `Phase 3` 仍为明确的后续 bounded work。

## 2026-06-30 Phase 2 loop rebatching

- 未接受“Phase 2 仍是大阶段占位符，因此 loop 必须 blocked”这一过早终止结论。
- 在同一 `topic-completion-loop` 下把 Phase 2 重新拆成可执行 bounded batches：
  - `phase-2-batch-1-managed-root-and-create-contracts`
  - `phase-2-batch-2-server-managed-create-and-file-write-path`
  - `phase-2-batch-3-web-managed-create-and-editors`
  - `phase-2-batch-4-diff-validate-and-deploy-flow`
  - `phase-2-batch-5-phase-2-validation-drift-guard-and-governance-sync`
- 保持同一 active topic，不创建新主题，不切换 recovery source。
- loop state 前移到 `phase-2-batch-1-managed-root-and-create-contracts`。

## 2026-06-30 Phase 2 Batch 1 managed root and create contracts

- 落地 managed create 的上游 authority owner，而不是下游兼容层：
  - `openapi/**` 新增 `managed-root`、`create-validate`、`create` canonical contract source。
  - `server/modules/project/**` 新增 managed root system config、create route/permission/message 合同与模块注册接入。
  - `web/src/modules/project/contract/**` 只同步最小稳定消费路径常量。
- 本批明确不实现实际文件写入、editor、diff、validate UI 或 deploy flow。
- 本批验证通过：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`f1f5a72d` `feat(project): define managed create root and contracts`

## 2026-06-30 Phase 2 Batch 2 server managed create and file write path

- 落地 `server/modules/project/**` authority owner：实现 managed create 的服务端 file-write path，在 managed root 下创建 working directory、写 compose/env 文件、解析配置、持久化 registry 与 snapshot bootstrap。
- 同步 `openapi/**` authority owner：为 `POST /api/ops/projects/create` 增加实际 create request payload，并把 create response 修正为同步创建结果语义，去除 batch 1 阶段遗留的 accepted-only 语义。
- create 流程在 registry 失败时清理本轮新建目录和文件，避免留下无主目录。
- 本批验证通过：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `cd server && go test ./modules/project/...`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`9ec8da91` `feat(project): add managed create file write path`

## 2026-06-30 Phase 2 Batch 3 web managed create and editors

- 落地 `web/src/modules/project/**` authority owner：建立 managed create route、managed-root/create/create-validate API 消费、create 页面，以及 Compose/Env editor surface。
- 本批保持在 web authority owner 内，没有进入 diff/deploy flow、remote host、backend runtime-state persistence，也没有改动 `server/**` / `openapi/**`。
- TDesign MCP preflight 已执行并采用：
  - components: `Form`, `Input`, `Textarea`, `Button`, `Card`, `Tabs`, `Alert`, `Drawer`, `Dialog`, `Space`, `Descriptions`, `Tag`, `Empty`
  - queries: `get_component_list`, `get_component_docs`, `get_component_dom`
- 本批验证通过：
  - `git diff --check`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend`
- 本批已提交：`db8c4bf1` `feat(project): add managed create web workflow`

## 2026-06-30 Phase 2 Batch 4 diff validate and deploy flow

- 落地 `openapi/**` + `server/modules/project/**` + `web/src/modules/project/**` authority owner：实现 managed compose project 的 `diff / validate / deploy` 流程。
- `Project` 继续只拥有配置草稿、差异、校验和部署编排，没有引入项目级 runtime 持久化，也没有越界到 container 私有实现或后端 project logs/events 聚合。
- 前端继续保持 `project detail` 的 `list-form-detail` 页型，在 `Configuration` tab 内承接编辑、diff、validate、deploy 流程。
- 本批验证通过：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批已提交：`beb75a48` `feat(project): add managed diff validate deploy flow`

## 2026-07-01 Phase 2 Batch 5 validation drift guard and governance sync

- 重新运行 Phase 2 closeout validation chain，确认 managed create/edit/diff/validate/deploy slice 与 generated/runtime consumers 没有新增 drift：
  - `git diff --check`
  - `node scripts/openapi-bundle.mjs`
  - `python3 scripts/validate_sql_migrations.py --paths server/modules/project/migrations/202606300002_project_registry_baseline.sql`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
- 本批未新增实现 authority owner；本轮只同步 `ai-plan/design/**` 与 active topic recovery materials，记录 Phase 2 acceptance 已可审计、可验收。
- 完成 Phase 2 archive-readiness check：
  - managed root create、Compose/Env editor、diff、validate、deploy 路径均已落地并通过完整验证链。
  - `Project` 与 `Container` authority 边界保持稳定，没有引入 project-level runtime persistence、project-owned container detail 或 backend project logs/events aggregation。
- Topic 未进入 `archive-ready`，因为 `Phase 3` 仍存在明确后续 bounded work。
- 将原来的 `phase-3-discovery-git-template-and-remote-host` 重切为安全的 Phase 3 batches：
  - `phase-3-batch-1-git-template-source-contract-and-boundary`
  - `phase-3-batch-2-directory-scan-and-auto-discovery-candidates`
  - `phase-3-batch-3-remote-host-boundary-and-activity-authority`

## 2026-07-01 Phase 3 Batch 1 git/template source contract and boundary

- 落地 `openapi/**` authority owner：
  - 新增 `GET /api/ops/projects/sources` source catalog contract。
  - 为 project list/detail 以及 managed source 响应补充最小 `source_metadata` / `source_type` contract。
- 落地 `server/modules/project/**` authority owner：
  - 新增 source catalog service 与 route。
  - 现有 managed create 路由收口到 `/create/managed`。
  - git/template 只保留 `planned` source entry，不执行 clone、template instantiate、directory scan、remote host 或 backend activity aggregation。
- 落地 `web/src/modules/project/**` authority owner：
  - `/ops/projects/create` 固定为 source selector。
  - `/ops/projects/create/managed` 承接现有 managed create 页面。
  - `/ops/projects/create/git` 和 `/ops/projects/create/template` 只保留 planned boundary 占位页。

## 2026-07-01 Phase 3 Batch 2 directory scan and auto-discovery candidates

- 落地 `openapi/**` authority owner：
  - 新增 `GET /api/ops/projects/discovery-candidates` 作为 discovery candidate 只读 contract source。
  - 固定 candidate preview 的字段边界：`candidate_key`、`candidate_kind`、`status`、`recommended_action`、`working_directory`、`compose/env files`、`declared_service_names`、`config_hash`、`warnings`、`conflicts`。
- 落地 `server/modules/project/**` authority owner：
  - 以 `managed root` 作为 bounded local directory scan authority。
  - 只返回 directory-scan / auto-discovery candidate preview，不写 registry、不自动 import、不引入后台发现任务。
  - 冲突复用现有 registry conflict 规则，仅返回 `review/import` 建议。
- 落地 `web/src/modules/project/**` authority owner：
  - 在 source selector 下新增 hidden discovery preview 页面。
  - UI 只展示 authority root、候选状态、建议动作与冲突/文件预览，不越界到 remote host 或 backend activity aggregation。

## 2026-07-01 Phase 3 Batch 3 remote-host boundary and activity authority

- 落地 `openapi/**` authority owner：
  - source catalog 新增 `remote-host` planned entry，并固定 `host_scope=remote`。
  - project list/detail 新增 `activity_authority` canonical contract。
  - `source_metadata` 新增 bounded planned 字段：`remote_host_key`、`remote_compose_path`、`activity_authority`、`activity_rollup_scope`。
- 落地 `server/modules/project/**` authority owner：
  - source catalog 新增 remote-host entry，但只保留 route/permission/metadata owner。
  - 本机 project 的 `activity_authority` 固定为 `frontend-fanout`；future remote / backend aggregation 固定为 `backend-planned`。
  - 未引入 remote execution、credential persistence、backend project logs/events aggregation 或 project realtime topic。
- 落地 `web/src/modules/project/**` authority owner：
  - source selector 展示 `remote-host` planned entry 与 host scope。
  - `/ops/projects/create/remote-host` 作为 planned boundary 页面接入。
  - detail 页面显式展示 `activity authority`，并在 Activity tab 提示当前 canonical authority。

## 2026-07-01 Topic archive readiness

- 当前 topic 的 Phase 1、Phase 2、Phase 3 bounded batches 均已完成。
- 主题达到 `archive-ready`：
  - `Project` 继续只拥有 registry、configuration、lifecycle、services aggregation 与 activity entry。
  - `Container` 继续拥有 runtime state、logs、events、stats、shell、inspect、networks、mounts。
  - remote-host 与 backend activity aggregation 仅保留 canonical planned boundary，没有半实现下游兼容层或 runtime 越权。

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
    "phase-3-batch-3-remote-host-boundary-and-activity-authority"
  ],
  "current_batch": "phase-3-batch-2-directory-scan-and-auto-discovery-candidates",
  "next_batch": "phase-3-batch-3-remote-host-boundary-and-activity-authority",
  "closeout_status": "phase-3-batch-2-completed"
}
```
