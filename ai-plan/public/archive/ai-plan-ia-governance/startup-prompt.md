`ai-plan-ia-governance` 已归档。不要把这个 archived topic 当成 active 实现入口继续推进。

Round context:

- governance source：root `AGENTS.md`
- task class：`docs/automation`
- recovery source：`parent topic`
- recovery entry：`ai-plan/public/archive/ai-plan-ia-governance/README.md`
- local execution truth：
  - `ai-plan/AGENTS.md`
  - `ai-plan/README.md`
- design authority：
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
  - `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`
  - `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`
- AI skills：
  - `$graft-multi-agent-loop`
  - `$graft-ai-plan-governance`
  - `$graft-ai-governance-audit`

Topic objective:

- 这份归档材料只保留治理收口证据，不再承担 active loop entry。
- 后续新的 `ai-plan/**` governance slice 应使用 root startup preflight + `$graft-ai-plan-governance` 重新开题。

Locked decisions:

1. root `AGENTS.md` 继续是唯一 startup-governance source。
2. `ai-plan/AGENTS.md` 是 `ai-plan/**` 的 local execution-truth document。
3. active topic 的最小恢复材料固定为 `README.md`、`startup-prompt.md`、tracking、trace。
4. `ai-plan/public/README.md` 只列 active topics，不承担 archive 摘要。
5. `ai-plan/design/decisions/**` 用于收敛多个后续 batch 之前必须先锁定的治理 ADR。
6. archive lifecycle 必须通过显式目录位置与 index 更新表达，不能靠隐式状态文本漂移。

Implementation guardrails:

- 必须优先修 authority owner，不得在下游文档再造第二套治理真值。
- 不得干扰 `compose-project-management` 或其他并发 active topic 的恢复状态。
- `ai-plan/catalog.json` 已存在；保持它 single-file、bounded、non-authoritative，不得扩成 whole-repo inventory、
  projection 层或 frontmatter retrofit 触发器。
- 在到达对应 batch 之前，不得提前创建 validator script 或新 skill。
- 不得批量 retrofit 现有 topic frontmatter，也不得做大规模 `ai-plan/design/**` 目录移动。
- `ai-plan/templates/**` 只提供 minimal starters；只有真正创建新 topic 或 ADR 时才按需复制并改写。
- 当前 validator 已落地；除非对应 batch 明确扩大 scope，不得把它扩成 whole-repo ai-plan linter。

Restart guidance:

- 如果任务是继续 `compose-project-management`，改用
  `ai-plan/public/compose-project-management/startup-prompt.md`。
- 如果任务是新的 `ai-plan/**` docs/automation 治理切片，重新执行 root startup preflight，读取 `ai-plan/AGENTS.md`，
  然后使用 `$graft-ai-plan-governance`。
