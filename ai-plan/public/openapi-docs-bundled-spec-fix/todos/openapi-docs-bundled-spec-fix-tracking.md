# OpenAPI Docs Bundled Spec Fix Tracking

## Topic

- Topic: `openapi-docs-bundled-spec-fix`
- Status: `complete`
- Branch: `feat/wt-oapi-codegen-types-only-spike`

## Done

- 复现 `/docs` 只显示 tags / models 的问题
- 确认 `/openapi.json` 之前暴露的是 unbundled root spec
- 新增 `openapi/dist/openapi.bundle.json`
- 新增 `bun scripts/openapi-bundle.mjs`
- `server` 改为读取 bundled JSON
- focused tests 覆盖 operation 可见性与外部 ref 缺失
- 补齐本 topic 恢复文档

## Validation Completed

- `git diff --check`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`

## Deferred

- `openapi bundle` stale gate
- web `openapi:types:check` 纳入 CI
- Go `oapi-codegen` freshness gate
- docs 页面内容美化与 tag description 完善
