# AI 任务追踪与恢复设计

## 1. 目标

这份文档定义 `ai-plan/` 的职责边界，避免把架构设计、实施路线和任务恢复状态混成一份文档。

它主要回答四个问题：

* 为什么仓库需要 `ai-plan/`
* `design`、`roadmap`、`public topic` 的职责如何划分
* 长期主题、长分支、追踪文件、轨迹文件、归档如何协作
* 什么时候应该新增 topic，什么时候应该归档 topic

---

## 2. 为什么不能只靠设计文档

仓库级设计文档可以回答架构和边界问题，但不能稳定承担这些职责：

* 记录当前长期任务推进到哪一阶段
* 记录某个主题的最新恢复点
* 记录最近一次验证和风险
* 让另一个贡献者或后续会话无需翻完整聊天历史即可继续工作

因此需要把“仓库真值”和“主题恢复状态”分离。

---

## 3. 目录职责

### 3.1 仓库级真值

`ai-plan/design/` 用于仓库级设计真值：

* 总体架构
* 插件与 DI 规则
* 前端模块规则
* 其它适用于整个仓库的长期设计

`ai-plan/roadmap/` 用于仓库级实施路线：

* MVP 阶段计划
* 跨主题的实施顺序
* 仓库级交付标准和测试清单

与 `ai-plan/` 并列的 `.ai/environment/` 用于仓库级环境真值：

* 当前机器与仓库相关的原始环境事实
* AI 启动时优先读取的环境摘要
* 工具选择、能力判断与“当前仓库是否已落地某条工具链”的辅助信号

它不属于某个 topic，也不替代 `ai-plan/` 的设计、路线或恢复职责。

根目录 `AGENTS.md` 继续负责仓库级启动治理真值，例如 startup preflight、最小 receipt、resume/restart
重验与 subagent 继承要求。`ai-plan/` 只提供恢复材料与恢复入口，不负责定义第二套 boot 链或启动闸门。

### 3.2 主题级恢复材料

`ai-plan/public/<topic>/todos/` 用于主题级跟踪：

* 当前目标
* 当前范围
* 当前恢复点
* 当前风险
* 最近验证
* 下一步

`ai-plan/public/<topic>/traces/` 用于主题级执行轨迹：

* 关键决策
* 阶段切换
* 验证里程碑
* 交接信息

当一个长期主题仍然需要一个统一恢复入口，但内部已经出现明显的 `server`、`web` 或其它边界分工时，可以在
active topic 下继续维护子主题恢复材料：

* `ai-plan/public/<topic>/subtopics/<name>/todos/`
* `ai-plan/public/<topic>/subtopics/<name>/traces/`

子主题不是新的并列 active topic，而是父主题内部的边界化恢复入口。

父主题负责：

* 跨边界目标
* 总体方向清单
* 共享风险
* 共享验证摘要
* 子主题入口指引

子主题负责：

* 单边界实现推进
* 单边界验证记录
* 单边界风险
* 单边界下一步

### 3.3 主题级专属文档

当某份设计或路线只服务于一个 topic，而不应提升为仓库真值时，放在：

* `ai-plan/public/<topic>/design/`
* `ai-plan/public/<topic>/roadmap/`

默认不要创建重复的 topic 专属设计；只有在确实存在 topic-only 规则时才新增。

---

## 4. 长期主题与长分支

长期主题表示一个需要跨多轮推进、需要稳定恢复入口的工作方向。

长期主题至少包含：

* 一个稳定 topic 名称
* 一个对应的长分支
* 一个 tracking 文件
* 一个 trace 文件

当前仓库的首个长期主题为：

* Topic: `mvp-extension-path`
* Branch: `feat/mvp-extension-path`

这个主题覆盖当前 MVP 主线，并保留为默认恢复入口。

当 `mvp-extension-path` 同时承载前后端持续迭代、但 tracking/trace 已经明显过重时，可以在该主题下引入
`server` 与 `web` 子主题，而不是把它们升级成多个并列 active topic。

---

## 5. Tracking 与 Trace 的分工

Tracking 文件是默认恢复入口，必须保持短小、可交接、可直接执行。

Tracking 文件应长期保留：

* 当前真值
* 当前阶段
* 当前风险
* 最近验证
* 立即下一步

当一个主题已经引入子主题时，父级 tracking 还应额外保留：

* 子主题清单
* 哪些事项必须留在父级
* 哪些事项应该下沉到子主题

Trace 文件记录执行轨迹，但也不能退化为无边界流水账。它应保留：

* 最近关键决策
* 最近里程碑
* 影响后续执行的上下文

当父主题下已有子主题时，父级 trace 只记录跨边界决策、共享里程碑和会影响多个子主题的上下文。

如果某一阶段已经完成且不再属于默认恢复路径，应把详细历史移入归档。

---

## 6. 归档规则

当 active topic 内部某一阶段完成后：

* 将过长的历史从 active tracking/trace 中裁剪
* 移入 `ai-plan/public/<topic>/archive/`
* 在 active 文件中只保留必要的归档指针

当整个 topic 完成后：

* 将整个 topic 目录移动到 `ai-plan/public/archive/<topic>/`
* 在 `ai-plan/public/README.md` 中移除该 topic

---

## 7. 何时新增 Topic

满足以下任一条件时，可以从仓库级真值下派生新的 active topic：

* 工作方向会跨多轮推进
* 恢复成本已经高到不能只靠聊天记录
* 需要独立风险、验证和下一步
* 该方向与现有 active topic 的边界已经明显不同

如果总体目标仍然一致，只是 `server`、`web`、插件族或某个子系统的恢复材料已经过重，优先在现有 active
topic 下增加子主题，而不是拆成多个并列 active topic。

满足以下任一条件时，优先新增子主题而不是新增 active topic：

* 父主题仍然是默认恢复入口
* 多个边界共享同一个长期分支或总体目标
* 纯边界内工作已经需要独立风险、验证和下一步
* 父级 tracking/trace 已经因为混合记录多个边界而变得冗长

不满足新增 active topic 或新增子主题条件时，优先继续挂在现有 active topic 下推进。

---

## 8. 结论

`ai-plan/` 不是单纯改名后的 `plan/`，而是把仓库设计真值、仓库实施路线、主题恢复材料正式分层。

判断这个体系是否成功的标准很简单：

* 新贡献者能快速找到仓库级真值
* 复杂主题能在不依赖聊天历史的前提下恢复
* 仓库不会同时维护多份互相冲突的计划真值

同样重要的是：恢复 topic 不等于恢复仓库治理状态。任何 resume/restart 都必须先经过根目录
`AGENTS.md` 定义的 startup preflight，再进入 `ai-plan/public/` 的恢复链。
