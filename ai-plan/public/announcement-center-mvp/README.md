# Announcement Center MVP

## 当前状态摘要

- 当前主题目标是在 `Graft` 增加公告中心能力，覆盖管理端公告发布和用户侧公告阅读。
- 任务分类为 `cross-boundary`，涉及 OpenAPI、server module、migration、RBAC/menu、web module、shell/global route 和 i18n。
- Canonical design：`ai-plan/design/公告中心设计.md`。
- 当前工作树存在与本主题无关的 notification 未提交改动，公告中心实现不得回退、覆盖或纳入这些改动。

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

- 分支已重命名为 `feat/announcement-center-mvp`。
- 设计 authority 已落到 `ai-plan/design/公告中心设计.md`。
- 当前主题恢复入口为本文档。
- 下一步进入 Phase 1，先建立 OpenAPI 和后端公告模块基础。

## Validation Targets

```bash
cd server && go run ./cmd/graft validate backend
cd server && go test ./modules/announcement/...
python3 scripts/validate_sql_migrations.py
python3 scripts/check_migration_versions.py --mode all
cd web && bun run check
cd web && bun run lint:i18n
cd web && bun run openapi:types:check
```
