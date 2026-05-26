# AI 任务追踪与恢复设计

## 1. 目标

这份文档定义 `ai-plan/` 的职责边界，避免把架构设计、实施路线和任务恢复状态混成一份文档。

它主要回答四个问题：

- 为什么仓库需要 `ai-plan/`
- `design`、`roadmap`、`public topic` 的职责如何划分
- 长期主题、长期工作树、长分支、追踪文件、轨迹文件、归档如何协作
- 什么时候应该新增 topic，什么时候应该归档 topic

---

## 2. 为什么不能只靠设计文档

仓库级设计文档可以回答架构和边界问题，但不能稳定承担这些职责：

- 记录当前长期任务推进到哪一阶段
- 记录某个主题的最新恢复点
- 记录最近一次验证和风险
- 让另一个贡献者或后续会话无需翻完整聊天历史即可继续工作

因此需要把“仓库真值”和“主题恢复状态”分离。

---

## 3. 目录职责

### 3.1 仓库级真值

`ai-plan/design/` 用于仓库级设计真值：

- 总体架构
- 插件与 DI 规则
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

当前仓库的执行级治理分层为：

- 根 `AGENTS.md`
  - 仓库级启动治理、恢复入口、验证入口 ownership、closeout/commit 规则、subagent 规则
- `web/AGENTS.md`
  - `web` 执行真值，例如模块边界、route 注册、contract/import 规则、frontend validation
- `server/AGENTS.md`
  - `server` 执行真值，例如 plugin 边界、DI 约束、Go 组织规范、Ent/migration、backend validation

`ai-plan/` 只提供恢复材料与恢复入口，不负责定义第二套 boot 链、关闭流程或启动闸门，也不应再承担
`server` / `web` 的日常执行级规则清单。

当仓库使用 `graft-multi-agent-loop` 时，它是同一主会话内的串行 subagent 编排模式，而不是外部 fresh-session
runner：

- 外层 main agent 负责 startup receipt、恢复入口、预算、停止条件、closeout 解析、验收与下一轮派发
- 每个实现 round 默认委派给一个 `worker` subagent，通过 `graft-multi-agent-task` 执行
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
- 外层 main agent 根据 ETA 只调整下一次等待窗口：
  - `high`：等待 `estimated_remaining_minutes`，但不超过 `max_grace_window`
  - `medium`：等待 `min(estimated_remaining_minutes, default_grace_window)`
  - `low`：只等待 `short_grace_window`
- ETA 只是建议，不得突破 round 总预算
- 如果 ETA 连续失准、无实质进展或长期无 closeout，先降低 worker reliability，再进入
  `retry_once_then_blocked`
- round closeout 缺失、畸形或自相矛盾时，使用 `retry_once_then_blocked`：
  - incomplete checkpoint 本身不是 retry 触发条件；必须先走 post-checkpoint grace handling
  - 先用新的 worker subagent 重试一次
  - retry worker 必须继承 partial diff、相关 logs、validation 结果与 previous worker failure reason
  - 第二次仍失败则 fail closed 为 `blocked`
- 该模式不恢复 `run_loop.py`、`test_run_loop.py` 或 `codex exec --ephemeral` 风格的外部 fresh-session runner

### 3.2 主题级恢复材料

`ai-plan/public/<topic>/todos/` 用于主题级跟踪：

- 当前目标
- 当前范围
- 当前恢复点
- 当前风险
- 最近验证
- 下一步

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

### 3.3 主题级专属文档

当某份设计或路线只服务于一个 topic，而不应提升为仓库真值时，放在：

- `ai-plan/public/<topic>/design/`
- `ai-plan/public/<topic>/roadmap/`

默认不要创建重复的 topic 专属设计；只有在确实存在 topic-only 规则时才新增。

---

## 4. 长期主题、长期工作树与长分支

长期主题表示一个需要跨多轮推进、需要稳定恢复入口的工作方向。

长期主题至少包含：

- 一个稳定 topic 名称
- 一个默认恢复入口
- 一个 tracking 文件
- 一个 trace 文件

当仓库仍由单一主线推进时，这个默认恢复入口通常只需要绑定到一个长分支。

当仓库进入并行推进阶段时，长期主题可以绑定到一个长期保留的本地 worktree，而不是只绑定到一个仍然存在的远程分支。

这里需要区分三种对象：

- 短分支
  - 提交、hotfix、fixbug 或短期验证所用
  - 提交完成后可以删除
  - 默认不进入 `ai-plan/public/README.md` 的 active-topic 映射
- 长分支
  - 某个长期主题当前使用的分支名
  - 可以作为 topic 的恢复线索之一，但不应假定它永远存在于远程
- 长期 worktree
  - 为某个长期主题长期保留的本地工作目录
  - 即使远程分支删除，本地 worktree 仍可继续作为该 topic 的默认恢复入口

因此，`ai-plan/public/README.md` 的公开映射应优先表达：

- worktree 名称（如果已经存在）
- 当前分支名
- 对应 active topic

如果某个长期主题尚未真正创建独立 worktree，则可以暂时只记录分支映射，并在 tracking 中明确“当前仍未拆出独立长期
worktree”的状态。

仓库早期的首个长期主题为：

- Topic: `mvp-extension-path`
- Branch: `feat/mvp-extension-path`

这个主题覆盖了早期 MVP 主线，并在当时作为默认恢复入口。

当该主线完成并被并回 `main` 后，应将其整体移入 `ai-plan/public/archive/`，而不是继续把已经完成的长期主题保留在
active topic 列表中伪装成默认恢复入口。

