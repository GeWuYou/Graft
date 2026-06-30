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
- Topic 未进入 `archive-ready`，因为 `Phase 2` 与 `Phase 3` 仍为明确的后续 bounded work；loop state 前移到 `phase-2-managed-create-editor-and-deploy`。

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
    "phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync"
  ],
  "pending_batches": [
    "phase-2-managed-create-editor-and-deploy",
    "phase-3-discovery-git-template-and-remote-host"
  ],
  "current_batch": "phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync",
  "next_batch": "phase-2-managed-create-editor-and-deploy",
  "closeout_status": "phase-1-complete-phase-2-ready"
}
```
