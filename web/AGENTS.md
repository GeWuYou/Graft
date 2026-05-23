# web/AGENTS.md

本文件是 `web` 前端工作的执行真值，覆盖 `web` 的实现边界、目录所有权、契约落点、UI 规范与完成态校验。

仓库级启动、恢复、提交与跨仓治理仍以根 `AGENTS.md` 为准；本文件不重复定义第二套启动或提交流程。

## 1. 适用范围

适用范围：

- `web/src/**`
- `web/package.json`、前端校验脚本与前端样式/测试配置
- 前端模块脚手架、页面接入、共享资产抽取与前端契约治理

前端任务改动前，至少读取这些文档：

- `../AGENTS.md`
- `../DESIGN.md`
- `../ai-plan/design/前端架构设计.md`
- `../ai-plan/design/前端视觉设计规范.md`
- `../ai-plan/design/TDesign-MCP-辅助开发规范.md`
- `../ai-plan/design/契约治理与魔法值治理规范.md`
  - 当任务涉及路由名、路径、权限码、存储键、请求头、认证方案、错误码、稳定状态枚举或跨模块 typed contract 时必须读取

如代码与文档分叉，先更新文档或在同一改动中一起更新。

## 2. 固定技术选择

`web` 的固定技术栈是：

- `Vue 3`
- `TypeScript`
- `Vite`
- `TDesign Vue Next`
- `Pinia`
- `Vue Router`
- `Axios`
- `UnoCSS`

禁止在未先更新设计文档的情况下：

- 切换到 React、Naive UI 或其他主 UI 体系
- 把 `UnoCSS` 升级为整套视觉体系重写工具
- 引入与 `TDesign Vue Next` 平行的第二套后台 UI 运行基线

## 3. 目录真值

`web/src` 的长期运行面固定为：

```text
web/src/
├─ app/
├─ layouts/
├─ modules/
├─ shared/
├─ contracts/
├─ config/
├─ locales/
├─ router/
├─ store/
├─ api/
├─ style/
├─ assets/
├─ types/
└─ utils/
```

目录职责冻结如下：

- `app/`
  - 壳层页面、异常页、认证页与应用装配入口
  - 不承载业务模块长期实现真值
- `layouts/`
  - 后台壳布局、导航、面包屑、Footer、安全留白和壳层组件
- `modules/<name>/`
  - 某个业务模块的唯一长期真值
  - 默认目录为 `pages`、`components`、`api`、`contract`、`types`、`locales`
- `shared/`
  - 跨模块复用且无业务语义的组件、composables、helpers、样式片段
- `contracts/`
  - 平台级前端稳定契约
  - 不放模块私有契约
- `config/`
  - 壳层级平台配置真值
  - 仅承载主题、样式、全局 UI 或应用装配所需的配置入口
  - 不承载模块业务配置真值
- `locales/`
  - 应用级 locale 状态、消息目录、查找入口与回退策略
- `router/`
  - 静态路由与动态路由装配
- `store/`
  - 跨页面共享状态
  - 不收纳页面局部表单状态
- `api/`
  - 平台级 request/auth/session adapter
  - 不放模块业务 API 真值
- `style/`
  - 全局样式与壳层样式基线
- `types/`
  - 平台级类型边界
  - 不放模块私有类型真值
- `utils/`
  - 平台级工具、路由装配工具、日志与请求基础设施
  - 不作为模块实现溢出的默认落点

冻结规则：

- `web/src/modules/<name>/**` 是业务模块页面、API、模块私有契约、局部类型、模块消息源与模块注册面的唯一长期真值
- `web/src/shared/**` 是唯一允许的跨模块业务无关复用层
- 永远不要重新引入根级 `web/src/pages/**` 作为运行面
- 永远不要重新引入根级模块专属 `web/src/api/**`、`web/src/api/model/**` 或 `web/src/contracts/<module>/**`
- 根级 `components/`、`hooks/`、业务专属 `utils/` 不是最终所有权层；如有新增同类目录，必须先证明它属于平台级基础设施

## 4. 模块边界与导入规则

