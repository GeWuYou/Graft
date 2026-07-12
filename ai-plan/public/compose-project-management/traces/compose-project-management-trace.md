# Compose Project Management Trace

## 2026-07-11 Import Lifecycle Confirmation

- 将运行时 Compose 导入从“注册后进入详情页审核”收口为导入向导内的强制生命周期审核步骤。
- OpenAPI 检查响应提供 project-authority 默认配置；最终导入请求携带该配置，并在同一注册写入中持久化为 `confirmed`。
- 导入流程保持不执行 lifecycle action；仅保存后续 `up`、`stop`、`restart` 和 `redeploy` 的结构化策略。

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

## 2026-07-04 Lifecycle configuration authority and docs sync

- 落地 `openapi/**` authority owner：
  - 新增 `PUT /api/ops/projects/{id}/lifecycle-configuration`。
  - 新增 `LifecycleStrategyKind`、`LifecycleReviewStatus`、saved lifecycle configuration 与 generated command preview contract。
  - 从 project action/batch-action/OpenAPI path 中移除 `update-deploy`；`redeploy` 成为 canonical deploy-style lifecycle action。
- 落地 `ai-plan/design/**` 与 active topic authority：
  - `Compose项目管理设计.md` 改为明确本地项目统一保存 `Lifecycle Configuration`，managed 默认 `confirmed`，imported 默认 `review_required`。
  - detail IA 新增 `Lifecycle` 页签；`Configuration` 继续保留 managed draft editor/diff/validate/deploy flow。
  - managed draft `deploy` 明确复用已保存 lifecycle configuration 的 final compose `up`，不再假定固定 `docker compose up -d`。
- 本批验证通过：
  - `git diff --check`
- 本批保持在 `openapi/** + ai-plan/**` authority owner 内，没有改动 `server/**` 或 `web/**`，也没有更新 generated artifacts。

## 2026-07-01 Import Existing Project folder-picker and inspect flow sync

- 收口 `openapi/**` authority owner：
  - 新增 `GET /api/ops/projects/import/directory-sources`
  - 新增 `GET /api/ops/projects/import/directories`
  - 新增 `POST /api/ops/projects/import/inspect`
  - `POST /api/ops/projects/import` contract 改为 `inspection_id + editable overrides`
- 收口 `server/modules/project/**` authority owner：
  - 新增 import directory browse / inspect flow
  - 通过短 TTL inspection cache 复用 inspect parse 结果并校验 file hash freshness
  - import 阶段不再信任前端回传 working directory / compose / env file 集合
- 收口 `web/src/modules/project/**` authority owner：
  - 新增 `FolderPicker.vue`
  - import 页面改为 `select directory -> inspect -> preview -> import`
  - compose/env/services/networks/volumes 改为 inspect readonly preview
- 同步 generated artifacts：
  - `openapi/dist/openapi.bundle.json`
  - `server/internal/contract/openapi/generated/types.gen.go`
  - `server/internal/app/zz_openapi_bundle_generated.go`

## 2026-07-03 Drift repair for aggregated project status and resource summary

- 修正 `openapi/**` authority owner：
  - `project-runtime-status` 从 `running | partial | stopped | empty` 收口为 `running | degraded | stopped | transitioning | unknown`
  - `project-container-counts` 扩展为 `running / stopped / transitioning / issue / total`
- 修正 `server/modules/project/**` authority owner：
  - 新增统一项目聚合状态推导 helper，列表与详情共用
  - 修复 `runtime_status` 没有写回 `ProjectListItem / ProjectDetailResponse` 的缺口，避免前端长期掉到 `未知`
  - `project` 继续只消费 `container` shared boundary 的成员状态，不引入 project-owned runtime detail
- 修正 `web/src/modules/project/**` consumer：
  - 列表列从 `运行时摘要 + 服务数 + 容器数量` 收口为 `状态 + 资源`
  - 资源列改为两行聚合摘要，展示服务数与容器聚合计数
  - 补齐资源列本地化，移除硬编码英文标签
  - `web/src/contracts/openapi/generated/schema.ts`
- 本轮验证通过：
  - `node scripts/openapi-bundle.mjs`
  - `cd server && go generate ./internal/contract/openapi ./internal/app`
  - `cd server && go test ./modules/project/...`
  - `cd web && bunx vitest run src/modules/project/shared/useProjectImportFlow.test.ts`
  - `cd web && bun run typecheck`
  - `Container` 继续拥有 runtime state、logs、events、stats、shell、inspect、networks、mounts。
  - remote-host 与 backend activity aggregation 仅保留 canonical planned boundary，没有半实现下游兼容层或 runtime 越权。

## 2026-07-01 Drift repair reopened

- 实机检查 `/ops/projects/create` 发现该页面已被 source selector 占用，并直接向用户暴露 raw i18n key 与内部 Phase 3 batch 文案。
- 复核 `Compose项目管理设计.md` 后确认 Phase 1 主入口应为 `Import Existing Project`，而不是 Phase 3 boundary surface。
- 主题从错误的 `archive-ready` 结论回滚到 `active`，先执行 `drift-repair-import-primary-entry-and-topic-truth`。

