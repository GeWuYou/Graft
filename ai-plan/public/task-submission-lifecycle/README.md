# Task Submission Lifecycle

## Current Status Summary

- Topic objective: 将 Task Reservation 重构为独立 Submission Saga，并以本地原子物化消除永久 pending reservation。
- Current status: `archive-ready`
- Task class: `cross-boundary`
- Intake summary: long-running refactor requiring design, topic, roadmap and ADR convergence.
- Canonical authority:
  - `ai-plan/design/domains/task/任务提交生命周期设计.md`
  - `ai-plan/design/decisions/ADR-022-task-submission-materialization.md`
- Completed: Submission runtime, Build atomic materialization, legacy migration, generated contracts and validation.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: Task Runtime owns Submission/Task state; Build owns its Snapshot writer; OpenAPI owns shared status wire semantics.

## Owned Scope

- `server/modules/task/**`, `server/modules/build/**`, `server/internal/moduleapi/task.go`
- `openapi/**`, generated contract artifacts, `web/src/modules/task/**`
- `ai-plan/design/domains/task/**`, `ai-plan/design/decisions/ADR-022-task-submission-materialization.md`, `ai-plan/roadmap/Task提交生命周期实施计划.md`

Out of scope:

- 外部 prerequisite 的实际 Saga/Outbox runtime 实现。
- 新的 Task producer，除 Build 和现有 legacy Reservation adapter 外。

## Locked Decisions

1. Submission 是独立聚合，Task 从 `ready` 开始，Worker 不感知 Submission。
2. 本地 prerequisite 使用通用 transaction-scoped writer 原子物化；两个事实表的局部唯一索引加 owner advisory lock 保证互斥，不引入第三份业务状态。
3. `activated` 是 Submission 终态；不新增 `activation_pending` 或 reservation events table。

## Phase Plan

- authority-and-contract
- persistence-and-owner-claim
- build-materializer
- legacy-migration-and-projection
- cross-boundary-validation

## Current Recovery Point

- v3 design 和 pasted review 已确认。
- 当前风险：Task status、OpenAPI、migration 与 Build store 必须在同一 authority slice 内同步。
- Next step: archive topic after developer integration review.

## Validation Targets

```bash
git diff --check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/task-submission-lifecycle/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
