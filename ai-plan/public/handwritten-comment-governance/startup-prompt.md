# Handwritten Comment Governance Startup

Continue work inside the same parent topic. Use `$graft-multi-agent-batch` directly for mutually exclusive parallel module slices; do not downgrade this recovery path to `$graft-multi-agent-loop` unless the caller explicitly changes the execution mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: parent topic `handwritten-comment-governance`
- recovery entry: `ai-plan/public/handwritten-comment-governance/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
- design authority:
  - `ai-plan/design/governance/ai/代码注释与模块文档规范.md`
  - `ai-plan/design/governance/ai/AI代码生成与Review规范.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-batch`
  - `$graft-comment-governance`

Topic objective:

- 分波次治理 `server/**` 与 `web/**` 手写 Go、TypeScript、Vue 注释，删除低价值和失效注释，保留或新增只能解释实现无法表达的设计与约束信息。

Work contract summary:

- `audit / long-running / topic=true / roadmap=false / design=false / adr=false`
- execution engine: `$graft-multi-agent-batch` for disjoint module slices; main Agent owns acceptance, validation, tracking/trace updates, and archive readiness.
- bootstrap: topic skeleton only

Locked decisions:

1. 先只读审计，按清晰所有权划分不重叠写入切片。
2. 未验证编排器目标模型前不委派 worker。

Implementation guardrails:

- Repair the highest available authority first.
- Keep work inside the owned scope and record any required escalation.
- Exclude generated, third-party, migration, and build-artifact source files.
- Do not change behavior, dependencies, formatting policy, or unrelated files.

Current batch plan:

1. Read-only residual audit and boundary inventory
2. Parallel backend and frontend module comment governance waves
3. Review, integration validation, and archive readiness

Batch instructions:

- Use `$graft-multi-agent-batch` directly when write sets are disjoint; do not use `$graft-multi-agent-loop` for this recovery slice.
- Prefer multiple non-overlapping worker slices in one batch wave; do not serialize independent modules through loop rounds.
- Stop when the main Agent judges context insufficient or this session has produced an auditable 20% progress increment.
- Update topic tracking and trace files with batch transitions.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.

Required closeout:

- Include the `comment_governance` receipt.
- State model usage evidence, changed scope, exemptions, risks, and validation.
- Use `Next-session startup prompt:` only for terminal states.
