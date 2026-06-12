# Announcement Center MVP Tracking

## Topic

Announcement Center MVP

## Scope

实现独立公告中心模块，支持管理员发布长期公告、用户阅读公告、置顶、过期、归档、已读状态和用户侧入口。

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/公告中心设计.md`
- `ai-plan/design/通知中心设计.md`
- `ai-plan/design/数据库表设计与迁移规范.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/前端架构设计.md`

## Current Recovery Point

- Phase 0 已完成并提交。
- Phase 1 已完成并提交，OpenAPI、migration、server module foundation 和 generated artifacts 已建立。
- Phase 2 已完成并提交，管理端 API 行为和 focused tests 已建立。
- Phase 3 已完成并提交，用户端 API、已读状态、未读数和用户隔离已建立。
- 下一批次进入 Phase 4，实现前端公告管理页。

## Task Checklist

- [x] Phase 0：设计和 topic 持久化
- [x] Phase 1：OpenAPI + migration + 后端领域模型
- [x] Phase 2：后端管理端 API
- [x] Phase 3：后端用户端 API + 已读状态
- [ ] Phase 4：前端公告管理页
- [ ] Phase 5：用户侧公告入口 / 工作台摘要
- [ ] Phase 6：测试、i18n、治理收尾