## 2026-07-01 Drift repair narrowed to runtime-candidate primary import

- 收口 `ai-plan/design/domains/compose/Compose项目管理设计.md` authority truth：
  - `Import Existing Project` 主流程固定为 `runtime candidate -> inspect -> preview -> import`
  - `directory browse / inspect` 保留为非主入口 inspect/file-system 复用底座，不再定义当前主 IA
  - runtime candidate 必须同时返回 `ready` 与 `unavailable`
  - `config_files` 固定为 stronger authority；`working_directory` 只作为 hint，并允许从 `config_files[0]` 派生
- 收口产品/契约共识：
  - 不完整 runtime metadata 不能让 candidate 静默消失，必须返回不可导入状态与稳定 reason code
  - `project` 继续只消费 runtime candidate authority，不直接解析 Docker labels
- 当前 drift-repair 的实现目标更新为：
  - `server/openapi`：新增 runtime candidate list/runtime inspect contract，并扩展最小 shared boundary
  - `web`：`/ops/projects/import` 改为 runtime candidate 列表驱动，Folder Picker 退出主交互

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

## 2026-07-11 Creation pipeline contract and server foundation

- `project` 新增 source-neutral creation pipeline；Managed materialize 与 Import adopt 在成功解析真实 workspace 后共用 lifecycle-confirmed aggregate、snapshot 与只读 runtime boundary，不自动 deploy。
- managed create contract 接受完整 text workspace manifest、显式 Compose/Env 引用及 lifecycle configuration；现有 compose/env 字段暂保留给当前 UI 过渡。
- `compose_projects.source_metadata_json` 成为无密钥来源 provenance 的持久化 owner；列表/详情优先读取持久化 metadata。
- Managed writer 支持 nested 与 dot text files，拒绝绝对路径、traversal、重复路径、NUL/非 UTF-8 内容；失败时仍只回滚该请求创建的内容。
- 下一批：`managed-workspace-wizard-and-lifecycle-review`，不改变 Container runtime authority。

## 2026-07-11 Managed workspace wizard and lifecycle review

- Managed Create 已使用 `Identity & Managed Root -> Workspace -> Lifecycle -> Review -> Create` 向导，草稿工作区支持 nested 与 dot text files，并通过 `ProjectMonacoSurface` 和现有语言解析保持编辑体验一致。
- 生命周期表单与命令预览被抽取为 source-neutral component；Import 仅保留其 inspect refresh 和专属步骤操作包装。
- 创建请求携带 canonical workspace manifest、Compose/Env references 和 lifecycle configuration；成功后进入配置工作区且不触发 deploy。
- 下一批：`import-creation-adapter-and-regression`。

## 2026-07-11 Import creation adapter and regression

- Import inspection commit 已确认通过 `CreationCommand` 进入 source-neutral creation pipeline；candidate availability、inspection TTL、file-hash freshness、conflict 与 adopt-without-write 仍由 Import adapter 独占。
- 回归覆盖 imported/local/external aggregate metadata、workspace files、snapshot、clean drift 与 confirmed lifecycle；既有 stale hash 回归继续保持 `inspection stale` 映射。
- Import 最终审核复用 source-neutral lifecycle configuration review，并明确 Import 只注册已检查的 workspace、打开项目详情，不会自动 deploy 或启动容器。
- 验证：focused Go import regressions、focused Vitest import flow/page tests、`bun run lint:i18n`、`git diff --check`；browser QA 由用户明确延期。
- 剩余 batches：`git-template-source-adapters`、`remote-source-adapter-and-activity-boundary`、`optional-deploy-after-create-and-archive-readiness`；下一批为 `git-template-source-adapters`。

## 2026-07-11 Git and Template source adapters

- Git source now clones only into request-scoped isolated staging with terminal prompts disabled, rejects symlinks/binary or oversized workspace files, and persists only repository URL, resolved reference, and Compose subpath.
- Template source owns a small module-local `empty-compose v1` text workspace catalog; it has no marketplace, dynamic discovery, or credential persistence.
- Both source routes validate/materialize under the managed root, parse the actual workspace, and invoke `CreationCommand`; neither source starts Compose services or deploys automatically.
- `/ops/projects/create/git` and `/ops/projects/create/template` now provide usable source forms and no longer reuse the planned-boundary page. Browser QA remains user-deferred.

## 2026-07-12 Optional deploy after create

- Managed and Template UI now expose an opt-in, default-off post-create deployment choice only when the caller has `ops.project.deploy`.
- The client first receives the successful create response, then invokes the existing independent Deploy operation. Deploy errors remain separate from creation and preserve the registered `Ready` project.
- Import and Git stay non-deploying in this batch; Remote is not expanded. User-directed source-surface simplification is the next bounded batch, so this topic is not archive-ready.

## 2026-07-12 Source surface simplification and extension seam