所有权边界：

- `shell-owned`
  - `app/**`
  - `layouts/**`
  - `router/**`
  - `locales/**`
  - 平台级 `contracts/**`
  - `config/**`
  - 平台级 `api/**`
  - 平台级 `types/**`
  - 平台级 `utils/**`
  - 全局 `store/**`
- `module-owned`
  - `modules/<name>/**`
- `shared-owned`
  - `shared/**`

导入约束：

- `app/**`、`layouts/**`、`router/**`、`store/**` 只能消费：
  - `shared/**`
  - `config/**`
  - 平台级 `contracts/**`、`api/**`、`types/**`、`utils/**`
  - 模块显式对外暴露的注册面或稳定契约
- `app/**`、`layouts/**`、`router/**` 不得直接导入其他模块的：
  - `pages/**`
  - `components/**`
  - `api/**`
  - `types/**`
  - `locales/**`
- `modules/**` 不得反向导入 `app/**`
- 模块之间允许跨边界消费的长期真值只有另一模块的 `contract/**`
- `modules/<name>/types/**` 一律视为模块私有实现细节
- `modules/<name>/shared/**` 允许承载仅在该模块内部被多个页面或组件复用的 helper、样式片段与 composable；它不是跨模块 `shared/**` 的替代品
- 禁止新增跨模块 `@/modules/<other>/types/**` 导入
- 如果某个 DTO、字面量联合、权限码、路径、表格查询形状或 capability type 需要被其他模块、壳层或平台基础设施稳定消费，必须提升到 `modules/<name>/contract/**`
- 根级 `contracts/**` 只收口平台级稳定契约；不要把模块契约提升到根级来逃避模块边界

共享提升规则：

- 只有同时满足“被多个模块或壳层复用”且“无业务语义”时，资产才允许提升到 `shared/**`
- 带有 `user`、`rbac`、`plugin`、权限码、路由名、DTO、API path、模块文案语义的资产，不得进入 `shared/**`
- 业务相关但需要跨模块稳定复用时，继续由所属模块持有，并通过 `contract/**` 暴露
- `shared/**` 不是临时存放区；无法说明复用边界和无业务语义时，不得放入

## 5. 路由、模块注册与 i18n

路由与注册面规则：

- `router/**` 只拥有静态路由和动态装配逻辑，不拥有模块页面真值
- 模块接入壳层的唯一新功能入口是 `modules/<name>/index.ts`
- 模块对外声明 bootstrap 动态路由的唯一入口是 `modules/<name>/bootstrap-routes.ts`
- `bootstrap-routes.ts` 只声明模块可接入壳层所需的最小注册信息，不扩散页面实现细节到壳层
- 壳层只消费模块注册结果，不直接维护“页面 path -> 模块页面组件”的第二套长期白名单
- 新增模块时，默认按 `pages`、`components`、`api`、`contract`、`types`、`locales` 组织
- 模块存在稳定子菜单或同级子页面时，`pages/<subpage>/index.vue` 是默认页面真值；测试与页面私有样式跟随该目录放在 `pages/<subpage>/index.test.ts`、`pages/<subpage>/index.less`
- 模块页面之间共享但不应暴露到跨模块 `shared/**` 的 helper、样式基线与 snapshot/composable，默认放在 `modules/<name>/shared/**`
- 新增模块时，至少补齐模块目录、`index.ts` 与 `bootstrap-routes.ts`
- 如果某个默认目录在当前切片暂时不存在，必须保证对应真值仍留在模块边界内，不得回退到根级平台目录
- 路由名必须稳定且唯一；不得为同一语义并行维护多套 route name 或 path 常量

i18n 与标题规则：

- `title_key` 是前端菜单与动态路由标题的唯一长期真值
- 上游返回的 `title` 只允许在 adapter 或 bootstrap 装配边界作为外部输入回退
- 模块和壳层内部不得长期并行维护 `title_key + 自己的 title 文案` 两套真值
- 新增消息 key 必须带边界前缀，能从 key 看出归属边界
- 建议前缀：
  - `app.*`
  - `layout.*`
  - `menu.*`
  - `<module>.*`
