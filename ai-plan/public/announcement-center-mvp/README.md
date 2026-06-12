# Announcement Center MVP

## 当前状态摘要

- 当前主题目标是在 `Graft` 增加公告中心能力，覆盖管理端公告发布和用户侧公告阅读。
- 状态：`archive-ready`。
- 任务分类为 `cross-boundary`，涉及 OpenAPI、server module、migration、RBAC/menu、web module、shell/global route 和 i18n。
- Canonical design：`ai-plan/design/公告中心设计.md`。
- 公告中心 MVP 已完成 Phase 0 到 Phase 6；后续不应在本主题下继续追加功能行为。

## Recovery Receipt

- governance source：root `AGENTS.md`
- task class：`cross-boundary`
- recovery source：`parent topic`
- authority summary：OpenAPI source + `server/modules/announcement` module contract/descriptor + `web/src/modules/announcement` bootstrap routes + `ai-plan/design/公告中心设计.md`

## Owned Scope

允许修改：

- `ai-plan/design/公告中心设计.md`
- `ai-plan/public/announcement-center-mvp/**`
- `ai-plan/public/README.md`
- `openapi/**`
- `server/modules/announcement/**`
- `server/internal/moduleregistry/generated.go`
- 必要的 backend module registry / migration registry 接入文件
- `web/src/modules/announcement/**`
- 必要的 `web` route/menu/i18n/module aggregation 文件
- 必要的 shell/header/dashboard integration 文件

禁止误触：

- 不得修改 `server/modules/notification/**`，除非用户明确授权公告与通知联动。
- 不得把公告正文写入 `notification_events`。
- 不得回退当前工作树里已有的 notification 未提交改动。

## Phase Plan

- Phase 0：设计和 public topic 持久化。
- Phase 1：OpenAPI、migration、后端模块骨架。
- Phase 2：后端管理端 API。
- Phase 3：后端用户端 API 和已读状态。
- Phase 4：前端公告管理页。
- Phase 5：用户侧公告入口、未读 badge、可选工作台摘要。
- Phase 6：测试、i18n、治理收尾、归档准备。

## Current Recovery Point

- 分支为 `feat/announcement-center-mvp`。
- 设计 authority 已落到 `ai-plan/design/公告中心设计.md`。
- Phase 6 最终验证和治理收尾已完成。
- Archive-ready 判定：`confirmed`。
- 无后续同主题批次；未来公告功能扩展应开启新的 bounded topic。

## Final Closeout

- 完成范围：
  - OpenAPI 公告管理端和当前用户端契约。
  - `server/modules/announcement` 模块、migration、权限菜单注册、管理端 API、当前用户 API 和已读状态。
  - `web/src/modules/announcement` 管理页、当前用户公告页、global route、顶部独立公告入口和 unread badge。
- 独立性确认：
  - MVP 不做 notification fan-out。
  - 公告正文只在公告领域模型中持久化，不写入 `notification_events`。
  - 顶部公告入口与 notification bell 分别由 `AnnouncementHeaderEntry` 和 `NotificationBellPanel` 装配。
- Dashboard 摘要延期：
  - 当前 dashboard 由后端 summary/widgets 驱动，缺少干净的前端模块贡献点。
  - 公告摘要应等待 dashboard contribution contract，而不是在工作台页面硬编码模块聚合。

## Validation Targets

```bash
cd server && go run ./cmd/graft validate backend
python3 scripts/validate_sql_migrations.py
python3 scripts/check_migration_versions.py --mode all
cd web && bun run check
git diff --check
```
