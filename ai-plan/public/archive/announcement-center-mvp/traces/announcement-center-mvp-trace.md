# Announcement Center MVP Trace

## 2026-06-12

- 启动任务并确认 task class 为 `cross-boundary`。
- 当前分支从 `build/web-tdesign-on-demand-imports` 重命名为 `feat/announcement-center-mvp`。
- 建立公告中心设计 authority：`ai-plan/design/domains/notification/公告中心设计.md`。
- 建立 public recovery topic：`ai-plan/public/announcement-center-mvp/README.md`。
- 明确公告中心不复用 notification domain model，MVP 不做 notification fan-out。
- Phase 0 已提交：`fc01e643 docs(announcement): define announcement center MVP`。
- Phase 1 已提交：`3e27181f feat(announcement): add center foundation`。
- Phase 1 完成 OpenAPI schemas/paths、server generated types、web generated schema、`server/modules/announcement`
  基础模块、migration、权限菜单注册和 focused validation。
- Phase 2 已提交：`12133c88 feat(announcement): implement management API`。
- Phase 2 完成管理端列表、创建、详情、更新、发布、归档、删除、状态流转、过滤排序、错误映射和 focused tests。
- Phase 3 已提交：`e9d74363 feat(announcement): implement current-user API`。
- Phase 3 完成用户侧列表、已读、全部已读、未读数、可见性规则和用户隔离 tests。
- Phase 4 已提交：`cecea602 feat(announcement): add management web UI`。
- Phase 4 完成 `web/src/modules/announcement` 管理端模块、API client、bootstrap route、presenter、i18n、
  管理页和 focused/full web validation。
- Phase 5 已提交：`0b019358 feat(announcement): add user entry dashboard`。
- Phase 5 完成用户侧 `/announcements` global route、当前用户公告页、已读操作、顶部独立公告 badge 入口和
  focused/full web validation。
- Phase 5 未直接加入 dashboard 摘要，因为当前 dashboard 由后端 summary/widgets 驱动，缺少干净的前端模块贡献点；
  该摘要保留为后续 dashboard contribution contract 设计项。
- Phase 6 发现 `server/modules/announcement/migrations` 有 live SQL 但缺少 `atlas.sum`，导致 backend
  完成态校验失败；已用 `atlas migrate hash --dir file://server/modules/announcement/migrations` 补齐。
- Phase 6 最终校验通过：
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
  - `python3 scripts/validate_sql_migrations.py`
  - `python3 scripts/check_migration_versions.py --mode all`
  - `git diff --check`
- Phase 6 独立性检查确认公告实现未向 `notification_events` 写入公告正文，顶部公告入口与 notification bell 分离。
- Phase 6 归档判定：`archive-ready`。

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-topic-persistence",
    "phase-1-openapi-server-foundation",
    "phase-2-server-management-api",
    "phase-3-server-user-api",
    "phase-4-web-management-ui",
    "phase-5-user-entry-dashboard",
    "phase-6-validation-governance-closeout"
  ],
  "pending_batches": [],
  "current_batch": "phase-6-validation-governance-closeout",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
