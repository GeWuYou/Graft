# Web UI Lessons

## LESSON-WEB-UI-SEMANTIC-PORT-001：从工作树移植功能时保留当前页面骨架

- Status: active
- Level: L1
- Applies to:
  - 从任务工作树或候选分支选择性移植的 `web` 页面能力
  - monitor、dashboard 与其他已有后台页面的指标、图表和深链增强
- Source:
  - 可观测性能力集成中，用户明确要求保留现有页面 UI，只增加指标趋势、统计和访问日志下钻
- Problem:
  候选工作树往往同时包含有价值的数据能力和与当前分支不一致的页面重构。整页合并会覆盖已经验收的布局、卡片层级、响应式行为和本地化键，导致功能增强变成无授权的视觉重写。
- Correct pattern:
  以当前分支页面的容器、主栅格、既有卡片、抽屉和导航交互作为 UI 真相；只移植需要的 OpenAPI、服务端聚合、页面内指标、趋势图、明细和深链语义。新增内容应嵌入已有信息架构，并通过桌面与移动端截图确认原有结构仍可辨识。
- Anti-pattern:
  - 直接合并候选工作树的整个页面、样式或 locale 文件
  - 为了加入一个指标替换现有卡片分组、主栅格或页面壳
  - 把已有功能的导航和回退上下文删除或改成新页面流
