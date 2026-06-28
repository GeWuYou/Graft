在同一个 `topic-completion-loop` 下推进 `design-ia-governance`，不要切换到其他 topic。

Round context:

- governance source：root `AGENTS.md`
- task class：`docs/automation`
- recovery source：`parent topic`
- recovery entry：`ai-plan/public/design-ia-governance/README.md`
- local execution truth：
  - `ai-plan/AGENTS.md`
  - `ai-plan/README.md`
- design authority：
  - `ai-plan/design/**`
  - `ai-plan/design/AI任务追踪与恢复设计.md`
  - `ai-plan/design/AI工具与MCP接入治理规范.md`
  - `ai-plan/design/decisions/ADR-001-ai-plan-authority-and-metadata-model.md`
  - `ai-plan/design/decisions/ADR-002-ai-plan-lifecycle-and-archive-model.md`
- AI skills：
  - `$graft-multi-agent-loop`
  - `$graft-ai-plan-governance`

Topic objective:

- 持续推进 `ai-plan/design/**` 的 IA 治理，直到该主题达到 `archive-ready`、进入 `blocked`，或必须重新定义 bounded batches。

Locked decisions:

1. root `AGENTS.md` 继续是唯一 startup-governance source。
2. `ai-plan/AGENTS.md` 是 `ai-plan/**` 的 local execution-truth document。
3. 本主题只治理 `ai-plan/design/**` 的 IA，不引入 whole-repo frontmatter retrofit、projection 层、多语言平台或文档站抽象。
4. `ai-plan/catalog.json` 保持 bounded、single-file、non-authoritative。
5. design 文档迁移必须先做分类和骨架，再分批移动与收敛。

Implementation guardrails:

- 必须优先修 authority owner，不得在下游 README 或 catalog 再造第二套治理真值。
- 不得干扰 `compose-project-management` 或其他并发 active topic 的恢复状态。
- 不得一开始就大规模移动所有 `ai-plan/design/*.md`。
- 不得把 validator 扩成 whole-repo `ai-plan` linter。

Current batch plan:

1. `phase-1-batch-1-design-inventory-and-target-ia-skeleton`
   - 盘点当前 design 文档
   - 建立分类矩阵
   - 定义目标目录骨架
   - 定义一级/二级 README 责任模型
   - 明确后续迁移批次边界
2. `phase-1-batch-2-create-target-design-directories-and-readmes`
3. `phase-1-batch-3-migrate-low-coupling-design-docs`
4. `phase-1-batch-4-migrate-domain-design-docs-and-fix-references`
5. `phase-1-batch-5-design-archive-naming-and-governance-sync-closeout`

Loop instructions:

- 默认 `loop_mode=topic-completion-loop`。
- 每轮只做一个 bounded batch。
- 每轮 closeout 后必须更新：
  - `ai-plan/public/design-ia-governance/todos/design-ia-governance-tracking.md`
  - `ai-plan/public/design-ia-governance/traces/design-ia-governance-trace.md`
- 当 `pending_batches` 非空时，主 agent 必须继续自动选择 `next_batch`，不要在普通 batch success 后停止。
- 每个成功 batch 完成后都要先过对应验证，再决定是否执行 `$graft-commit`。

Validation expectations:

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

Required closeout:

- 明确当前 batch
- 明确变更的 authority owner
- 明确已运行验证与跳过原因
- 更新 loop batch state
- 仅在 terminal state 下输出 `Next-session startup prompt:`
