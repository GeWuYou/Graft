# Web UI Lessons

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

## LESSON-WEB-UI-LOG-AUDIT-001：日志审计页必须按页面类型复用标准结构

- Status: active
- Level: L2
- Applies to:
  - `web` log and audit pages
  - `log-audit` page type
  - access-log, app-log, and future observability list pages
- Source:
  - app-log page remediation after user feedback that the page did not follow the access-log page pattern
  - duplicate local implementations discovered while aligning app-log with access-log
- Problem:
  日志类页面若按单页临时实现，很容易出现筛选器、表格排序、详情抽屉、深链参数、空态、错误提示和交互文案不一致。
  这种分叉会让相同类型页面看起来像不同产品，也会在严格重复代码检查下产生维护成本。
- Correct pattern:
  新增或重做日志审计页时，先声明页面类型为 `log-audit`，再以访问日志这类已验证页面为主模板。筛选构建器、分页表格、
  URL 深链、列表错误提示和交互处理应优先沉淀到 `web/src/shared/observability` 或同类型共享层，由页面只提供领域字段、
  API 查询和展示文案。
- Anti-pattern:
  - 在每个日志页手写一套筛选器、表格、详情抽屉和 URL 参数解析
  - 只复制访问日志页面的视觉结果，却保留不同的数据流和交互语义
  - 用本地兼容映射掩盖后端契约缺少排序、筛选或分页字段的问题
  - 为通过重复代码检查而做无语义的改名或拆行
- Enforcement:
  修改日志类页面时，检查根节点或页面元信息是否标识 `log-audit`，并对照访问日志页面确认筛选、表格、详情、深链、
  i18n 和错误态结构一致。若 app-log 或 future log 页面需要新增行为，优先评估能否扩展共享 observability helper，
  同时用 `bun run dupcode:check` 和相关 Vitest 用例验证没有重新分叉。
- Promotion:
  - AGENTS.md: no
  - Design doc: no
- Related:
  - `web/src/modules/access-log/pages/list/index.vue`
  - `web/src/modules/app-log/pages/list/index.vue`
  - `web/src/shared/observability`
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
  - `ai-plan/design/前端视觉设计规范.md`
  - `web/src/shared/components/management/ManagementPageContent.vue`
- Updated at:
  2026-05-22