- Enforcement:
  选择性移植前比较候选页与当前页的布局和 locale 差异；实现后运行相关组件测试与 `bun run check`，并用浏览器截图复核既有首屏结构、响应式断点和新增下钻路径。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/modules/monitor/pages/dependencies/index.vue`
  - `web/src/modules/monitor/pages/request-performance/index.vue`
  - `web/src/modules/access-log/pages/list/index.vue`
- Updated at:
  2026-07-22

## LESSON-WEB-UI-MENU-ICON-AUTHORITY-001：菜单图标必须从上游 descriptor 的语义 key 统一解析

- Status: active
- Level: L2
- Applies to:
  - server menu descriptor metadata
  - web shell navigation
  - Iconify menu icon resolver
- Source:
  - 用户反馈指出可见 sidebar 将不识别的 menu icon 回退为重复 folder，Docker 又被误替换为不协调的品牌样式
- Problem:
  直接沿用旧图标库 identifier 或在页面按菜单名称修补，会让 resolver 回退到同一个 generic glyph；菜单层级、Overview、Runtime、Dependencies、日志等对象因此失去区分度。品牌图标若绕过同一接口或运行时加载，也会破坏尺寸、stroke 和离线可用性。
- Correct pattern:
  先修 server descriptor 的 canonical semantic key，再由一个 shell-owned Iconify resolver 静态映射。常规导航使用 Lucide；仅缺少专业 glyph 时使用 Tabler；Docker 使用静态 `tabler:brand-docker`。新 key 必须有 resolver test，菜单修改必须用真实 sidebar 截图确认 SVG 已渲染、相邻可见入口不重复。
- Anti-pattern:
  - 把未知 icon key 静默回退为 folder 并当作完成
  - 为单一菜单名称添加前端条件分支
  - 用通用 server/container 图标替代 Docker 品牌 glyph
  - 依赖 Iconify CDN 或运行时网络加载
- Enforcement:
  修改可见菜单时，检查 descriptor identifier、`web/src/shared/icons` resolver 和 `MenuContent` SVG test；运行 `bun run check`、相关 Go menu tests，并用 browser agent 截图核验。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes (`ai-plan/design/domains/compose/Compose项目管理设计.md`)
- Updated at:
  2026-07-12

## LESSON-WEB-UI-MONACO-WORKER-001：Monaco YAML worker 故障要先区分 Vite 入口问题和 createWebWorker API 漂移

- Status: active
- Level: L2
- Applies to:
  - `web` Monaco editor / diff editor integration
  - `monaco-yaml`
  - Vite worker bundling
  - third-party Monaco language-service adapters that still依赖 legacy `createWebWorker({ label, moduleId, createData })`
- Source:
  - project configuration workspace YAML editor runtime failure
  - user feedback that the correct action is root-cause repair, not temporary disablement of `monaco-yaml`
  - diagnosis of repeated browser errors: `Missing requestHandler or method: doValidation`, `resetSchema`,
    `findLinks`, `findDocumentSymbols`, `getFoldingRanges`
- Problem:
  Monaco YAML 能正常高亮并不代表 YAML language worker 已经正确接入。该问题至少有两层常见故障面：
  一是 Vite 下 `yaml.worker` 入口未按 `monaco-yaml` workaround 包装，导致 YAML worker 文件本身无法正确产出或启动；
  二是
  `monaco-yaml` 依赖的 `monaco-worker-manager` 仍按旧接口调用 `monaco.editor.createWebWorker({ label, moduleId, createData })`，
  但当前 `monaco-editor` standalone ESM 实现已经收敛到 `worker` 形态，不再按旧参数自行创建 foreign worker。第二层漂移会让
  `monaco-yaml` 以为拿到了 YAML worker，实际上拿到的是普通 `editorWorkerService`，随后一切
  `doValidation/resetSchema/findLinks/findDocumentSymbols/getFoldingRanges` 调用都会落到错误 worker 并统一报
  `Missing requestHandler or method: ...`。
- Correct pattern:
  接入或修复 `monaco-yaml` 时，必须按两层顺序排查：
  1. 先确保 YAML worker 入口遵循 `monaco-yaml` README 的 Vite workaround：本地包装
     `project-yaml.worker.js`，再用 `?worker` 方式导入并创建 YAML worker，而不是直接引用包内 worker 路径。
  2. 再确认当前仓库使用的 `monaco-editor` 版本是否仍兼容旧的 `createWebWorker({ label, moduleId, createData })`
     签名；若不兼容，应在项目自己的 Monaco 启动层补一个兼容桥，把 legacy 调用转成当前 Monaco 可工作的
     `worker: Promise<Worker>` 形态，并继续按 `label` 显式路由到 `yaml` / `json` / `editor` worker。
  3. 排查时优先看浏览器日志是否真的出现：
     - `label: 'yaml'`
     - `workerKind: 'yaml'`
     - `worker-created { kind: 'yaml' }`
     若只有 `editorWorkerService`，说明仍然没有真正进入 YAML foreign worker。
- Anti-pattern:
  - 看到 YAML 运行时报错就直接停用 `configureMonacoYaml(...)`
  - 只修 `yaml.worker` 打包入口，却忽略 `createWebWorker` 旧接口在当前 Monaco 版本上已经漂移
  - 只根据 `Could not create web worker(s)` 一条 warning 判断根因，而不继续核对后续方法缺失报错
  - 看到 `Missing requestHandler or method: ...` 就去改 `monaco-yaml` provider 逻辑，而不先确认实际拿到的 worker 类型
  - 不留浏览器侧 worker 路由日志，导致后续只能重复猜测
- Enforcement:
  修改 Monaco / `monaco-yaml` 接入时，至少运行 `cd web && bun run typecheck` 和相关 Vitest 用例；浏览器验收时，
  打开控制台确认 YAML worker 的 `label`、`workerKind`、`worker-created` 三组日志。若再次出现
  `Missing requestHandler or method: doValidation/resetSchema/findLinks/findDocumentSymbols/getFoldingRanges`，
  应直接检查：
  - `web/src/modules/project/shared/project-monaco.ts`
  - `web/src/modules/project/shared/project-monaco-worker.ts`
  - `web/src/modules/project/shared/project-yaml.worker.js`
  - `web/node_modules/monaco-yaml/README.md`
  - `web/node_modules/monaco-worker-manager/index.js`
  - `web/node_modules/monaco-editor/esm/vs/editor/standalone/browser/standaloneWebWorker.js`
  - `web/node_modules/monaco-editor/esm/vs/common/workers.js`
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/modules/project/shared/project-monaco.ts`
  - `web/src/modules/project/shared/project-monaco-worker.ts`
  - `web/src/modules/project/shared/project-yaml.worker.js`
  - `web/src/modules/project/shared/project-monaco-debug.ts`
  - `web/node_modules/monaco-yaml/README.md`
  - `web/node_modules/monaco-worker-manager/index.js`
  - `web/node_modules/monaco-editor/esm/vs/editor/standalone/browser/standaloneWebWorker.js`
  - `web/node_modules/monaco-editor/esm/vs/common/workers.js`