- 模块私有消息源优先放在 `modules/<name>/locales/**`
- 应用级 locale 状态、回退语言、持久化策略与消息查找入口继续收口在 `locales/**`

契约落点规则：

- 平台级路由名、特殊路径、存储键、请求头、认证方案、平台错误码等稳定契约放在根级 `contracts/**`
- 模块级权限码、API path、跨模块 DTO、模块稳定状态值、模块消息 key 常量等稳定契约放在 `modules/<name>/contract/**`
- 模块私有 `types/**` 不得充当跨模块 contract
- 不得通过 alias、根级 re-export 或兼容副本维持第二套长期契约真值

壳层 Footer 约定：

- 全局布局默认提供统一 Footer 与底部安全留白
- 页面级 footer 元信息应保留在路由 meta 的壳层边界内
- 若某个页面需要禁用 Footer 或替换 footer 文案，应通过 route meta 的显式配置完成，不要在单页重复实现底部布局

## 6. UI 与 TDesign

UI 约束：

- `TDesign Vue Next` 是唯一主 UI 体系
- `UnoCSS` 只用于辅助布局和少量原子样式
- 生成或修改页面前应先读取根 `DESIGN.md`，再查 TDesign MCP 或官方文档
- 不得随意覆盖 TDesign 内部 DOM；涉及组件 DOM、插槽、事件、props、升级影响时，先查 TDesign MCP 或官方文档
- AI 生成或修改 `web` 代码时，默认按 `vue-next` 组件资料执行，不凭经验猜测组件 API
- 新页面优先复用既有后台模式：页头、筛选区、表格、抽屉、弹窗、状态标签、操作列
- `web/ai-libs/**` 只是 starter/demo 参考源，不是运行时依赖，也不是第二个前端真值
- `ai-plan/design/graft-design-system/**` 是 Graft 风格参考模板目录，只作为设计参考和 AI 生成约束，不是运行时依赖

页面类型与 vibe coding 规则：

- 每个前端需求都必须先声明页面类型
- 首阶段内置 4 类基础页面母版：
  - `shell`
  - `auth`
  - `overview-dashboard`
  - `list-form-detail`
- 这 4 类只覆盖当前 Graft 高频后台页面，不是页面类型全集
- 若需求无法自然归入上述 4 类，必须先登记为扩展页面类型，并补充：
  - 信息层级
  - 组件组合
  - 状态集合
  - 主题响应要求
  - i18n 要求
  - 验收规则
- 新增页面、重构页面、复杂布局页面：
  - 必须先输出结构方案，再进入编码
  - 结构方案至少包含页面类型、`page header`、`primary action area`、`main content surface`、`feedback surface`、主题依赖与 i18n 边界
- 简单文案、样式、小交互修复：
  - 可以直接实现
  - 但仍必须通过页面类型、i18n、主题和可见文案自检

交互排障规则：

- 前端交互异常、路由跳转异常、菜单展开/选中异常、图表或布局只在特定操作序列下失效时，默认先做最小化诊断，再决定实现修复
- 默认优先使用结构化控制台日志、路由守卫日志、事件链路日志或最小可复现测试来确认真实运行路径，不要只凭静态阅读代码猜测交互行为
- 诊断日志应放在实际交互边界，例如 `layouts/**`、`router/**`、页面事件处理器、图表同步点或 store action，而不是分散打印噪音日志
- 临时诊断日志必须遵守 `ai-plan/design/前端架构设计.md` 的日志治理约束；问题确认后，应在提交前删除、降级或收口到明确开关下
- 当用户能够提供浏览器控制台、录屏、复现步骤或截图时，优先结合这些运行时证据收敛问题，再决定是否修改路由、状态或布局实现

可见文案治理规则：

- 文案禁词治理只作用于用户可见 UI 文案、菜单、按钮、空态、帮助提示与页面说明
- 文案禁词治理不作用于：
  - `ai-plan/**`
  - `AGENTS.md`
  - 代码注释
  - 测试名称
  - 开发文档
