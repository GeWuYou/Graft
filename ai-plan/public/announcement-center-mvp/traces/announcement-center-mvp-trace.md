# Announcement Center MVP Trace

## 2026-06-12

- 启动任务并确认 task class 为 `cross-boundary`。
- 当前分支从 `build/web-tdesign-on-demand-imports` 重命名为 `feat/announcement-center-mvp`。
- 建立公告中心设计 authority：`ai-plan/design/公告中心设计.md`。
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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-0-design-topic-persistence",
    "phase-1-openapi-server-foundation",
    "phase-2-server-management-api",
    "phase-3-server-user-api"
  ],
  "pending_batches": [
    "phase-4-web-management-ui",
    "phase-5-user-entry-dashboard",
    "phase-6-validation-governance-closeout"
  ],
  "current_batch": null,
  "next_batch": "phase-4-web-management-ui"
}
```
