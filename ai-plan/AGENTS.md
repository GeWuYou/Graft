# ai-plan/AGENTS.md

本文件只拥有 `ai-plan/**` 结构与恢复材料执行规则。根 [AGENTS.md](../AGENTS.md) 是唯一启动、授权、验证归属、
提交和 closeout authority；复用本轮回执，不重新启动。修改治理使用 graft-ai-plan-governance；同时影响 AI 工具、
skills、脚本时配合 graft-ai-governance-audit，仅覆盖必要 owner。

## Authority and Router

- `design/**`：跨主题长期设计真相；`design/decisions/ADR-XXX-*.md`：需要约束后续批次的已收敛决策。
- `roadmap/**`：独立分阶段交付计划，不能复制 tracking 批次表。
- `public/README.md`：只列 active topics，每项一个恢复入口；完成 root preflight 后才用于恢复。
- `public/<topic>/**`：当前恢复和批次状态；`public/<topic>/archive/**`：active topic 已完成阶段历史。
- `public/archive/<topic>/**`：已完成主题历史，不覆盖当前治理，不作为默认启动上下文。
- `lessons/**`：可复用纠错与稳定经验；`templates/**`：最小起始材料，不是活跃主题 authority。
- `README.md`：目录导航；`catalog.json`：单文件、显式有限覆盖的辅助机器索引，不是第二真相。

按变更风险选语义审查，证据可合并，局部文档修复不强制全量设计读取。规则冲突先修最高错误 owner，并同步下游。

## Work Intake and Active Topic

新 long-running 工作先走 graft-work-intake；short-lived 工作不建 topic/Work Contract。详见
[ADR-003](design/decisions/ADR-003-work-intake-and-bootstrap-model.md) 和
[追踪与恢复设计](design/governance/ai/AI任务追踪与恢复设计.md)。只创建 contract 选中的 design/topic/roadmap/ADR。

每个 active topic 必须包含 README.md、startup-prompt.md、todos/<topic>-tracking.md、traces/<topic>-trace.md。
README 提供目标、恢复回执、authority、owned scope 和待办方向；tracking 持有恢复点、checklist、acceptance、batch
状态和 Work Contract（专用标题下一个 YAML block）；trace 记录日期、决策、验证、下一步。禁止独立 work-contract.yaml、
在 README/catalog 复制完整 contract，或要求所有文档新增 frontmatter。

startup-prompt 引用 root preflight，携带 governance source/task class/recovery source/owned scope；父主题先于
子主题恢复。修改 tracked topic 同步 tracking/trace；子主题只写本范围，父主题保留共享里程碑/风险/下一步。

## Editing and Archive

- ADR 先记录 context/decision/status/consequences，再铺开相关 router/skill/validator 批次。模板必须去占位并适配。
- 不批量回写旧主题只为模板一致。仅修正 active recovery 中覆盖新治理的当前通用指令，保留用户明确选型和历史 trace。
- 保留并发 topic 状态；新建/归档同时更新 public/README。完成时整体移到 public/archive 并移除 active 索引。
- active 入口保持简短，阶段详细历史移入 topic archive。无秘密、绝对文件系统路径、用户名、私有地址或本地恢复库；
  使用相对路径、commit、branch、PR 编号和命令。
- 不另建 startup、validation、commit、recovery 或 intake 流程；catalog 不扩成全库清单；批次范围未到不添加工具。

## Validation and Closeout

纯文档运行 `git diff --check`。修改 router/catalog/templates/active recovery 或 structure guard 时运行
`python3 scripts/validate_ai_plan_structure.py`；该检查保持现有 bounded coverage，不是全库 ai-plan lint。
guard 逻辑变更加 `python3 -m unittest scripts/test_validate_ai_plan_structure.py`。
影响 skills/AI 工具规范或相关脚本时运行 `python3 scripts/validate_ai_governance.py`，脚本测试按受影响行为选择。

复用有效检查，root 提交门槛与 hooks 保持。closeout 说明 authority/topic/ADR、路由影响、验证和缺口、所有权/暂存阻塞；
不能把历史归档当当前执行权限。规模预算 4096 UTF-8 bytes；详细契约保持在对应设计/skill reference 中。
