# AI 任务追踪与恢复设计

## 1. 目标

这份文档定义 `ai-plan/` 的职责边界，避免把架构设计、实施路线和任务恢复状态混成一份文档。

它主要回答四个问题：

- 为什么仓库需要 `ai-plan/`
- `design`、`roadmap`、`public topic` 的职责如何划分
- 长期主题、临时任务分支、可复用工作树、追踪文件、轨迹文件、归档如何协作
- 什么时候应该新增 topic，什么时候应该归档 topic
- 新的长期工作应该如何进入这套体系

---

## 2. 为什么不能只靠设计文档

仓库级设计文档可以回答架构和边界问题，但不能稳定承担这些职责：

- 记录当前长期任务推进到哪一阶段
- 记录某个主题的最新恢复点
- 记录最近一次验证和风险
- 让另一个贡献者或后续会话无需翻完整聊天历史即可继续工作

因此需要把“仓库真值”和“主题恢复状态”分离。

同样也需要把“工作进入系统的入口判断”与“进入之后的恢复材料”分离。

---

## 3. 目录职责

### 3.1 仓库级真值

`ai-plan/design/` 用于仓库级设计真值：

- 总体架构
- 模块与 DI 规则
- 前端模块规则
- 其它适用于整个仓库的长期设计

`ai-plan/roadmap/` 用于仓库级实施路线：

- MVP 阶段计划
- 跨主题的实施顺序
- 仓库级交付标准和测试清单

与 `ai-plan/` 并列的 `.ai/environment/` 用于仓库级环境真值：

- 当前机器与仓库相关的原始环境事实
- AI 启动时优先读取的环境摘要
- 工具选择、能力判断与“当前仓库是否已落地某条工具链”的辅助信号

它不属于某个 topic，也不替代 `ai-plan/` 的设计、路线或恢复职责。

根目录 `AGENTS.md` 继续负责仓库级启动治理真值，例如 startup preflight、最小 receipt、resume/restart
重验、boot 后的 multi-agent 评估、slice-end closeout/commit 决策链，以及 subagent 继承要求。

authority-first overlay 补充：

- `ai-plan/` 恢复材料可以帮助 agent 恢复 topic 状态，但不能把 `owned scope` 误写成 canonical authority
- bounded scope forbids unrelated expansion, not required authority repair
- 如果恢复材料与现行 authority-first 规则冲突，以根 `AGENTS.md` 与现行设计文档为准；旧归档措辞只作为历史记录，不具规范性

当前仓库的执行级治理分层为：

- 根 `AGENTS.md`
  - 仓库级启动治理、恢复入口、验证入口 ownership、closeout/commit 规则、subagent 规则
- `web/AGENTS.md`
  - `web` 执行真值，例如模块边界、route 注册、contract/import 规则、frontend validation
- `server/AGENTS.md`
  - `server` 执行真值，例如 module 边界、DI 约束、Go 组织规范、Ent/migration、backend validation

`ai-plan/` 只提供恢复材料与恢复入口，不负责定义第二套 boot 链、关闭流程或启动闸门，也不应再承担
`server` / `web` 的日常执行级规则清单。

对于新的长期工作，`ai-plan/` 的推荐进入顺序是：

- startup preflight
- Work Intake
- Work Contract
- contract-driven minimal bootstrap
- specialized skill execution
- closeout / archive

这里的关键约束是：

- `design`
- `roadmap`
- `topic`
- `ADR`

都不是新的长期工作的并列入口；它们是 intake 决策之后派生出的 artifact。

当仓库使用 `graft-multi-agent-loop` 时，它是同一主会话内的串行 subagent 编排模式，而不是外部 fresh-session
runner：

- 默认 loop mode 是 `topic-completion-loop`
  - 如果调用方没有显式设置 `loop_mode`，外层 main agent 必须使用 `topic-completion-loop`
  - `checkpoint-loop` 只能在调用方显式请求时使用，不能作为省略 `loop_mode` 时的回退
- 外层 main agent 负责 startup receipt、恢复入口、batch state、预算、停止条件、closeout 解析、验收与下一轮派发
- 当任务包含多个依赖切片时，外层 main agent 在首次 dispatch 前建立最佳已知任务 DAG，并持有
  `topology_revision`、节点依赖、ready frontier 和未执行节点的局部重规划 authority
