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
- Completed so far: startup preflight, Work Intake bootstrap, model verification, and first-wave dispatch
- Not started yet: first-wave closeout and main-agent acceptance

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
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

- 已完成启动收据、权威文档读取和初步文件规模统计。
- 已验证 worker 配置为 `model=gpt-5.6-luna`、`reasoning_effort=medium`。
- 第一波已启动：Pascal 只读审计，Kierkegaard 负责 server core 切片，Epicurus 负责 web app/layouts 切片。
- Next step: 等待各 agent final closeout，随后由主 Agent 复核差异并执行对应验证。

## Work Intake

- This topic was created through `Work Intake`.
- Persist the full `Work Contract` in the tracking file, not here.
- Use this README for navigation, summary, and recovery entry only.

## Pending Batch Direction

- 第一波：comment-audit-agent 只读审计，同时执行 server core 与 web app/layouts 不重叠治理。
- 后续批次：按 audit inventory 继续分配 server 包集合、web 模块/壳层集合和只读 review。

## Validation Targets

```bash
git diff --check
```

批次代码验证按 changed scope 执行 `graft validate backend` 或 `bun run check`；只读审计不伪造运行时验证。

## Loop Entry

- Preferred entry: `ai-plan/public/handwritten-comment-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