如果仓库接下来需要先在 `main` 上治理共享基线，以便后续再从本地分支拉出多个长期 worktree，可以新增一个以 `main`
为默认恢复入口的治理型 active topic。这个 topic 的职责应收敛为：

- worktree/topic 映射治理
- 共享热点 ownership 治理
- 归档旧 topic
- 为后续独立长期 worktree 准备新的 active topics

此类治理型 topic 不替代未来真正按 worktree 拆出的业务 active topics；它只是多 worktree 切分前的主分支准备阶段。

当 `mvp-extension-path` 同时承载前后端持续迭代、但 tracking/trace 已经明显过重时，可以在该主题下引入
`server` 与 `web` 子主题，而不是把它们升级成多个并列 active topic。

### 4.1 `main` 共享基线阶段

在真正拆出多个长期 worktree/topic pair 之前，`main` 可以短期承担“共享基线 worktree”的角色，但它不是永久默认入口。

这个阶段的 `main` topic 应只承载：

- 共享热点 ownership 收口
- 长期 worktree 候选边界梳理
- active topic / subtopic 与 worktree 映射治理
- 已完成旧 topic 的归档和恢复入口切换

不应继续把下列内容长期堆在 `main` topic 里：

- 某个插件的日常 feature backlog
- 本可独立 owned 的长期实现细节
- 反复混写多个插件的验证记录和下一步

如果某个方向仍需频繁修改共享热点、仍未形成稳定 owned scope，说明它还停留在共享基线治理阶段，不应过早登记为 dedicated
long-lived worktree/topic pair。

### 4.2 dedicated long-lived worktree/topic pair 的准入条件

一个方向要从 `main` 共享基线切换成独立长期 worktree/topic pair，至少同时满足：

- 有清晰且可长期维持的 owned scope
- 共享热点白名单已明确，而不是把所有白名单都默认算可写范围
- 已知的 cross-worktree 依赖可以收敛到共享稳定边界或短生命周期集成点
- 该方向值得拥有独立 tracking / trace，恢复时不再依赖父 topic 的混合上下文
- 该方向的验证责任已经清楚，能够独立报告最近验证与剩余风险

若这些条件不满足，优先继续留在 `main` 治理型 topic 或父 topic 的子主题下推进。

### 4.3 从根分支切换到 dedicated pair 的步骤

从 root branch / `main` 切换到 dedicated long-lived worktree/topic pair 时，治理顺序应固定为：

1. 在 `main` 治理 topic 中先确认该方向的 owned scope、共享热点白名单和当前 branch/worktree 候选名
2. 为该方向建立独立 worktree 与长期分支，并同步建立对应 tracking / trace 恢复入口
3. 在 active-topic 映射中登记新的 topic 与 worktree 关系，让后续 startup preflight 可以得到明确恢复入口
4. 将该方向的日常下一步、验证、风险迁移到新 topic 或子主题
5. 把 `main` 治理 topic 收缩回共享基线职责，不再继续承接该方向的常规实现推进

如果切换后发现该方向仍反复争抢共享热点，应撤回为共享基线治理问题处理，而不是放任 dedicated pair 名义存在但实际持续混写。

### 4.4 shared local resource 规则

创建新的 dedicated 或临时 worktree 时，本地共享资源必须只有一套仓库真值：

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

当一个主题已经引入子主题时，父级 tracking 还应额外保留：

- 子主题清单
- 哪些事项必须留在父级
- 哪些事项应该下沉到子主题
- 哪些方向仍留在 `main` 共享基线阶段
- 哪些方向已经拥有 dedicated long-lived worktree/topic pair

Trace 文件记录执行轨迹，但也不能退化为无边界流水账。它应保留：

- 最近关键决策
- 最近里程碑
- 影响后续执行的上下文

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

如果未来准备让“一个长期 worktree 对应一个长期 topic”，还应同时满足：

- 该方向会持续跨多轮推进，而不是一次性切片
- 该方向有足够稳定的 owned scope，可避免长期反复争抢共享热点
- 该方向值得拥有独立 tracking / trace，而不是继续挂在父主题下等待整合
- 该方向的共享热点白名单和切换后的验证责任已经写清

如果总体目标仍然一致，只是 `server`、`web`、插件族或某个子系统的恢复材料已经过重，优先在现有 active
topic 下增加子主题，而不是拆成多个并列 active topic。

满足以下任一条件时，优先新增子主题而不是新增 active topic：

- 父主题仍然是默认恢复入口
- 多个边界共享同一个长期分支或总体目标
- 纯边界内工作已经需要独立风险、验证和下一步
- 父级 tracking/trace 已经因为混合记录多个边界而变得冗长

不满足新增 active topic 或新增子主题条件时，优先继续挂在现有 active topic 下推进。

如果旧 active topic 已经完成并且仓库正在 `main` 上为多 worktree 做共享治理准备，则应优先：

- 归档旧 active topic
- 在 `main` 上建立新的治理型 active topic
- 等新的长期 worktree 真正创建后，再把它们登记为独立 active topics
- 只把仍需共享基线治理的事项留在 `main` topic，其余长期实现方向尽快下沉到 dedicated pair 或父 topic 子主题

---

## 8. 结论

`ai-plan/` 不是单纯改名后的 `plan/`，而是把仓库设计真值、仓库实施路线、主题恢复材料正式分层。

判断这个体系是否成功的标准很简单：

- 新贡献者能快速找到仓库级真值
- 复杂主题能在不依赖聊天历史的前提下恢复
- 仓库不会同时维护多份互相冲突的计划真值

同样重要的是：恢复 topic 不等于恢复仓库治理状态。任何 resume/restart 都必须先经过根目录
`AGENTS.md` 定义的 startup preflight，再进入 `ai-plan/public/` 的恢复链。
