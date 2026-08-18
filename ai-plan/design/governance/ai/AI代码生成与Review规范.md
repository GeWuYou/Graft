# AI代码生成与Review规范

## 1. 目标

本规范定义 `Graft` 中 AI agent 参与代码生成、修改、评审与多 agent 协作时的执行边界。

目标：

- 限制 AI 在 authority 不清、风险过高或跨切片过大的场景下擅自扩张。
- 统一 closeout 证据，避免“改了什么、风险在哪、怎么回滚”不可复核。
- 给单 agent 与多 agent 协作提供相同的 review 清单。

## 2. 默认工作方式

- 先识别 authority owner，再冻结改动范围。
- 只在当前 owned scope 内修改；发现 authority 在上游时，先升级任务范围，不做下游补丁伪完成。
- 没有明确要求时，不做顺手修复、顺手重命名、顺手升级依赖或顺手整理结构。
- 以最小闭环完成当前切片：规则、实现、必要测试、closeout 证据。
- 前端页面的数据流设计先区分 server state 与 UI/client state；出现重复请求、手工 loading/error/data、手工 refresh / 去重 / 轮询或 realtime 后 HTTP 刷新时，先按 `ai-plan/design/architecture/前端架构设计.md` 评估 `@tanstack/vue-query`。
- 不把 TanStack 当作默认替换方案：Table、Virtual、Router 和 Form 只有在既有能力被可复核的性能或维护性证据否定后才可进入独立设计。

## 2.1 Inspection 与写操作授权

- PR review 与 security alert 请求默认只授权只读 inventory、证据提取、finding 分类和根因诊断。
- 只有用户明确要求修复，才授权进入对应代码 repair；只读审查本身不授权修改工作树。
- GitHub 评论、alert 回复或关闭、commit、push、PR 创建或更新等 external write 必须由用户明确要求，并继续受
  对应 `graft-commit`、`graft-push`、`graft-pr-create` 或专项 skill 约束。
- 用户显式调用 `graft-pr-remediation` 时，该调用就是当前分支 PR 的 bounded repair / validate / commit / push /
  supported-AI thread reply / resolve / managed-ledger append 授权；它不授权 force-push、merge、PR metadata、
  arbitrary issue comment、human-authored thread resolution 或 unbounded finding。所有回复、resolve 与 ledger 写入
  仍必须在远端分支和 PR head 精确匹配本地 HEAD 后执行，ledger schema 和 append-only payload validation 继续由
  `graft-pr-review` 持有。
- 从 inspection 进入 repair 或 external write 时必须重新确认 authority、owned scope、验证和 closeout 路径；
  不得把“finding 可执行”解释为已获得写操作授权。

## 2.2 Semantic Review Layer

语义审查是设计阶段的默认层，不是用户显式要求时才运行的工具。Boot 必须按 authority、task class 和影响面
主动匹配所有适用 Review Skill，并在设计或 Work Contract 中记录选择、发现和未决决策。

匹配矩阵：OpenAPI/共享 wire contract 使用 `graft-openapi-contract-review`；跨 server/OpenAPI/generated web/query
链使用 `graft-cross-boundary-review`；平台/runtime/Task/Submission/config/source-of-truth 使用
`graft-platform-architecture-review`；API、domain、event、TypeScript、Query key、permission、module、test seam、
change、consistency、delete 分别使用同名 `graft-*` Review Skill。

缺失的 Skill 必须记录为 skill gap 并使用最近的现有治理清单，不得静默跳过。Semantic Review 只提供 authority、
边界、语义风险、设计决策和验证建议，不创建第二套 startup、validation、commit、closeout、issue 或 recovery truth。
设计阶段发现的 blocker 必须在实现前解决，或转化为明确的风险、验收条件和后续责任。

## 3. Agent 禁区

以下场景默认属于 agent no-go area，未经明确授权不得直接落地：

