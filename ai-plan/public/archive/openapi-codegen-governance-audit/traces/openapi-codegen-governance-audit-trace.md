# OpenAPI Codegen Governance Audit Trace

## 2026-05-24 audit closeout

- Completed a cross-boundary audit of the current OpenAPI / generated-type / validation / docs-page state without modifying runtime business code.
- Confirmed that this branch no longer matches the older simplified statement "oapi-codegen is deferred":
  - a constrained Go `types-only` chain now exists under `server/internal/contract/openapi/**`
  - the current rollout still does not generate server interfaces, strict server stubs, or runtime clients
- Confirmed the current split-spec structure:
  - root spec at `openapi/openapi.yaml`
  - route fragments under `openapi/paths/**`
  - reusable schemas/responses/security/parameters under `openapi/components/**`
- Confirmed shared envelope/error baseline is present and aligned with current runtime semantics:
  - `ApiEnvelope`
  - `ErrorResponse`
  - `messageKey`
  - `locale`
  - `traceId`
  - `data.field`
- Confirmed backend toolchain status:
  - `oapi-codegen` config exists at `server/internal/contract/openapi/oapi-codegen.yaml`
  - `go generate` entrypoint exists at `server/internal/contract/openapi/generate.go`
  - generated output is checked in under `server/internal/contract/openapi/generated/types.gen.go`
  - current output mode is models/types only
  - no generated-Go stale gate exists in validate, hooks, or CI
- Confirmed frontend toolchain status:
  - `openapi-typescript` generation and compare-check exist in `web/package.json`
  - tracked output lives at `web/src/contracts/openapi/generated/schema.ts`
  - `bun run check` does not explicitly include `openapi:types:check`
- Confirmed backend generated-type adoption status:
  - non-`auth` `user` and `rbac` write routes already bind generated request body types
  - `auth` write routes still bind handwritten DTOs
- Confirmed frontend generated-type adoption status:
  - `auth`, `user`, and `rbac` consume generated schema aliases at API boundaries
  - `web/src/utils/request.ts` remains the only transport/runtime truth
  - generated payload types are not being used as long-lived page form truth by default
- Estimated current OpenAPI governance maturity at `50-75%`, best summarized as `about 60-70%`.
- Identified the highest-value next slice as a docs-page MVP loop rather than another DTO migration wave.

## Recommended Docs MVP

- Prefer server-served docs over web-shell embedding for the first implementation.
- Suggested routes:
  - `GET /openapi.json`
  - `GET /docs`
- Suggested exposure policy:
  - dev/test on by default
  - prod off by default or admin-guarded
- Suggested rendering model:
  - Go server returns minimal HTML
  - HTML loads `Scalar` or `Swagger UI` from CDN
  - no new repository dependency introduction

## Validation Evidence

- Lightweight audit commands intended for this slice:
  - `git diff --check`
  - `git status --short`
  - `cd web && bun run openapi:types:check`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
- Heavy validations intentionally left for follow-up implementation slices:
  - `cd web && bun run check`
  - `cd server && go test ./...`
  - `cd server && go run ./cmd/graft validate backend`

## Next-Session Startup Prompt

```text
使用 $graft-multi-agent-task。

governance source: root AGENTS.md
task class: cross-boundary
recovery source: current repository state + ai-plan/public
repository root: 当前工作树
topic: openapi-docs-mvp
owned scope:
  - ai-plan/public/openapi-docs-mvp/**
  - server/internal/app/**
  - server/internal/httpx/** only if required by the docs route wiring
  - server/internal/config/** only if required by a minimal docs exposure config
  - server/cmd/graft/** only if validation/help text must mention the new docs route
  - openapi/** read-only unless a tiny spec-exposure adjustment is strictly necessary
forbidden scope:
  - 不得修改业务 handler/service/repository 实现
  - 不得修改数据库 schema/migration
  - 不得批量迁移 DTO
  - 不得引入新的 Go/Bun 依赖
  - 不得接入 web 前端菜单或大范围前端页面
objective:
  为当前 Graft 项目实现“项目启动后可通过端口访问 OpenAPI 文档页面”的最小闭环：
  1. 提供 GET /openapi.json
  2. 提供 GET /docs
  3. dev/test 默认开启，prod 默认关闭或受限
  4. 不影响现有 API route，不引入 Swagger UI/Scalar/Redoc 包依赖，优先使用 server 直接返回最小 HTML + CDN 资源
validation:
  - git diff --check
  - cd server && go test ./...
  - cd server && go run ./cmd/graft validate backend
  - 若新增配置分支，补最小 focused test
deliverables:
  - 最小 docs 访问闭环代码
  - 风险说明
  - 下一步是否需要把 docs 链接接入 web 菜单的判断
```
