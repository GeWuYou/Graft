# Lessons Index

## Active Lessons

| ID                                     | Title                                          | Area       | Level | Status | Location                        | Promoted                                                                  |
| -------------------------------------- | ---------------------------------------------- | ---------- | ----: | ------ | ------------------------------- | ------------------------------------------------------------------------- |
| LESSON-BACKEND-HTTPX-CONTEXT-001       | 守卫发布安全审计前必须先写回增强后的请求上下文 | backend    |    L1 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| LESSON-BACKEND-MIGRATION-VERSION-001   | 已执行 Atlas migration 版本不能追加新 DDL      | backend    |    L1 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| LESSON-BACKEND-MODULE-LIFECYCLE-001    | Builder 不应解析 Register 才暴露的跨模块服务   | backend    |    L2 | active | `ai-plan/lessons/backend.md`    | -                                                                         |
| LESSON-GOVERNANCE-BROWSER-BACKEND-001  | 浏览器验收需要真实后端时不要停在 mock 登录页   | governance |    L1 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-GOVERNANCE-SCHEMA-AUTHORITY-001 | 动态配置必须消费 schema 与 i18n authority      | governance |    L2 | active | `ai-plan/lessons/governance.md` | -                                                                         |
| LESSON-WEB-UI-DENSITY-TOKEN-001        | 信息密度切换必须治理 token 消费面              | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-EMPTY-STATE-001          | 表格空状态不应做成小灰色卡片                   | web-ui     |    L3 | active | `ai-plan/lessons/web-ui.md`     | `web/AGENTS.md`, `ai-plan/design/graft-design-system/list-form-detail.md` |
| LESSON-WEB-UI-LOCALE-TIME-001          | 可见时间不能依赖宿主默认语言环境               | web-ui     |    L3 | active | `ai-plan/lessons/web-ui.md`     | `web/AGENTS.md`, `ai-plan/design/前端架构设计.md`                         |
| LESSON-WEB-UI-LOG-AUDIT-001            | 高级查询列表页必须优先抽通用查询结构           | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-PAGE-CONTAINER-001       | 后台页面容器应统一复用共享容器与宽度变量策略   | web-ui     |    L2 | active | `ai-plan/lessons/web-ui.md`     | `ai-plan/design/前端视觉设计规范.md`                                      |
| LESSON-WEB-UI-PROTECTED-STATE-001      | 系统保护状态不应伪装成错误告警                 | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |
| LESSON-WEB-UI-ROUTE-LOADING-001        | 路由切换不能让主内容区短暂卸载为空             | web-ui     |    L1 | active | `ai-plan/lessons/web-ui.md`     | -                                                                         |

## Promoted Rules

| Rule                                                                                                          | Target          | Source Lesson                 | Design Doc                                               |
| ------------------------------------------------------------------------------------------------------------- | --------------- | ----------------------------- | -------------------------------------------------------- |
| Table/list management pages must use `t-empty` or table empty slots instead of custom small gray empty cards. | `web/AGENTS.md` | LESSON-WEB-UI-EMPTY-STATE-001 | `ai-plan/design/graft-design-system/list-form-detail.md` |
| User-visible time must bind the current app locale and must not use host-default datetime formatting.         | `web/AGENTS.md` | LESSON-WEB-UI-LOCALE-TIME-001 | `ai-plan/design/前端架构设计.md`                         |

## Deprecated / Superseded

| ID   | Title | Status | Replacement |
| ---- | ----- | ------ | ----------- |
| None | -     | -      | -           |