- 用户可见文案不得泄露：
  - AI 调试文本
  - starter/demo 迁移说明
  - 实现阶段说明
  - 仅面向开发者的契约治理术语
- 用户可见文案默认应偏向操作语义，而不是实现语义

页面骨架规则：

- 关键页至少覆盖 `page header`、`primary action area`、`main content surface`、`feedback surface` 的存在性和结构稳定性
- 不同页面类型可按母版裁剪
- 不强制所有页面都出现 `table`、`card`、`detail` 三件套
- 不得为了“概览感”把后台页面做成营销页 hero
- table/list management 页面空态必须使用 `t-empty` 或 table empty slot；禁止在 table body 里实现自定义小灰卡片空态
- table/list management 页面空态必须保留 header/body/footer 结构，保持分页稳定，并使用主题 token 而不是硬编码颜色

推荐技能：

- 处理 `web` 页面、布局、文案、主题、页面母版或前端 AI 提示词任务时，优先使用仓库技能 `graft-web-vibe-coding`

主题与图表规则：

- 页面、模块与样式优先消费现有主题 token，例如 `--td-*`、`settingStore.chartColors` 与现有 brand theme 解析结果；不要把只适配单一明暗模式的十六进制颜色硬编码进业务页面
- 当页面引入 ECharts 或其它图表时：
  - tooltip、legend、axis、splitLine、series 主色与容器边框都必须响应当前 color mode 和 brand theme
  - 图表颜色若需要回退值，只能作为 token 缺失时的最终兜底，不能成为运行时主真值
  - 需要在 `mode`、brand theme、locale 或容器尺寸变化后重新同步图表
- 使用 CSS 渐变、`color-mix` 或自定义背景时，必须同时验证浅色和深色模式下的可读性、边框对比度和状态语义，不得制造仅在一种模式下可读的卡片或图表面板
- 模块内状态色应优先映射到 TDesign 语义 token，如 success / warning / error / placeholder，对应健康、降级、异常、未启用等状态，不要私造第二套长期状态色规范
- 若某个页面需要依赖主题 token 才能正确渲染，相关 Vitest 或最小直接验证应至少覆盖一次图表/主题同步路径，而不是只验证纯文案渲染

## 7. 验证与工具链

前端完成态的强制校验入口是：

```bash
bun run check
```

执行顺序固定为：

1. `format:check`
2. `typecheck`
3. `lint`
4. `stylelint`
5. `test:run`
6. `build`

执行规则：

- 功能完成、任务完成、准备合并时，必须跑完整 `bun run check`
- 中间迭代可先跑最小直接验证，但不能把局部验证当作完成态
- 默认完成态要求 `typecheck`、`lint`、`stylelint`、`test:run`、`build` 全部零 warning
- 前端治理测试应至少覆盖：
  - 用户可见文案禁词范围
  - 关键 `title_key` 解析
  - 主题响应路径或对应最小直接验证
- `Vitest` 是正式前端测试基线，不把“前端没有测试”当作默认前提
- `Stylelint` 用于约束样式覆盖边界，避免随意改写 TDesign 结构
- 不允许用大面积 `as any`、`any` 或关闭 strict 的方式绕过类型问题；必须把不安全边界收口到 adapter、client、schema 或迁移兼容层

Bun 工具链规则：

- web 的安装、开发、校验、构建、预览默认都通过仓库当前环境中的 bun 执行
- 不得混用多套 Bun 或其他包管理器刷新 web/node_modules
- 如前端工具链规则发生变化，必须同步更新 .ai/environment/tools.ai.yaml

## 8. 禁止事项

禁止新增或恢复以下做法：

- 根级 `web/src/pages/**` 运行面
- 模块专属根级 `api/model/contracts` 兼容桥
- 跨模块导入他人模块的 `types/**`
- 壳层直接持有模块页面真值
- 未经文档依据的 TDesign DOM 猜测式样式覆盖
- 把业务语义资产塞进 `shared/**`
- 把平台级基础设施塞进模块目录，或把模块真值塞回根级平台目录
