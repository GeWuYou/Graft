# OpenAPI Monitoring Coverage Audit Tracking

## Status

- Topic: `openapi-monitoring-coverage-audit`
- State: audit completed
- Contract backfill: not started
- Commit required this round: yes, if doc-only validation passes

## This Round

1. 完成启动预检与治理读取。
2. 盘点 `server` 监控相关 route 与 core 状态/docs route。
3. 盘点 `web` 监控模块页面、共享 snapshot 与 API 调用。
4. 对比 `openapi/openapi.yaml` 与 `openapi/paths/**` 覆盖现状。
5. 记录“缺 OpenAPI 覆盖，不缺真实后端接口”的结论。

## Decision

- 是否确认缺少监控接口：否，后端 route 已存在。
- 是否确认缺少监控 OpenAPI 覆盖：是。
- 本轮是否补齐契约：否。
- 不补齐原因：本轮优先审计；已能给出最小补齐方案，但不在本轮直接修改 `openapi/**`。

## Validation Plan

本轮只做审计文档修改，最小验证执行：

- `git diff --check`
- `cd server && go run ./cmd/graft validate backend --stage openapi`
- `cd web && bun run openapi:types:check`

## Risks / Blockers

- 监控接口响应结构较大，下一轮补契约时需要控制 schema 拆分粒度，避免一次性引入过多无关生成改动。
- `openapi/dist/openapi.bundle.json` 是入库产物；若下一轮改 `openapi/**`，需要同步检查 bundle freshness。

## Next Suggestion

下一轮建议采用最小切片：

1. 仅为 `GET /api/monitor/server-status` 补 `monitor` tag、path 和必要 schema。
2. 仅在 `openapi/**` 与本 topic 文档范围内改动。
3. 运行：
   - `bun scripts/openapi-bundle.mjs` 或仓库等价 bundle 命令
   - `cd web && bun run openapi:types:check`
   - `cd server && go run ./cmd/graft validate backend --stage openapi`
   - `git diff --check`

## Next-Session Startup Prompt

`governance source: root AGENTS.md`

`task class: cross-boundary`

`recovery source: current repository state + ai-plan/public/openapi-docs-mvp + ai-plan/public/openapi-docs-bundled-spec-fix + ai-plan/public/openapi-monitoring-coverage-audit`

`owned scope: ai-plan/public/openapi-monitoring-coverage-audit/** + openapi/**`

`forbidden scope: server/internal/ent/**, migrations, business behavior changes, frontend page refactor, broad generated changes, unrelated formatting`

`goal: backfill the existing GET /api/monitor/server-status contract into OpenAPI without changing handler/service behavior`