- 跨模块重构
- authority 与范围不清的跨 `server` / `web` / OpenAPI / shared contract 连锁改动
- 自动生成或自动改写数据库迁移
- 依赖升级、锁文件批量刷新、工具链版本漂移
- 大范围 rename wave、目录迁移、批量移动文件
- 以“统一风格”为名的仓库级格式化或整理 import

Agent 默认不得执行最终 merge 或 cherry-pick；开发者在当前任务中明确授权具体集成操作后，Agent 可以在主工作区执行该操作，但不因此取得最终仓库状态 authority。普通改动可以在临时工作树中完成验证并提交经过验证的任务分支；开发者在主工作区 review 后负责集成。若 authority discovery 已将 OpenAPI 变更确定为 bounded cross-boundary slice，则它不因同时触及 `openapi/**`、server generated binding 和 web generated schema 而成为禁区。

Atlas migration、非 OpenAPI generated code、lock file 和 snapshot 继续属于线性资源。OpenAPI source 与确定性 generated artifacts 则属于同一个 task-owned contract closure：Agent 修改 API contract 时必须在任务分支中同步生成、验证并提交 `openapi/**` source、bundle、server binding、runtime embedded bundle 与 web schema，并执行 `just generate`、`just openapi-check` 及 task class 要求的完成态验证，不能把 freshness drift 留给集成阶段。并行分支发生冲突时，开发者先合并 canonical source，再从合并后的 canonical OpenAPI source 重新生成并复验，而不是手工拼接生成文件。

这些场景如确有必要，必须先形成明确设计、范围、验证与回滚路径，再进入实施。

## 4. 禁止机会主义修复

AI 在当前任务中不得：

- 顺手修 unrelated warning、lint、注释、命名或历史遗留
- 把局部问题扩成跨切片整治
- 借用户没限制的空档做个人偏好的架构调整
- 在评审阶段夹带功能修改

例外只允许在以下同时满足时发生：

- 与当前 authority 修复直接相关
- 不扩大 task class
- 有直接验证
- closeout 明确列出

当验证或提交链路暴露出需要修复的问题时，Agent 仅可在根因已诊断且被直接修复、归属明确、全部文件/hunk 均留在用户已确认 task scope 内、并且 authority 与行为均不变时连续完成修复，并重跑受影响验证。任一条件不满足时，必须遵循根 `AGENTS.md` 的 `Repair Confirmation Interaction Contract`，形成结构化 `Repair required` proposal，而不是只写“暂停”或询问 `Approve?`、`Should I fix this?`、`Confirm repair?` 要求用户重新说明方向。proposal 必须包括失败命令和根因、精确文件路径与行号或 hunk、拟改内容、ownership 与 blast radius、将重跑的验证，以及独立 repair commit 或合并当前 commit 的策略。Agent 必须优先通过宿主原生 structured-choice interaction（例如 `request_user_input`）提供 `execute_repair`、`continue_current_scope`、`show_detailed_diff`、`cancel_workflow` 四个选项，不能在普通回复中要求用户输入编号；只有 `execute_repair` 才允许修改、暂存、提交或推送结构化提案中展示的 repair，`show_detailed_diff` 必须展示 patch 后重新调用同一控件，且新文件或 hunk 必须重新提案。只有宿主确认不支持原生交互时，才允许输出完整 proposal、四个数字选项各自的说明与后果后停止，并等待下一轮用户仅回复 `1 / 2 / 3 / 4`；该 fallback 必须映射到同一四个选项，不能用于方便起见的降级或二元确认。

## 5. 禁止 TODO 泄漏

完成态代码不得新增以下内容作为未交付占位：

- 无责任人的 `TODO`
- 无退出条件的 `FIXME`
- “后续再补测试 / 审计 / 权限”的占位注释
- 伪实现、空分支、静默 fallback

若必须保留临时标记，至少满足：

- 使用统一格式
- 写明责任边界
- 写明清理触发条件
- 本次改动已具备最小可接受行为