- 每个实现 round 默认委派给一个 `worker` subagent，通过 `graft-multi-agent-task` 执行
- 在 `topic-completion-loop` 中，外层 main agent 必须在每轮 closeout 验收后维护 batch state：
  - 更新 `completed_batches`
  - 更新 `pending_batches`
  - 自动选择 `next_batch`
  - 在没有 terminal stop condition 时派发下一 worker
- 当 `pending_batches` 变为空时，外层 main agent 不能直接停止：
  - 必须先执行 final archive-readiness check
  - 如果验收条件全部通过，输出 `archive-ready` 并提交本轮 owned archive 或 closeout 文档
  - 如果验收条件未通过但下一批 bounded work 明确，重新生成 `pending_batches` 并继续
  - 如果验收条件未通过且没有安全可定义的下一批，进入 `blocked`
- 在 `topic-completion-loop` 中，普通 batch success 的默认行为是 `success -> update batch state -> continue`
  - 不能因为没有 `Next-session startup prompt:` 就结束 loop
- `Next-session startup prompt:` 只用于 terminal handoff：
  - `blocked`
  - `archive-ready`
  - explicit user stop
- 外层 main agent 在 active round 期间不得编辑 repo-tracked 实现文件
- 外层 main agent 做的是 bounded orchestration，不是实时 remote-control worker
- `timeout != stalled`；stalled 判定至少同时要求：
  - 已超过 soft timeout
  - 长时间无新的可见输出证据
  - worker 尚未进入 closeout
  - 发送 checkpoint request 后仍无有效响应
- 如果当前工具面没有直接的 activity 查询能力，main agent 不得把“无法观测 tool activity”伪装成“无 tool activity”；
  保守判定只能基于经过时间、可见 transcript、worker 最近一次响应，以及 checkpoint 内容
- 每个 round 默认 `checkpoint_budget=1`
- 高风险或长运行 round 可显式提升到 `2` 或 `3`，但必须写进 round budget
- checkpoint request 使用 `interrupt=true`，且只能用于健康检查：
  - 不允许改变任务目标
  - 不允许扩大 scope
  - 不允许追加新的实现需求
- checkpoint request 必须受 cooldown 约束，避免把 loop 退化为高频人工遥控
- worker 的 checkpoint 响应必须包含：
  - `current_phase`
  - `changed_files`
  - `last_validation`
  - `next_action`
  - `can_continue`
  - `estimated_remaining_minutes`
  - `eta_confidence`
  - `risks_or_blockers`
- checkpoint 响应不是 closeout，也不是 round 终态
- 当 checkpoint 响应 `can_continue=true` 时，外层 main agent 必须继续同一个 worker round，并显式恢复到等待该 worker
  最终 closeout 的状态，不能因为最近一条消息是 checkpoint 就关闭、替换或判定 round malformed
- 外层 main agent 在 post-checkpoint 阶段必须先分类：
  - `silent_timeout`
    - 走完必需的 post-checkpoint grace window 后仍无 usable final closeout
    - 没有 recent meaningful progress 证据
    - 也没有可信的继续执行信号
  - `active_but_unfinished`
    - checkpoint 或其后的可见证据表明仍有 recent meaningful progress，且 `can_continue=true`
    - 只是 final closeout 尚未返回
  - `blocked`
    - worker 明确报告 blocker、unsafe continuation、out-of-scope repair 或 `can_continue=false`
- recent meaningful progress 至少包括以下任一项：
  - explicit diagnosis
  - owned-scope file edits
  - validation output
  - 相关 `git diff` 变化
  - concrete next step 加上可见 tool activity
- 只有 `silent_timeout` 才允许在 post-checkpoint grace 之后立即关闭当前 worker；`active_but_unfinished` 必须获得一次
  bounded continuation 或 refreshed grace，而不是直接进入 `retry_once_then_blocked`
- 外层 main agent 根据 ETA 只调整下一次等待窗口：
  - `high`：等待 `estimated_remaining_minutes`，但不超过 `max_grace_window`
  - `medium`：等待 `min(estimated_remaining_minutes, default_grace_window)`
  - `low`：只等待 `short_grace_window`
- ETA 只是建议，不得突破 round 总预算
- 如果刚出现 recent owned-scope file edits、validation work 或新的相关 `git diff`，外层 main agent 不得立即关闭该
  worker；必须在当前 hard limit 内至少刷新一次 grace，再根据后续是否重新转为 silent 判断
