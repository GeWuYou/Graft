# Handwritten Comment Governance

## Current Status Summary

- Topic objective: 分波次治理 `server/**` 与 `web/**` 手写 Go、TypeScript、Vue 注释，使其符合中文高价值注释规范。
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: 这是需要稳定恢复、审计分批、并发执行和逐批验证的长期治理审计。
- Canonical authority:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/design/governance/ai/代码注释与模块文档规范.md`
  - `.agents/skills/graft-comment-governance/SKILL.md`
- Completed so far: startup preflight, G115 lint repair, mixed-commit reconciliation, and eight parallel comment-governance waves
- Current status detail: residual inventory wave completed; archive-readiness remains pending because untouched
  server module and web module scopes still contain audit candidates.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `handwritten-comment-governance`
- authority summary: 注释语义由仓库注释规范统一定义，server/web 子域规则约束执行与验证边界。

## Owned Scope

- handwritten Go under `server/**`
- handwritten TypeScript and Vue under `web/**`
- topic-local audit inventory and recovery state

Out of scope:

- generated, third-party, migration, and build-artifact source files
- behavior changes, refactors, dependency changes, whole-repository formatting, and unrelated fixes
- creation of a second audit, validation, or recovery path

## Locked Decisions

1. 先只读审计，再按不重叠目录/模块切片委派。
2. 委派前必须验证编排器实际可选且可证明的 worker 配置：`model=gpt-5.6-luna`、`reasoning_effort=medium`。

## Phase Plan

- 只读注释审计、候选分类与批次边界冻结
- server 与 web 分波次治理、逐批注释回执与最小验证
- 主 Agent 集成复核、剩余范围盘点与归档准备

## Current Recovery Point

- 已完成启动收据、权威文档读取、G115 修复提交 `e3806925` 和混合提交范围复核；未改写 `e9fb50d4` / `249280ed` 历史。
- 本回合已完成八个不重叠波次；每个 worker 均完成 scoped commit，主 Agent 保留编排层模型证据 `gpt-5.6-luna/medium`。
- backend lint 已通过；web 全量检查仅剩既有 `configuration-workspace` 测试失败。
- Next step: continue from the residual inventory of untouched server modules and web modules; archive readiness is not yet established.

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.
- Use this README for navigation, summary, and recovery entry only.

## Pending Batch Direction

- 已完成波次：project workspace/canonical/project boundary、auth、system-config/task、web project、web shared/request、web auth/dashboard/monitor。
- 后续批次：对剩余 server module 与 web module 候选继续做只读 inventory 后再冻结互斥写集；本回合已完成约 20% 的可审计增量，停止并交接。

## Validation Targets

```bash
git diff --check
```

批次代码验证按 changed scope 执行 `graft validate backend` 或 `bun run check`；只读审计不伪造运行时验证。

## Loop Entry

- Preferred entry: `ai-plan/public/handwritten-comment-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch` for disjoint module slices; main Agent owns acceptance and archive readiness.