- Updated at:
  2026-07-08

## LESSON-WEB-UI-ROUTE-LOADING-001：路由切换不能让主内容区短暂卸载为空

- Status: active
- Level: L1
- Applies to:
  - `web` shell layout
  - `router-view` / `keep-alive` route transitions
  - Footer and page-container layout stability
- Source:
  - container detail navigation screenshot where Header, Sidebar, and Tabs were rendered but the main content area was
    empty, causing the global Footer copyright to appear in the middle of the viewport
  - shell-level route loading remediation for async route component and detail API latency
- Problem:
  后台壳层在路由切换、异步组件加载或标签页刷新期间，如果直接让 `router-view` 短暂渲染为空，页面主内容区会失去高度。
  Footer 虽然没有 fixed，也会因为前面的内容坍缩而上浮，等详情页或 API 数据回来后再被挤下去，形成明显 layout shift。
- Correct pattern:
  壳层应在主内容区内提供统一的 TDesign page loading host，例如 `t-loading` 包裹 `router-view`，并让 loading host、
  page container 和 content body 连续 `flex: 1` 且有稳定 `min-height`。路由守卫负责路由组件加载阶段的 loading
  状态，页面内部再负责 API 数据阶段的 skeleton/loading/error/empty 状态。Loading 不应遮住 Header、Sidebar 或 Tabs。
- Anti-pattern:
  - 在每个详情页复制一套路由切换 loading
  - 为了遮住上浮把 Footer 改成 fixed
  - 用 `display: none` 或空 `router-view` 隐藏切换期内容
  - 详情数据未返回时卸载整个页面外壳，只留下空白主内容区
  - 只保留 NProgress 顶部进度条，却不保留主内容区占位高度
- Enforcement:
  修改 shell layout、route guard、tabs refresh 或详情页首屏 loading 时，用 `bun run check` 覆盖前端完成态；
  对可见布局问题补浏览器测量或截图，确认切换期间 content host 有稳定高度、Footer 位置不跳动、Header/Sidebar/Tabs
  不被 loading 遮挡且没有横向滚动。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/layouts/components/Content.vue`
  - `web/src/router/route-loading.ts`
  - `web/src/style/layout.less`
- Updated at:
  2026-06-17

## LESSON-WEB-UI-LOCALE-TIME-001：可见时间不能依赖宿主默认语言环境

- Status: active
- Level: L3
- Applies to:
  - `web` dashboard, monitor, audit, log, notification, and future time-display surfaces
  - shared visible time formatters
  - i18n governance scripts
- Source:
  - dashboard screenshot showing English host-locale timestamp fragments in a localized Chinese UI
  - remediation of the i18n governance gate gap that allowed host-locale datetime formatting
- Problem:
  用户可见时间若使用 `new Intl.DateTimeFormat(undefined, ...)` 或无参数 `toLocaleString()` 等 API，会从浏览器或运行环境继承默认
  locale。这样即使页面文案和菜单已经切到 `zh-CN`，时间仍可能显示英文月份、英文 AM/PM 或其他宿主格式，形成局部未本地化。
- Correct pattern:
  可见时间格式化必须显式绑定应用当前 `vue-i18n` locale，并默认收口到 `web/src/shared/observability` 的 locale-aware
  formatter。API payload、URL query 和持久化状态继续保持 canonical timestamp 或页面本地输入语义，不为了展示效果把
  wire contract 改成本地化字符串。
- Anti-pattern:
  - 在页面、组件或模块共享展示 helper 中使用 `new Intl.DateTimeFormat(undefined, ...)`
  - 用无参数 `toLocaleString()` / `toLocaleDateString()` / `toLocaleTimeString()` 渲染可见时间
  - 在 server 或 API response 中预渲染常规 UI 展示时间来绕过前端 locale 绑定
  - 每个模块各自维护一套时间 formatter，导致日期、时间、秒级精度和空值回退不一致
- Enforcement:
  `bun run lint:i18n` 必须阻断生产源码中的 host-locale 可见时间格式化；修改时间展示时运行 `bun run lint:i18n`、
  相关 formatter/page 测试和 `bun run check`。新增例外必须证明不是用户可见时间路径。
- Promotion:
  - AGENTS.md: yes
  - Design doc: yes
- Related:
  - `web/AGENTS.md`
  - `ai-plan/design/architecture/前端架构设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
  - `web/scripts/check-i18n-governance.ts`
  - `web/src/shared/observability/time.ts`
