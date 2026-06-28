# Compose Project Management Tracking

## Topic

Compose Project Management

## Scope

在 `Graft` 新增 Docker Compose Project 管理能力，保持 `Project` 作为管理与聚合层，保持 `Container` 作为运行时 authority，并按 Phase 1-3 分阶段完成导入、项目注册、生命周期、配置只读、活动聚合与后续扩展。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/Compose项目管理设计.md`
- `ai-plan/design/容器管理设计.md`
- `ai-plan/design/模块与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/服务端API边界与兼容治理规范.md`
- `ai-plan/design/后端安全与信任边界治理规范.md`
- `.agents/skills/graft-multi-agent-loop/SKILL.md`

## Current Recovery Point

- Phase 0 已完成：
  - Compose Project authority 文档已落地
  - active topic / tracking / trace / startup prompt 已落地
- 当前实现尚未开始，仓库中仍无 `project` module。
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
- [ ] phase-1-batch-1：project contract、route space、data model、migration plan
- [ ] phase-1-batch-2：server project module skeleton、repository、import validate/import/register/refresh
- [ ] phase-1-batch-3：lifecycle executor、ownership guard、container aggregation shared boundary
- [ ] phase-1-batch-4：web project module list/detail/overview/services/configuration/activity
- [ ] phase-1-batch-5：Phase 1 validation、drift guard、docs sync、Phase 1 archive-readiness check
- [ ] Phase 2：managed create、editor、diff、validate、deploy
- [ ] Phase 3：git/template/scan/discovery/remote-host/backend activity aggregation

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

## Phase 3 Acceptance Conditions

- 支持 git/template/scan/discovery/remote-host 扩展路径
- 支持后端 project activity aggregation authority

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-authority-and-topic-persistence"
  ],
  "pending_batches": [
    "phase-1-batch-1-project-contract-and-data-model",
    "phase-1-batch-2-server-project-module-import-and-refresh",
    "phase-1-batch-3-server-lifecycle-and-container-aggregation-boundary",
    "phase-1-batch-4-web-project-list-detail-and-readonly-configuration",
    "phase-1-batch-5-phase-1-validation-drift-guard-and-governance-sync",
    "phase-2-managed-create-editor-and-deploy",
    "phase-3-discovery-git-template-and-remote-host"
  ],
  "current_batch": "phase-0-design-authority-and-topic-persistence",
  "next_batch": "phase-1-batch-1-project-contract-and-data-model",
  "closeout_status": "phase-0-completed"
}
```
