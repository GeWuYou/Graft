# Graft Web Signals Theme Runtime 评估与最小 POC 设计

## 1. 背景

本文件用于收敛 `web` 是否需要在 `setting/theme` 范围内引入 `signals`。

本次约束固定如下：

* `Pinia` 保持为正式共享状态层
* 不设计全局替换 `Pinia` 的方案
* 只允许产出评估文档和最小 POC 设计
* 不新增依赖
* 不修改业务代码
* `signals` 只允许作为 `setting/theme` 内部的局部响应式内核候选
* 候选实现优先为纯运行时 `alien-signals`

## 2. 当前链路现状

当前 `theme runtime` 相关逻辑主要分布在以下位置：

* `web/src/store/modules/setting.ts`
  负责主题模式、品牌色、预设、token 覆盖、运行时初始化、样式注入与部分 DOM 属性副作用
* `web/src/layouts/components/theme-workbench/ThemeWorkbenchPanel.vue`
  负责主题工作台总面板与设置入口
* `web/src/layouts/components/theme-workbench/ThemeTokenEditor.vue`
  负责 token 草稿编辑、提交、清空与分组切换
* 若干 dashboard/detail 页面
  通过重复的 `watch(() => store.mode / brandTheme)` 驱动图表刷新

当前可确认的复杂点：

* `setting` store 同时承载持久化状态、派生计算与 DOM 副作用
* `ThemeTokenEditor.vue` 依赖多处 `watch` 与本地 `draftValues` 协调编辑态
* 页面侧存在若干对 `mode`、`brandTheme` 的重复 watch

## 3. 评估结论

当前结论为：`不进入 signals 试点`。

原因如下：

* 目前只能证明 `theme runtime` 存在一定复杂度，尚不能证明它已经进入“现有 `computed / watch / store action` 难以维护”的状态
* 已观察到的问题仍更接近“主题运行时职责尚未完全收口”，未形成必须更换局部响应式内核才能解决的证据
* 当前 `web` 子主题主线仍是 starter 壳层与真实后端 `auth + menu + permission + locale` 契约接线，不应因前端技术偏好扩展新的基础设施试点

因此，本次只固定边界和准入规则，不进入 POC。

## 4. 试点准入标准

只有未来再次评估且满足以下至少一项时，才允许把最小 POC 从“候选附录”升级为正式下一步：

* 同一主题派生链存在重复同步逻辑，且无法通过现有 `Pinia + computed + composable` 明显收敛
* store 内部派生与副作用长期耦合，已经持续影响主题能力扩展或回归控制
* 局部主题预览需要更细粒度订阅，而现有方案已经产生明确的理解成本或维护返工

以下情况一律视为证据不足：

* 只是代码偏长或存在少量 `watch`
* 只是从技术趋势、跨框架想象或未来标准化可能性出发
* 只是希望寻找比 `Pinia` “更新” 的状态管理表达

## 5. 最小 POC 候选设计

本节只作为未来候选，不代表当前批准实施。

边界固定如下：

* 候选实现仅限 `alien-signals`
* `Pinia setting store` 继续作为对外状态入口与持久化来源
* `signals` 只承接 `setting/theme` 内部的局部派生与预览计算
* 若未来存在 `useThemeRuntime()`，它只能作为 `setting/theme` 内部 adapter
* `useThemeRuntime()` 不得暴露为业务模块通用状态标准
* 不修改 `auth`、`permission`、`router`、`tabs-router`、`API cache`、表单提交状态、后端业务实体状态

候选数据流约束：

* 单一主源仍为 `Pinia`
* adapter 只从现有 `setting` 状态读取输入
* adapter 只向 theme workbench 或 theme runtime 内部提供只读派生结果与明确更新函数
* 禁止形成 `Pinia` 与 `signals` 双向互写的双状态源

## 6. 成功标准

只有同时满足以下条件，未来最小 POC 才算值得继续：

* 已证明 `theme runtime` 确有真实维护问题
* POC 能显著收口局部派生或预览逻辑，而不是引入第二套共享状态规范
* `Pinia` 的正式共享状态地位不受影响
* `useThemeRuntime()` 没有向普通业务模块扩散
* 团队能够清楚说明哪些问题不应使用 `signals`

## 7. 退出标准

任一情况成立，即停止或不进入 POC：

* 评估证据不足，无法证明现有方案难维护
* 需要新增依赖、改业务代码或改变 store 主职责才能验证
* 预期收益主要来自“未来跨框架可能复用”
* 需要把 adapter 扩展为通用状态方案才看得到收益
* 无法避免双状态源
* 需要触碰禁用范围中的任一领域

## 8. 测试范围

若未来满足准入标准并进入最小 POC，测试仅覆盖 `setting/theme` 局部范围：

* `light / dark / auto` 模式切换
* `brandTheme` 切换
* preset 切换
* token override 更新、清空、重置
* theme workbench 打开、关闭、分组切换
* runtime 初始化与持久化恢复后的主题一致性
* DOM 主题属性与样式注入结果一致性
* 现有依赖 `setting` 的布局与页面主题展示不回归

明确不纳入范围：

* 登录态
* 权限装配
* 动态路由
* `tabs-router`
* `API cache`
* 表单提交流
* 业务实体状态

## 9. 文档落点

仓库级规则落点：

* `ai-plan/design/前端架构设计.md`

当前具体评估与候选 POC 落点：

* `ai-plan/public/mvp-extension-path/subtopics/web/design/signals-theme-runtime-evaluation.md`

后续只有在重新评估并满足准入标准时，才允许基于本文件追加新的实施计划。