- Updated at:
  2026-06-10

## LESSON-WEB-UI-PROTECTED-STATE-001：系统保护状态不应伪装成错误告警

- Status: active
- Level: L1
- Applies to:
  - `web` management pages with builtin/system/protected records
  - readonly drawers, dialogs, and action menus
  - `list-form-detail` page type
- Source:
  - role management system-role UX remediation
  - user feedback that builtin roles are normal business state, not warning/error conditions
- Problem:
  系统内置角色、只读权限、受保护配置这类状态是平台的正常保护模型。若用橙色 warning 块、置灰可写按钮或隐藏原因来表达，
  用户会误以为数据异常或权限系统出错，也不知道下一步能做什么。
- Correct pattern:
  对受保护但正常的业务状态，应使用 info、neutral 或 primary-light 语义，文案明确说明“这是正常限制，不是异常”。
  操作模型应从可写操作切换为只读操作，例如“查看权限”替代禁用的“分配权限”；更多菜单只暴露可执行动作，例如详情、
  查看、复制为自定义对象。只读弹窗应允许搜索、展开和查看说明，但隐藏保存按钮并保留关闭动作。
- Anti-pattern:
  - 用 warning/error 样式表达正常系统保护状态
  - 保留可写按钮但简单 disabled，且没有说明原因
  - 对系统内置对象暴露编辑、删除、分配权限等不可执行操作
  - 只在前端隐藏危险操作，却不确认后端 authority 有拒绝路径
- Enforcement:
  修改带 builtin/system/protected 字段的管理页时，检查表格操作、详情抽屉、弹窗标题、提示块和 footer 是否按只读/保护态
  切换；确认文案在模块 i18n 中；确认后端或 contract authority 已拒绝不可执行写操作。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/shared/components/assignment/AssignmentFooter.vue`
- Updated at:
  2026-06-07

## LESSON-WEB-UI-EMPTY-STATE-001：表格空状态不应做成小灰色卡片

- Status: active
- Level: L3
- Applies to:
  - `web` table/list management pages
  - `list-form-detail` page type
  - TDesign Vue Next table empty states
- Source:
  - 用户管理 / 角色管理空状态修复
  - user feedback on wrong empty-state implementation direction
- Problem:
  AI 曾在用户管理、角色管理这类表格页中，把“暂无数据”实现成表格中间的小灰色卡片，视觉上像浮层或占位块。
  这种做法不符合 TDesign 管理页空态模式，也容易与表格容器、分页和暗色主题产生割裂。
- Correct pattern:
  对于 table/list management 页面，应优先使用 `t-empty` 组件或 `t-table` 的 `empty` 插槽。空状态应位于 table body
  中央，保留表头和分页结构，颜色与边框必须使用 TDesign token。创建型管理页可提供主操作按钮；当筛选条件激活时，可同时
  提供“清空筛选”操作。
- Anti-pattern:
  - 手写 `empty-card` 或 `empty-box`
  - 在表格中间放一个小灰色盒子
  - 硬编码 `#f5f5f5`、`#fff`、`#000`
  - 让空状态挤乱分页
  - 让空状态与表格 header/body/footer 结构割裂
  - 让暗色主题下的空状态背景或文案失去可读性
- Enforcement:
  实现或修改 table/list management 页面时，必须检查 empty state 是否使用 `t-empty` 或 table empty slot，分页是否稳定，
  表头/空态/分页结构是否连续，以及颜色是否全部来自 TDesign token。
- Promotion:
  - AGENTS.md: yes
  - Design doc: yes
- Related:
  - `web/AGENTS.md`
  - `ai-plan/design/graft-design-system/list-form-detail.md`
- Updated at:
  2026-05-22