- 如果 ETA 连续失准、无实质进展或长期无 closeout，先降低 worker reliability，再进入
  `retry_once_then_blocked`
- round closeout 缺失、畸形或自相矛盾时，使用 `retry_once_then_blocked`：
  - incomplete checkpoint 本身不是 retry 触发条件；必须先走 post-checkpoint grace handling
  - `active_but_unfinished` 不是 retry 触发条件；只有 `silent_timeout`、显式 `blocked`、post-checkpoint grace 之后仍无
    recent meaningful progress 的 malformed final closeout，或 checkpoint budget exhausted 且没有 usable health
    response，才允许进入 retry
  - 先用新的 worker subagent 重试一次
  - retry worker 必须继承 partial diff、相关 logs、validation 结果与 previous worker failure reason
  - 第二次仍失败则 fail closed 为 `blocked`
- 该模式不恢复 `run_loop.py`、`test_run_loop.py` 或 `codex exec --ephemeral` 风格的外部 fresh-session runner

当仓库使用 `graft-multi-agent-batch` 时，它是同一会话内的一次并行 batch wave，而不是主 agent 对多个 worker 的实时遥控：

- batch wave 只执行当前 DAG ready frontier：节点的所有依赖必须先成功结算；无安全并行条件时允许退化为单节点波次
- ready frontier 内的并行仍受 owned scope、authority owner、execution context 和 acceptance gate 独立性约束
- 每个波次结算后由 outer main agent 重新计算 ready 集合；只允许增补、拆分、合并或重排尚未 dispatch 的节点
- 已完成或已 dispatch 的节点是不可重写的历史事实；重规划必须记录 `topology_revision`、原因、证据、受影响节点、依赖变化、authority 影响和验证影响
- outer main agent 负责切片、派发、验收与后续验证准备，不负责替代 active worker 完成其已委派的实现切片
- 对 write-capable worker，`timeout != stalled`
  - 一次 wait window 超时不等于 stalled
  - `no visible diff yet` 不等于 stalled
  - checkpoint 前必须先区分 `no visible diff yet`、`no new visible output evidence`、`closeout not started`
- batch wave 中的 checkpoint 也是 bounded health check：
  - 不允许改变任务目标
  - 不允许扩大 scope
  - 不允许把 checkpoint 响应解释成主 agent 接管实现的许可
- 当 checkpoint 响应 `can_continue=true` 时，必须继续同一个 worker slice，并至少给一个 post-checkpoint grace
  window 等待最终 closeout
- 如果 worker closeout 缺失、畸形或自相矛盾，先用新的 worker 重试同一 bounded slice 一次
- 第二次仍失败时，该 slice 必须显式进入 `blocked` 或由 batch wave 显式停止；不能由主 agent 静默本地接手补完

### 3.2 主题级恢复材料

`ai-plan/public/<topic>/todos/` 用于主题级跟踪：

- 当前目标
- 当前范围
- 当前恢复点
- 当前风险
- 最近验证
- 下一步
- authority discovery 结论
- 当前 authority owner / downstream consumer 关系
- 是否需要 authority escalation
- 该 topic 若来自 `Work Intake`，则其 persisted `Work Contract`

`ai-plan/public/<topic>/traces/` 用于主题级执行轨迹：

- 关键决策
- 阶段切换
- 验证里程碑
- 交接信息

当一个长期主题仍然需要一个统一恢复入口，但内部已经出现明显的 `server`、`web` 或其它边界分工时，可以在
active topic 下继续维护子主题恢复材料：

- `ai-plan/public/<topic>/subtopics/<name>/todos/`
- `ai-plan/public/<topic>/subtopics/<name>/traces/`

子主题不是新的并列 active topic，而是父主题内部的边界化恢复入口。

父主题负责：

- 跨边界目标
- 总体方向清单
- 共享风险
- 共享验证摘要
- 子主题入口指引

子主题负责：

- 单边界实现推进
- 单边界验证记录
- 单边界风险
- 单边界下一步

但子主题不得把“单边界 owned scope”解释成“禁止跨端 authority repair”；若 authority 在父主题或共享契约层，必须向上升级。

### 3.3 主题级专属文档