## 6. 验证责任与人工验收

任务开始时必须先记录验证分类，不得因为改动涉及页面、交互或截图就默认启动浏览器：

- `agent-only`
  - 可由编译、静态检查、单元/组件/集成测试、API/HTTP、OpenAPI、迁移或仓库完成态入口直接证明；无剩余产品判断。
- `human-acceptance-required`
  - 自动验证完成后，仍需人工判断视觉质量、可用性、真实角色体验、多步骤业务体验或环境依赖结果。
- `browser-automation-required`
  - 已登记的 CI 浏览器测试契约，或已获本任务本地授权且只能在真实浏览器复现/诊断的缺陷。
- `mixed`
  - 先完成客观自动验证，再交付最小人工验收流程。

Agent 自动验证优先使用当前仓库的静态检查、受影响 Go/Vitest 测试、`graft validate backend`、`bun run check`、
OpenAPI/contract freshness、迁移 SQL 验证和已有 API/HTTP 检查。UI 不等于人工验收：可由现有组件、路由、状态或
Vitest 测试证明的行为仍由 Agent 验证。

本地浏览器是受控诊断和检查工具，不是默认完成态。只有用户/开发者在本任务明确授权，且需要调查浏览器专属缺陷、
真实浏览器环境，或用户明确要求浏览器自动化时，才可按 `web/AGENTS.md` 和 `graft-web-browser-agent` 执行。未来已
登记且隔离的 CI browser test 可在 CI 自动运行，但当前本地 Agent 操作浏览器始终需要授权。截图、DOM snapshot 和
artifact 只记录检查证据，不能替代自动化测试或人工验收。

授权后的本地浏览器目标可从 Git 忽略的 `.ai/private/graft-browser-targets.yaml` 选择；它支持多个 environment、
instance 和 service，但只属于开发者本机元数据。首次调用仅在文件不存在时创建非秘密占位模板，不得覆盖已有配置；
凭据必须保持私有且在所有可见输出中脱敏。branch / HEAD 不从配置取得；每次浏览器运行前都必须动态验证所选 runtime
正在服务的 branch 与完整 HEAD，并在无法验证或不匹配时停止。验证必须通过 approved runtime identity
process 独立完成，配置本身不是证据，本规范也不假定存在返回身份的特定 server endpoint。selector 解析已配置
defaults，显式 `--url` 只作为单次覆盖；目标清单不得持有可信 `runtime_identity`，base URL 也不得进入 tracked
文件或 summary。

当自动验证通过但仍需真实用户判断时，Agent 停止继续验证，输出：

`Implementation complete; automated verification passed; awaiting human acceptance.`

最小人工验收契约必须只包含：

- 前置条件（分支/版本、必要服务或测试数据）
- 登录角色
- 3 至 7 个操作步骤与每步预期结果
- 必要的负向场景
- 清理步骤
- 已完成的自动化验证与已知缺口

## 7. Closeout 最低要求

AI 参与的任务 closeout 必须包含：

- 变更摘要
- 风险摘要
- 验证结果
- 回滚思路

推荐结构：

```text
AI closeout:
- summary: <改了什么>
- risk: <主要风险或 not-applicable>
- validation: <运行的命令或未运行原因>
- rollback: <如何回退或 not-applicable>
```

如果存在兼容桥接、批量操作、安全影响或多 agent 协作，closeout 还应补充对应专项证据。

若 `human_acceptance` 为 `required`，closeout 还必须给出最小人工验收契约，并将验收状态写为
`awaiting_human_acceptance`；不得把浏览器截图、Agent 交互或人工验收待办写成已完成验收。

## 8. 多 Agent 协作规则

多 agent 协作时，主 agent 负责：

- 提供启动收据与 inherited context package
- 分配不重叠 owned scope
- 汇总验证与最终 closeout
- 拒绝把未知来源改动打包进同一结论