## LESSON-WEB-UI-LOG-AUDIT-001：高级查询列表页必须优先抽通用查询结构

- Status: active
- Level: L2
- Applies to:
  - `query-builder-list-detail` page type
  - `web` log and audit pages
  - `log-audit` page type
  - access-log, app-log, audit logs, and future field-heavy query pages
- Source:
  - app-log page remediation after user feedback that the page did not follow the access-log page pattern
  - duplicate local implementations discovered while aligning app-log with access-log
  - query-list refactor after user feedback that the abstraction should serve future field-heavy query pages, not only logs
- Problem:
  字段多、筛选复杂、需要分页表格和详情抽屉的页面若按单页临时实现，很容易出现筛选器、表格排序、详情抽屉、深链参数、
  空态、错误提示和交互文案不一致。这种分叉会让相同类型页面看起来像不同产品，也会在严格重复代码检查下产生维护成本。
- Correct pattern:
  新增或重做字段密集查询页时，先声明页面类型为 `query-builder-list-detail`；日志审计只是 `log-audit` 变体。页面壳、
  筛选构建器、分页表格、列设置抽屉、列表错误提示和通用交互应优先沉淀到 `web/src/shared/components/query-list`，
  由页面只提供领域字段、API 查询、深链语义、详情组件和展示文案。
- Anti-pattern:
  - 在每个字段密集查询页手写一套筛选器、表格、列设置、详情抽屉和错误反馈
  - 只复制访问日志页面的视觉结果，却保留不同的数据流、外壳结构和交互语义
  - 用本地兼容映射掩盖后端契约缺少排序、筛选或分页字段的问题
  - 为通过重复代码检查而做无语义的改名或拆行
