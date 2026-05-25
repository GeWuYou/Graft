# OpenAPI Docs MVP Tracking

## Topic

- Topic: `openapi-docs-mvp`
- Status: `implementation complete, validation pending`
- Branch: `feat/wt-oapi-codegen-types-only-spike`

## Done

- 增加 `server` core-level OpenAPI docs/spec 路由
- 增加 docs 暴露配置策略
- 增加 focused tests
- 增加登录后和登录页顶部文档按钮
- 更新 GitHub 链接目标
- 补齐 `ai-plan/public/openapi-docs-mvp/**` 恢复文档

## Validation Pending

- `git diff --check`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check`

## Deferred

- prod 下的更细粒度访问控制
- docs 页面可见文案与国际化
- 是否在后续接入 `web` 菜单
- `/openapi.json` bundled artifact stale gate