- 用户将可用来源固定为 Managed、Template 与 Import Existing；Import 继续是独立的 runtime-to-workspace 主流程。
- 移除了 Git 与 Remote Host 的公开 API、OpenAPI schema、catalog entry、路由、页面和 metadata；没有用 planned placeholder 替代。
- 保留 source-neutral `CreationCommand` / `createProjectFromWorkspace`：未来来源必须先由 adapter 构建或获取 Workspace，再进入统一 lifecycle/review、aggregate/snapshot 和只读 runtime sync。
- 该范围不包含浏览器验证，按用户要求由用户自行验证。
- archive-readiness 验证通过：diff、OpenAPI bundle/generation、backend validate、web check、i18n 与 ai-plan structure guard；主题达到 `archive-ready`。

## 2026-07-12 Unified creation entry

- 列表页收敛为唯一的“创建项目”操作；`/ops/projects/create` 是 `blank`、`template` 与 `import` 的统一选择入口。
- `GET /api/ops/projects/creation-methods` 取代旧 `/sources` 目录。后端仅发布创建方式、可用性和阻塞码；项目已持久化的 `source_kind` 继续描述项目来源。
- 空白创建、模板创建和导入分别进入 `/ops/projects/create/blank`、`/ops/projects/create/template` 与 `/ops/projects/create/import`，未保留旧路由或旧 API 别名。
- `ops.project.source.view` 经 RBAC migration 直接更名为 `ops.project.creation-method.view`，保留原 permission ID 与既有角色关联。

## 2026-07-12 Saved-view foundation and project contract

- 新增 module-owned `saved_views` 表和 generic `moduleapi.SavedViewService`，没有菜单、快捷操作或无法实施领域授权的公共 HTTP API。
- 表按 owner user 与 consumer `surface_key` 隔离；live 名称通过部分唯一索引保持唯一。持久化筛选/查询 JSON、页大小和可见列；当前页不持久化，应用视图从第一页开始。
- `project` 保持 `/api/ops/projects/saved-views` 的授权与 payload authority，以 `ops.project.view` 保护，并仅接受当前 project list 定义的筛选字段和列键。
# 2026-07-12 Application Runtime Target Association

- Runtime Target remains the target/provider authority; Project stores only nullable migration-bridge `runtime_target_id` and consumes summaries through `moduleapi`.
- New Compose registrations resolve a Docker target before persistence. Historical local rows are idempotently backfilled after Runtime Target Boot discovery.
- `/api/ops/projects` now models the generic application list projection with `application_type=compose`, target summary and server-side identity/target/provider/source/drift filters. The bridge becomes non-null only after live backfill evidence confirms no remaining rows.

## 2026-07-12 Application Management Web And Icon System

- `/projects` remains the stable route but now presents Application Management. Compose is visible as the current application type, while Runtime Target and Provider are first-class columns and server-backed filters.
- The Project saved-view consumer now lists, applies, creates, updates and deletes user-private saved views. Applying a view restores filters, page size and visible columns while resetting to page one.
- Menu descriptors use distinct semantic identifiers across visible domain and entry nodes. The web resolves them through one static Iconify boundary: Lucide for ordinary navigation and Tabler's outline Docker brand glyph for Docker.
- The icon authority repair expanded to core domain descriptors plus currently visible module entries; it changes only menu metadata and resolver mappings, never routes, permissions, or page behavior.
- Browser evidence captured Application Management, Runtime Targets and the visible Docker sidebar logo under `.ai/artifacts/browser/application-management-page` and `.ai/artifacts/browser/runtime-target-icons`.
- Follow-up visual audit replaced legacy/fallback menu identifiers for core domains and visible observability, security and platform entries. Final browser evidence at `.ai/artifacts/browser/observability-icons-final` shows Overview, Runtime, Dependencies, Module Runtime, Access Log and App Log with distinct semantic glyphs; Docker uses the Tabler outline brand glyph.

## 2026-07-12 Cross-boundary acceptance and recovery update

- Accepted the saved-view contract: generic storage is private to `(owner user, surface)` and enforces one live display name per scope. It persists filters, page size and visible columns, while the Project consumer validates query/column state and always restores at page one.
- Accepted the application projection: Compose stores only a Runtime Target reference and consumes its target/provider summary; list keyword and application, target, provider, source, runtime and drift filters are server-owned. Container remains the runtime authority.
- Accepted the Application Management UI and icon boundary: `/projects` supports saved-view CRUD and target/provider filtering; menu rendering has one static Iconify resolver, with Lucide defaults and Tabler's outline Docker logo.
- Validation passed: `git diff --check`, `python3 scripts/validate_sql_migrations.py`, `node scripts/openapi-bundle.mjs`, `cd web && bun run openapi:types:check`, targeted Go/Vitest suites, `cd server && go run ./cmd/graft validate backend`, `cd web && bun run check`, `python3 scripts/validate_shared_asset_registries.py`, and `python3 scripts/validate_ai_plan_structure.py`.
- Archive-readiness remains false. The previously deferred `remote-source-adapter-and-activity-boundary` is independent Phase 3 work; it must not be represented by a placeholder route, API, or source catalog entry.