当某份设计或路线只服务于一个 topic，而不应提升为仓库真值时，放在：

- `ai-plan/public/<topic>/design/`
- `ai-plan/public/<topic>/roadmap/`

默认不要创建重复的 topic 专属设计；只有在确实存在 topic-only 规则时才新增。

### 3.4 Work Intake 与 Work Contract

新的长期工作先进入 `Work Intake`，而不是先决定“我要写 design”或“我要建 topic”。

`Work Intake` 负责：

- 分类这是什么工作
- 判断是否属于长期工作
- 判断是否需要 `design`
- 判断是否需要 `topic`
- 判断是否需要 `roadmap`
- 判断是否需要 `ADR`
- 决定执行引擎与 dispatch skill

`Work Intake` 不负责：

- 写领域正文
- 代替 design / roadmap / ADR skill 产出内容
- 发明第二套 startup receipt
- 发明第二套 recovery truth

对于需要 active topic 的长期工作，`Work Intake` 产出的 `Work Contract` 应持久化到 topic tracking 文件，作为后续
bootstrap、loop、closeout 和归档消费的统一输入。

`Work Contract` 不是独立文件，不进入 `catalog.json`，也不要求所有请求都持久化。短任务和一次性工作可在 intake 后直接执行，
不必生成 topic 级 contract。

---

## 4. 长期主题与可复用工作树

长期主题是跨多轮推进、需要稳定恢复入口的工作方向；工作树只是 Agent 的临时执行空间。主题至少包含稳定 topic 名称、默认恢复入口、tracking 和 trace，不能把某个本地目录或任务分支当作主题身份。

这里需要区分三种对象：

- active topic：持久恢复与工作状态的唯一入口。
- task branch：一次 Agent 任务的唯一分支；完成集成后可删除，默认不写入 active-topic 映射。
- reusable worktree：编号固定的临时目录；用完回到 `main` 基线并可被下一任务复用。

`main` 是稳定基线。开发者主工作区负责集成与 review；review 是开发者负责的审查活动，不是 Agent 可执行的集成操作。merge 或 cherry-pick 默认不由 Agent 执行，但开发者在当前任务中明确授权具体集成操作后，Agent 可以执行该操作，且不因此取得最终仓库状态 authority。分支名可按当前集成需要变化。`ai-plan/public/README.md` 只映射 active topic 与恢复文件，最多记录当前任务分支作为辅助线索。

### 4.1 可复用工作树生命周期

1. Agent 使用 `graft-worktree-manager acquire <branch>` 获取最低编号的已注册空闲目录；若没有，先恢复最低编号的安全
   `main-XX` marker-only 槽位，再补齐最低缺号，最后才连续扩容。目录、Git 注册和 marker 不一致时 fail-closed，必须先
   `doctor` 并按 `repair` 处理。
2. Agent 在唯一任务分支内修改、验证和提交。`acquire` 在 Git common directory 写入本地 lease；它只记录槽位操作状态，
   不替代 topic tracking 或任务恢复真值。
3. 任务完成时，Agent 必须在工作区干净且已验证提交后通过 closeout 记录 `release-ready`。脏工作树只能表示进行中或阻塞，
   不能被声明为已完成。
4. 开发者以两阶段方式调用 `release`：先不带确认引用运行预览，只输出 task branch、提交与 diff 统计且不改变任何
   worktree、branch 或 lease；确认集成引用包含 task branch 后，再以该引用运行确认阶段。确认阶段刷新到当前
   `origin/main`，恢复对应的 `main-XX` 标记分支，并清理本地 task branch 和 lease。历史无 lease 分支在 clean 与
   集成证明满足时可一次性回收，并显式报告 `legacy-untracked`。

对于历史上以 detached HEAD 保存的干净槽位，开发者可使用 `reconcile --confirm <slot> ...` 进行一次性转换。该命令只处理
明确指定且可证明属于旧 main 基线的槽位，不接管脏工作树、任务分支或包含未合并提交的 detached 工作树。

常规生命周期不频繁执行 `git worktree remove/add`。仓库物理迁移是一次性、开发者批准的操作，必须先确保所有工作树干净且提交已推送。

### 4.2 shared local resource 规则

创建或复用临时 worktree 时，本地共享资源必须只有一套仓库真值：