- Enforcement:
  修改字段密集查询页时，检查页面类型是否为 `query-builder-list-detail` 或其变体，并确认页面壳、筛选器、分页表格、
  列设置和错误态是否复用 `shared/components/query-list`。业务字段、API 查询、URL deep-link 和详情内容仍留在模块内。
  用 `bun run dupcode:check`、相关 Vitest 用例和 `bun run check` 验证没有重新分叉。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/shared/components/query-list`
  - `web/src/modules/access-log/pages/list/index.vue`
  - `web/src/modules/app-log/pages/list/index.vue`
  - `web/src/modules/audit/pages/logs/index.vue`
- Updated at:
  2026-06-04

## LESSON-WEB-UI-PAGE-CONTAINER-001：后台页面容器应统一复用共享容器与宽度变量策略

- Status: active
- Level: L2
- Applies to:
  - `web` management pages
  - shared page containers such as `ManagementPageContent`
  - large-screen layout tuning and visual-centering fixes
- Source:
  - cross-page layout consistency review for access control, user, role, and monitor pages
  - repeated left/right whitespace and centering drift discussions
- Problem:
  后台页面在大屏下容易出现左右留白、宽度策略不一致和视觉重心漂移。若每个页面单独修宽度或偏移，长期会导致容器策略失控。
- Correct pattern:
  后台页面应优先复用共享页面容器与宽度变量策略，例如统一的 `max-width`、`width-ratio`、`min-padding` 和
  `margin-inline: auto`。管理页可以通过变量覆盖获得更宽的内容面，但不能破坏整体居中。排查时应优先查看 DOM/CSS
  计算结果，而不是只凭截图主观修偏移。
- Anti-pattern:
  - 单页写死 `margin-left`
  - 用 `transform` 修视觉偏移
  - 每个页面自行定义长期 `max-width` 策略
  - 忽略滚动条、悬浮工具按钮或容器嵌套对视觉重心的影响
  - 让共享容器和页级容器同时争夺宽度真值
- Enforcement:
  新增或调整后台页面宽度时，先检查是否已有共享容器可复用；若需要覆盖宽度，必须通过变量而不是局部偏移 hack；若出现视觉不居中，
  需要检查实际容器计算宽度与布局约束。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes
- Related:
  - `ai-plan/design/governance/frontend/前端视觉设计规范.md`
  - `web/src/shared/components/management/ManagementPageContent.vue`
- Updated at:
  2026-05-22

## LESSON-WEB-UI-DENSITY-TOKEN-001：信息密度切换必须治理 token 消费面

- Status: active
- Level: L2
- Applies to:
  - `web` theme workbench and density presets
  - `web/src/style/**` global layout tokens
  - `web/src/layouts/**`, `web/src/shared/**`, and module page styles
- Source:
  - information-density switch remediation after user feedback that only a small amount of text spacing changed
  - full-web density governance gate added for `web/src/**`
- Problem:
  信息密度 preset 如果只更新 TDesign component size token 或单个 `--graft-theme-density-scale`，而页面继续写死
  `gap`、`padding`、`margin`、`t-space size` 和图表 tooltip 内联间距，设置面板会显示可切换，但真实页面节奏几乎不变。
  这种分叉会让主题工作台变成“看得见的配置、感受不到的体验”。
- Correct pattern:
  信息密度能力必须同时覆盖 source token 和消费面。密度相关布局应使用 `--td-comp-*`、`--graft-density-*` 或
  `calc(...var(--graft-theme-density-scale)...)`，并让共享组件、业务页面、TDesign `Space` 尺寸和图表 tooltip
  内联模板共同响应同一套 density authority。
- Anti-pattern:
  - 只在 store 里生成密度 token，不替换页面里的固定间距
  - 在业务页面继续新增裸 `gap: 16px`、`padding: 12px 14px` 或 `<t-space size="8px">`
  - 用局部 class 名如 `compact` 伪装为全局信息密度响应
  - 把图表 tooltip HTML 字符串排除在密度治理之外
- Enforcement:
  修改前端布局或新增页面时，运行 `bun run density:check` 或完整 `bun run check`。扫描发现的固定密度间距必须改为
  TDesign/Graft density token；只有图标盒、断点、安全区、滚动条、媒体尺寸等非信息密度几何值才允许进入脚本白名单，
  且白名单必须写明具体原因。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/scripts/check-density-governance.ts`
  - `web/src/store/modules/setting.ts`
  - `web/src/style/layout.less`
- Updated at:
  2026-06-05

## LESSON-WEB-UI-TAB-INDICATOR-001：TDesign Tab 指示条必须定位在完整激活导航项边缘

- Status: active
- Level: L1
- Applies to:
  - TDesign Vue Next `Tabs` / `TabPanel` 的自定义导航样式
  - 后台壳层 Tab 激活态、主题色指示器和类似边缘状态标记
- Source:
  - 用户反馈 Tab 色条无法正确预览，顶部和底部效果都落在标签文字中央
  - 修复主题工作台中的 Tab 指示条定位与 TDesign 激活态选择器
- Problem:
  把边缘指示器伪元素挂在 Tab 的文字包裹层时，伪元素尺寸只跟随文案内容，`top` / `bottom` 视觉上会退化成文字区域中央的色块；如果同时创建两个未定位的伪元素，顶部和底部也会看起来没有区别。
- Correct pattern:
  先确认 TDesign 的 DOM 结构，再把指示器定位在完整 `.t-tabs__nav-item.t-is-active` 上；顶部只生成 `::before` 并设 `top: 0`，底部只生成 `::after` 并设 `bottom: 0`。预览缩略图应复用同样的上下边缘语义，并使用主题 token 颜色。
- Anti-pattern:
  - 把 Tab 边缘色条挂在内部文字或 dropdown 包裹节点
  - 依赖文案节点的 active 标记或 `:has()` 反推组件激活态
  - 同时创建上下两个未定位的伪元素
- Enforcement:
  修改 TDesign Tab 样式时查询组件 DOM，检查选择器是否命中完整激活导航项；为 `none`、`top`、`bottom` 补组件测试，并运行 `bun run check`。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/layouts/components/LayoutContent.vue`
  - `web/src/layouts/components/theme-workbench/ThemeWorkbenchPanel.vue`
  - `web/src/layouts/components/LayoutContent.test.ts`
- Updated at:
  2026-08-08

## LESSON-WEB-UI-PREVIEW-SHELL-001：设计预览应区分匿名直达与已登录真实壳层

- Status: active
- Level: L1
- Applies to:
  - `web` development-only 页面预览
  - 需要与正式页面做视觉对照的截图验收
  - 复用 Graft Header、Sidebar、Tabs 和内容容器的壳层内原型
- Source:
  - 首页工作台第二轮视觉验收发现：预览路由在认证流程前无条件短路会让已登录会话也失去 bootstrap 菜单，导致截图中的空侧栏成为非设计差异
- Problem:
  开发预览若为了匿名直达而无条件跳过会话 bootstrap，页面内容虽然可渲染，但 Header、Sidebar、菜单授权状态和内容壳层会偏离正式页面。此时新旧截图无法只比较目标页面，评审容易把壳层差异误判为信息架构或视觉质量问题。
- Correct pattern:
  development-only 预览可以在无 token 时直接进入，以保留独立 UI 验收能力；检测到已有会话时，应沿用正式 bootstrap、菜单和壳层装配。截图验收固定 viewport、zoom、主题和侧栏状态，只让目标内容发生变化。release build 必须继续排除预览路由。
- Anti-pattern:
  - 对所有预览访问无条件绕过 bootstrap
  - 为截图伪造一套侧栏或菜单数据
  - 用空侧栏截图与正式首页做视觉优劣对比
  - 为复用壳层而把 development-only 预览注册进 release 路由
- Enforcement:
  预览路由同时测试匿名直达与已登录 bootstrap 两条分支，release 路由测试确认预览不可达；最终截图检查 Header、Sidebar、viewport、zoom 和主题与正式对照一致。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/app/bootstrap/route-guards.ts`
  - `web/src/router/development-routes.development.ts`
  - `web/src/permission.test.ts`
- Updated at:
  2026-08-17

## LESSON-WEB-UI-HOMEPAGE-AUTHORITY-001：首页内容扩充必须落在首页权威链路

- Status: active
- Level: L2
- Applies to:
  - 首页、Dashboard 和全局工作台的内容扩充
  - 通过 registry 聚合多个模块事实的页面
  - 首页到管理列表、对象详情和隐藏旧路由的 drill-down 关系
- Source:
  - 用户明确纠正：扩建一个进入后仍然稀疏的 Docker 资源聚合页，不等于丰富首页；其初衷是让首页展示更多真实内容，必要时可以修改后端事实贡献
- Problem:
  首页内容不足时，如果把工作转向一个无稳定导航身份的隐藏聚合页，或者只展示“注册了多少模块”而没有展示模块拥有的运行事实，
  最终交付会偏离用户正在评价的首页。canonical 管理页能够承接下钻，不代表它可以替代首页内容；模块数量也只能说明覆盖面，
  不能回答首页应回答的当前状态、关注事项、健康与近期活动。
- Correct pattern:
  先确认实际首页路由、产品定位、Dashboard Registry 和 presentation composition 的权威，再由相关模块提供真实、权限过滤、
  数量有界的 owner-owned contribution，由首页统一编排首屏层级、展开预算和证据状态。管理列表与对象详情继续作为
  canonical navigation target，只承接查看和处置；它们不能替代首页事实，也不能为了填充首页而复活隐藏聚合页。
- Anti-pattern:
  - 新建或扩充一个稀疏、隐藏、无稳定管理对象的下游聚合页，并把它当作首页扩充结果
  - 只统计已注册模块、菜单或入口数量，却没有把模块拥有的真实状态贡献到首页
  - 在管理页展示大量原始表格，再用一个首页链接宣称已经补足首页信息
  - 只验收新页面截图，不打开实际首页检查 DOM、内容层级和 drill-down 路由
- Enforcement:
  实施前用设计与规格 review 明确首页 authority、模块事实 owner、展示预算和 canonical drill-down；实现后必须在实际首页路由
  获取浏览器 DOM 与截图证据，并逐一验证首页动作进入预期管理页或对象详情。若变更涉及隐藏旧页或迁移路径，使用 old-path grep
  检查旧路由、旧菜单和旧页面引用已按设计删除，例如执行
  `rg -n '/infrastructure/docker/containers/resources' server web openapi` 复核生产引用，并确认首页没有通过旧路径获取或替代内容。
- Promotion:
  - AGENTS.md: no
  - Design doc: yes (`ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`，已有权威设计)
- Related:
  - `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
- Updated at:
  2026-08-17
