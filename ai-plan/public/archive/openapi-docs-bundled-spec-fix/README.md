# OpenAPI Docs Bundled Spec Fix

## Topic

- Topic: `openapi-docs-bundled-spec-fix`
- Task class: `cross-boundary`
- Branch: `feat/wt-oapi-codegen-types-only-spike`
- Recovery source: current repository state + `ai-plan/public/openapi-docs-mvp` + commit `65f786c`

## Goal

修复 `GET /openapi.json`，让 `GET /docs` 读取到 Scalar 可直接消费的 bundled OpenAPI 文档，而不是仍然保留外部文件 `$ref` 的 root spec JSON 化结果。

## Scope

- `ai-plan/public/openapi-docs-mvp/**`
- `ai-plan/public/openapi-docs-bundled-spec-fix/**`
- `openapi/**`
- `scripts/**`
- `server/internal/app/**`

## Root Cause

- `/openapi.json` 之前直接来自 `openapi/openapi.yaml` 的加载结果再 `MarshalJSON()`。
- 根文档仍然保留 `./paths/**`、`./components/**` 外部文件 `$ref`。
- `/docs` 使用 Scalar CDN 在浏览器端读取 `/openapi.json`，但 server 没有暴露这些 fragment 文件路径，所以 Scalar 只能显示 tags / models，无法展开具体 operation。

## Fix Shape

- 保留 `openapi/openapi.yaml` 作为拆分源文档。
- 新增 checked-in artifact：`openapi/dist/openapi.bundle.json`。
- 新增生成命令：`bun scripts/openapi-bundle.mjs`。
- `server/internal/app/openapi_docs.go` 改为：
  - 继续读取原始 `openapi/openapi.yaml` 供 `/openapi.yaml`
  - 读取并校验 `openapi/dist/openapi.bundle.json` 供 `/openapi.json`
- `/docs` 继续读取 `/openapi.json`
- `/docs` 路由继续保持在 root，不进入 `/api/**`

## Risks

- 当前 bundle artifact 是 checked-in 生成物，仍然存在 stale 风险。
- 本轮只修复 docs 可用性，不把 bundle stale gate 接入现有 CI / hook。

## Next Suggestion

1. 增加 `openapi:bundle:check` 或等价 stale gate。
2. 把 web `openapi:types:check` 与 Go `oapi-codegen` freshness gate 明确进入 CI。

## Final Archive Record

- Status: `archived`
- Archive series: `ARCHIVED_OPENAPI_GOVERNANCE_SERIES`
- Archive reason: the bundled docs spec fix completed its mission and was absorbed into the final OpenAPI docs and governance closeout state.
- Final result: the topic remains as evidence for the bundled `/openapi.json` docs-serving fix, not as an active recovery line.
- Final validation summary:
  - docs-only archive update in this round
  - no server / web / openapi runtime files changed
  - backend and web validation not run in this round because the archive slice is documentation-only
- Follow-up status: `superseded`
- Superseded by:
  - `openapi-governance-closeout-audit`
- Next-session prompt: `No continuation required.`
- Archive note:
  - This topic is archived as part of the final OpenAPI/OAPI governance closeout. The relevant implementation, audit, generated-boundary, bundled-docs, monitor-coverage, and closeout findings have either been completed, superseded by later governance topics, or converted into guardrail-oriented follow-up guidance. No next-session recovery is required for this topic.