- worktree 初始化入口应统一走仓库 skill / helper，而不是每个贡献者维护一份私有脚本
- 不要硬编码机器专属 `ROOT_DIR`、`REPO_DIR`、`WORKTREE_ROOT`
- 共享本地资源清单应收口在仓库根一个 tracked manifest，并使用相对路径描述 source / target
- `server/.env`、`web/.env.development`、`.run`、`.idea` 这类本地资源应优先通过相对 symlink 复用 canonical
  repository root 中的一份本地文件，而不是在每个 worktree 里复制
- optional 本地文件缺失时应显式告警，但不应因为缺少个人环境文件就阻断 worktree 创建
- `.local` 之类只在某台机器上临时放过脚本的目录，不得继续作为仓库级 worktree 共享约定或第二真值

---

## 5. Tracking 与 Trace 的分工

Tracking 文件是默认恢复入口，必须保持短小、可交接、可直接执行。

Tracking 文件应长期保留：

- 当前真值
- 当前阶段
- 当前风险
- 最近验证
- 立即下一步
- authority source / authority owner
- 当前 slice 是 authority repair 还是 downstream consumer 修正
- 若采用 compatibility，记录为何不能直接修 authority

当一个主题已经引入子主题时，父级 tracking 还应额外保留：

- 子主题清单
- 哪些事项必须留在父级
- 哪些事项应该下沉到子主题
- 哪些方向仍需要开发者集成处理共享热点
- 当前任务是否依赖某个临时 Agent worktree

Trace 文件记录执行轨迹，但也不能退化为无边界流水账。它应保留：

- 最近关键决策
- 最近里程碑
- 影响后续执行的上下文
- authority escalation 决策与 compatibility 例外决策

当父主题下已有子主题时，父级 trace 只记录跨边界决策、共享里程碑和会影响多个子主题的上下文。

如果某一阶段已经完成且不再属于默认恢复路径，应把详细历史移入归档。

---

## 6. 归档规则

当 active topic 内部某一阶段完成后：

- 将过长的历史从 active tracking/trace 中裁剪
- 移入 `ai-plan/public/<topic>/archive/`
- 在 active 文件中只保留必要的归档指针

当整个 topic 完成后：

- 将整个 topic 目录移动到 `ai-plan/public/archive/<topic>/`
- 在 `ai-plan/public/README.md` 中移除该 topic

---

## 7. 何时新增 Topic

满足以下任一条件时，可以从仓库级真值下派生新的 active topic：

- 工作方向会跨多轮推进
- 恢复成本已经高到不能只靠聊天记录
- 需要独立风险、验证和下一步
- 该方向与现有 active topic 的边界已经明显不同

长期主题的准入取决于恢复与协调成本，而不是是否拥有独立工作树。若某方向会持续跨多轮推进、拥有独立风险和验证责任，就应建立 tracking / trace；它使用哪个编号 worktree 由每次 acquire 决定。

如果总体目标仍然一致，只是 `server`、`web`、模块族或某个子系统的恢复材料已经过重，优先在现有 active
topic 下增加子主题，而不是拆成多个并列 active topic。

满足以下任一条件时，优先新增子主题而不是新增 active topic：

- 父主题仍然是默认恢复入口
- 多个边界共享同一个总体目标
- 纯边界内工作已经需要独立风险、验证和下一步
- 父级 tracking/trace 已经因为混合记录多个边界而变得冗长

不满足新增 active topic 或新增子主题条件时，优先继续挂在现有 active topic 下推进。

如果旧 active topic 已经完成并且仓库正在收口多 worktree 共享治理，则应优先：

- 归档旧 active topic
- 建立新的治理型 active topic
- 只把仍需共享基线治理的事项留在该 topic，其余长期实现方向下沉到父 topic 子主题或独立 topic

---

## 8. 结论

`ai-plan/` 不是单纯改名后的 `plan/`，而是把仓库设计真值、仓库实施路线、主题恢复材料正式分层。

判断这个体系是否成功的标准很简单：

- 新贡献者能快速找到仓库级真值
- 复杂主题能在不依赖聊天历史的前提下恢复
- 仓库不会同时维护多份互相冲突的计划真值

同样重要的是：恢复 topic 不等于恢复仓库治理状态。任何 resume/restart 都必须先经过根目录
`AGENTS.md` 定义的 startup preflight，再进入 `ai-plan/public/` 的恢复链。
