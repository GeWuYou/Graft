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

- Phase 0 正在完成。
- 设计文档和 public topic 已创建。
- 下一批次应从 OpenAPI、migration 和 `server/modules/announcement` 模块骨架开始。

## Task Checklist

- [ ] Phase 0：设计和 topic 持久化
- [ ] Phase 1：OpenAPI + migration + 后端领域模型
- [ ] Phase 2：后端管理端 API
- [ ] Phase 3：后端用户端 API + 已读状态
- [ ] Phase 4：前端公告管理页
- [ ] Phase 5：用户侧公告入口 / 工作台摘要
- [ ] Phase 6：测试、i18n、治理收尾

## Dirty Worktree Note

当前工作树存在 notification 相关未提交改动：

- `server/modules/notification/migrations/atlas.sum`
- `server/modules/notification/publisher_test.go`
- `server/modules/notification/store/sql_repository.go`
- `server/modules/notification/store/store.go`
- `server/modules/notification/migrations/202606120002_notification_delivery_deleted_at_epoch.sql`

这些文件不属于公告中心 owned scope。
