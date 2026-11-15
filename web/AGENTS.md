# web/AGENTS.md

本文件拥有 web 的实现边界、UI 规则与完成验证。启动、授权、协作、提交按根 [AGENTS.md](../AGENTS.md)，
复用本轮回执与有效证据。shared contract/menu/route authority 漂移先修上游，不以 UI mapping/alias 掩盖。
技术栈固定 Vue 3/TypeScript/Vite/TDesign Vue Next/Pinia/Vue Router/Axios/UnoCSS，Bun 统一安装与验证。

## 目录与契约

- `modules/<name>/**` 是业务页面/components/api/contract/types/locales/registration 唯一长期 owner。
  auth 页面真相归 modules/auth，app/auth 仅装配。禁止恢复根 src/pages 或模块专属根 api/model/contracts 桥。
- app/layouts/router 为 shell，根 config/contracts/api/types/utils/store/locales/style 仅承载平台语义。
  shell 只消费模块显式注册面或 contract，不直接导入 pages/components/api/types/locales 内部实现。
  module 不反向导入 app；跨模块长期接口只通过对方 contract，禁止引用 types 私有实现。
- shared 是跨模块复用且无业务语义的唯一公共层；业务 DTO/权限/route/API path/消息继续由模块 contract 持有。
  模块内 shared 只服务本模块。模块契约不得为避开边界提升到根 contracts；复用资产先 graft-shared-asset-reuse。
- 菜单/permission/route/bootstrap 变更按根跨边界规则查 server descriptor/OpenAPI 权威。新增模块明确
  menu + route + page + API + permission；不由 shell 私自拥有模块页面语义。
- server state 与 UI/client state 分开。API 缓存/加载/失效/重复请求优先评估当前 TanStack Query；URL、草稿、
  选择、编辑器实例不入 query cache。不能用手工刷新层复制已存在资源能力。

## 按风险读取

小文案、局部样式、已定位实现修复只读直接规则，不要求全量设计、页面方案或新 topic。

- **新页面/页面重构/复杂布局**：[DESIGN](../DESIGN.md)、[前端视觉](../ai-plan/design/governance/frontend/前端视觉设计规范.md)、graft-web-vibe-coding
- **模块/状态/路由/公共接口**：[前端架构](../ai-plan/design/architecture/前端架构设计.md)、graft-web-module-scaffold、对应根语义审查
- **新增或不确定 TDesign API/DOM/版本**：[TDesign MCP](../ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md)
- **UI 滚动/弹窗/图表/调试或检查配置**：[前端实现细则](../ai-plan/design/governance/frontend/前端实现与检查细则.md)
- **共享契约；配置**：[契约](../ai-plan/design/governance/platform/契约治理与魔法值治理规范.md)；[配置](../ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md)
- **i18n/locale/消息源**：graft-localization-governance；[前端架构](../ai-plan/design/architecture/前端架构设计.md)
- **注释/组件文档**：[注释规范](../ai-plan/design/governance/ai/代码注释与模块文档规范.md)、graft-comment-governance

## 页面与组件

- 新增/重构/复杂布局先确定页面类型 shell/auth/overview-dashboard/list-form-detail；不适合时登记扩展类型的信息
  层级、组合、状态、主题/i18n 与验收。方案明确 header、primary action、main content、feedback；可按类型裁剪，
  不强制 table/card/detail 三件套。小修复仅检查已有页面语义、可见文案、主题和 i18n。
- TDesign 按需加载：Vite resolver 解析组件子入口；不得 app.use(TDesign) 或引入 es/style/index.css 全量样式。
  程序式插件/渲染函数显式 import，必要时组件级 style import。UnoCSS 仅辅助布局，不重建 UI 体系。
- 新增或不确定组件 API 查询 vue-next docs；不确定名称查 list；依赖内部 DOM 查 dom；版本变化查 changelog。
  当前任务已查且版本/用法未变的证据可复用；简单文案/token 修复无须重查。优先 MCP，不可用时用官方文档并说明。
- 外部 UI 示例/plugin 只提供 guidance，转换为 Vue SFC + TS/TDesign/token/theme/i18n。禁止引入 React/shadcn/
  Tailwind 运行基线、raw DOM widget、全局 reset、新 package 或隐藏验证路径。MCP 是本机知识源，不进依赖/CI/hook。
- 可见文案描述用户操作，不泄漏 AI 调试、starter/demo、实现阶段或开发治理术语。文案禁词仅作用用户 UI，不管
  注释/测试/设计文档。稳定 code/key 对应展示文案由模块 locales 本地化；未知项临时回退须遵守根兼容规则。
- 使用主题 token（--td-*、chartColors、品牌主题）、字体 token 和语义状态色，覆盖明暗模式。数据图表用 ECharts
  或已批准方案，不在页面手绘 SVG/path/坐标轴/tooltip。图表随 mode/brand/locale/尺寸同步并有直接测试。
- 列表空态用 t-empty/table slot，保留 header/body/footer 和分页；筛选空态提供 clear filters，只读页不伪造创建。
- 滚动容器统一 graft-scrollbar，native scrollbar 样式仅受登记 allowlist 约束；新增/修改滚动面同步治理测试和
  check-scrollbar-governance。禁止 alert/confirm/prompt，统一 TDesign Dialog；默认位置由 TConfigProvider 持有。
- 可见时间显式使用当前 vue-i18n locale 与 shared/observability formatter；API/URL/持久化保留 canonical 时间语义。
- 交互异常先根据最小复现、测试或实际事件链证据定位，不凭静态猜测。保留的调试日志归 shell debug runtime，
  namespaced flag 默认关闭，敏感功能另受 DEV；临时探针提交前移除或收口。

## 验证与完成

完成入口：`cd web && bun run check`。阶段内并发：

1. format:check、typecheck、openapi:frontend-governance:check、lint:i18n、lint、stylelint、hygiene:check。
2. 第一阶段全部通过后 test:run 与生产构建；check 内 Vite 构建复用 typecheck，独立 build 仍含 typecheck。

中间可用最小直接测试；功能/任务/合并完成必须完整 check。typecheck/lint/stylelint/test/build 零 warning，
hygiene 的 deadcode、约定 dupcode、非白名单固定字号为零。Vitest 是正式基线；不得 any/as any/关闭 strict
绕过类型问题。不安全边界收口到真实 adapter/client/schema，而非新增无依据兼容层。

已有 hooks/CI 不变：pre-push 命中 web 跑 hygiene；i18n 相关 staged/branch paths 分别触发 pre-commit/pre-push
strict lint:i18n，check 包含同一脚本。配置检查细则见条件文档；不因进入另一个 skill 重跑未变输入的有效全套检查。
手写 TS/Vue closeout 做 graft-comment-governance 审查；仅实际注释 diff 输出相应回执，不新建注释 lint。

编号工作树禁止服务/browser QA/Playwright/截图，完成由 check 证明。浏览器只在获任务授权的开发者主检出区，
独立证明服务 branch/full HEAD 后执行；目标配置不是证明。普通 UI 不默认授权浏览器。需要 browser evidence 时
记录主检出区 follow-up，不阻塞已验证 scoped commit；截图/DOM 不是验收。人工视觉/业务判断仍待验收时明确报告。

closeout 简述变更、边界/风险、验证及缺口；TDesign 涉及时报告 queried/reused/not-needed/fallback 与受影响组件，
已有证据可引用，无关项不重复空模板。Bun 版本/工具链改变同步生成环境 inventory，不混用包管理器刷新 node_modules。