多 agent 委派的模型约束：

- 子 agent 模型不得高于直接委派者模型；同级或更低级模型才可直接委派。
- `fork_context=true` 只能继承父模型与推理强度；显式模型配置必须使用 `fork_context=false`，并完成等级比较。
- 更高模型需要用户对具体模型、范围和风险理由的明确授权；等级未知或无法验证时不得 dispatch。
- retry、sidecar、loop worker 和嵌套 worker 必须重新验证该关系，不能通过重试或嵌套路径升级模型等级。

子 agent 必须：

- 只在分配范围内工作
- 发现 authority 越界时立即上报
- 不假设其他 agent 已经处理测试、迁移、注释或回滚
- 输出可供主 agent 复核的差异、风险与验证
- 如果任务来自 PR review finding repair，不得把 `Outside diff range comments`、`Nitpick comments` 或其它
  folded latest-review findings 视为可忽略项；主 agent 建立的完整 finding inventory 仍然是子切片边界前提

## 9. Review 清单

### 9.1 单 Agent Review

- authority owner 是否确认清楚
- 是否严格留在 owned scope
- 是否存在机会主义修复
- 是否新增 TODO 泄漏、伪实现或静默 fallback
- 前端 server state 是否已优先评估 Query，且未把 URL、草稿、选择或编辑器实例放进 query cache
- 是否有证据支持 Table、Virtual、Router 或 Form 的新增 TanStack 依赖
- closeout 是否给出 summary / risk / validation / rollback
- 是否记录 verification classification，且 browser 与 human acceptance 状态没有被混作自动验证
- 如果任务来自 PR review，是否明确覆盖 `Outside diff range comments`、`Nitpick comments` 和其它 folded
  latest-review findings，而不是只处理 open threads 或高优先级子集

### 9.2 多 Agent Review

- 各 agent 的 owned scope 是否明确且不重叠
- 是否有人越界修改上游 authority 或共享契约
- 汇总结果是否遗漏冲突、风险或未验证项
- 主 agent 是否明确列出每个子切片的验证状态
- 是否把“另一位 agent 会处理”当成未完成工作的借口
- 每个委派是否包含 `parent_model`、`worker_model`、`model_relation` 和可复核的等级比较证据
- 如果协作来自 PR review finding repair，是否把 `Outside diff range comments`、`Nitpick comments` 和其它
  folded latest-review findings 继续保留在统一 disposition 清单内，而不是在拆批后丢失

## 10. 证据要求

AI 生成、修改或 review 任务的 closeout 至少记录：

```text
ai review evidence:
- authority_owner: confirmed | escalated | unknown
- scope_discipline: clean | mixed | escalated
- opportunistic_fix: none | included-with-justification
- todo_leakage: none | temporary-with-expiry
- validation: <command or reason>
- verification_class: agent-only | human-acceptance-required | browser-automation-required | mixed
- browser_status: not-needed | authorized-local | ci-contract
- human_acceptance: not-required | required | awaiting_human_acceptance
- rollback: documented | not-applicable
- multi_agent: no | yes
```

若为多 agent 任务，另补：

```text
multi-agent evidence:
- slices: <count>
- overlap: none | detected
- inherited_context: complete | incomplete
- integration_validation: done | partial | not-run
```

## 11. 适合进入 CI 的规则

适合结构化检查或 PR 模板门禁的规则：

- closeout 包含 summary / risk / validation / rollback 字段
- 禁止新增裸 `TODO` / `FIXME`
- 禁止无授权的大范围 rename、锁文件刷新或依赖升级
- 多 agent 任务必须声明 slices 与验证状态
- browser 不得被写成所有 Web 任务的默认完成门槛

更适合留在文档 / review 的规则：

- 这次跨模块重构是否真的必要
- owned scope 是否划分合理
- 回滚方案是否足够现实
- 机会主义修复是否真的与 authority 修复强相关
