# Lessons Index

## Active Lessons

| ID                                     | Title                                          | Area       | Level | Status | Location                        | Promoted                                                                  |
| -------------------------------------- | ---------------------------------------------- | ---------- | ----: | ------ | ------------------------------- | ------------------------------------------------------------------------- |
| LESSON-BACKEND-CAPABILITY-AUTHORITY-001 | 能力健康必须与平台可达性分层                  | backend    |    L2 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| LESSON-BACKEND-HTTPX-CONTEXT-001       | 守卫发布安全审计前必须先写回增强后的请求上下文 | backend    |    L1 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| MIG-001                               | Existing-data uniqueness must reconcile or abort before the index | migrations | L4 | active | `ai-plan/lessons/migrations.md` | `ai-plan/design/governance/backend/数据库表设计与迁移规范.md` |
| MIG-002                               | Executed Atlas versions must receive forward-only repairs | migrations | L3 | active | `ai-plan/lessons/migrations.md` | `ai-plan/design/governance/backend/数据库表设计与迁移规范.md` |
| LESSON-BACKEND-MODULE-LIFECYCLE-001    | Builder 不应解析 Register 才暴露的跨模块服务   | backend    |    L2 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| LESSON-BACKEND-TASK-OWNER-001          | 跨模块 Task owner 必须使用资源公开稳定标识    | backend    |    L2 | active | `ai-plan/lessons/backend.md`    | `ai-plan/design/architecture/任务执行运行时设计.md`                       |
| LESSON-BACKEND-SAVED-VIEW-001          | 分页保存视图必须分离通用存储与消费页面语义     | backend    |    L2 | active | `ai-plan/lessons/backend.md`    | `ai-plan/design/domains/compose/Compose项目管理设计.md`                  |
| LESSON-GOVERNANCE-BROWSER-BACKEND-001  | 浏览器验收需要真实后端时不要停在 mock 登录页   | governance |    L1 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-GOVERNANCE-COMMENT-VALUE-001    | 代码注释必须记录无法从实现推导的决策           | governance |    L3 | active | `ai-plan/lessons/governance.md` | `AGENTS.md`, `server/AGENTS.md`, `web/AGENTS.md`                         |
| LESSON-GOVERNANCE-PORTABLE-WSL-MCP-001 | 共享 MCP 指引不得固化本机 WSL 发行版            | governance |    L2 | active | `ai-plan/lessons/governance.md` | `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`                 |
| LESSON-GOVERNANCE-EXTERNAL-SKILL-001   | 外部 skill 应按现有能力边界转换并路由          | governance |    L2 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-GOVERNANCE-RELEASE-LINT-001     | Release tag checkout 不应 prune lint 基准分支  | governance |    L1 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-GOVERNANCE-ACTIONS-OUTPUT-TYPE-001 | GitHub Actions job output 不得按布尔值隐式判断 | governance |    L1 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-GOVERNANCE-SCHEMA-AUTHORITY-001 | 动态配置必须消费 schema 与 i18n authority      | governance |    L2 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-WEB-UI-DENSITY-TOKEN-001        | 信息密度切换必须治理 token 消费面              | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-EMPTY-STATE-001          | 表格空状态不应做成小灰色卡片                   | web-ui     |    L3 | active | `ai-plan/lessons/web-ui.md`     | `web/AGENTS.md`, `ai-plan/design/graft-design-system/list-form-detail.md` |
| LESSON-WEB-UI-HOMEPAGE-AUTHORITY-001   | 首页内容扩充必须落在首页权威链路               | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`           |
| LESSON-WEB-UI-LOCALE-TIME-001          | 可见时间不能依赖宿主默认语言环境               | web-ui     |    L3 | active | `ai-plan/lessons/web-ui.md`     | `web/AGENTS.md`, `ai-plan/design/architecture/前端架构设计.md`            |
| LESSON-WEB-UI-LOG-AUDIT-001            | 高级查询列表页必须优先抽通用查询结构           | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-MONACO-WORKER-001        | Monaco YAML worker 故障要先区分两层兼容问题    | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-PAGE-CONTAINER-001       | 后台页面容器应统一复用共享容器与宽度变量策略   | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | `ai-plan/design/governance/frontend/前端视觉设计规范.md`                  |
| LESSON-WEB-UI-PREVIEW-SHELL-001        | 设计预览应区分匿名直达与已登录真实壳层         | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-SEMANTIC-PORT-001        | 从工作树移植功能时保留当前页面骨架             | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-PROTECTED-STATE-001      | 系统保护状态不应伪装成错误告警                 | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-ROUTE-LOADING-001        | 路由切换不能让主内容区短暂卸载为空             | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-SELECTOR-AUTHORITY-001   | 多来源选择器必须保留用户意图并独立表达可用性   | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-TAB-INDICATOR-001       | TDesign Tab 指示条必须定位在完整激活导航项边缘 | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |

## Promoted Rules

| Rule                                                                                                          | Target          | Source Lesson                 | Design Doc                                               |
| ------------------------------------------------------------------------------------------------------------- | --------------- | ----------------------------- | -------------------------------------------------------- |
| Table/list management pages must use `t-empty` or table empty slots instead of custom small gray empty cards. | `web/AGENTS.md` | LESSON-WEB-UI-EMPTY-STATE-001 | `ai-plan/design/graft-design-system/list-form-detail.md` |
| User-visible time must bind the current app locale and must not use host-default datetime formatting.         | `web/AGENTS.md` | LESSON-WEB-UI-LOCALE-TIME-001 | `ai-plan/design/architecture/前端架构设计.md`            |
| Handwritten Go, TypeScript, and Vue changes must receive a value-based comment review before closeout.         | `AGENTS.md`, `server/AGENTS.md`, `web/AGENTS.md` | LESSON-GOVERNANCE-COMMENT-VALUE-001 | `ai-plan/design/governance/ai/代码注释与模块文档规范.md` |

## Deprecated / Superseded

| ID   | Title | Status | Replacement |
| ---- | ----- | ------ | ----------- |
| None | -     | -      | -           |
