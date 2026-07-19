# Cross-Boundary Contract Projection

## 当前状态摘要

- 目标：以 server 为 canonical authority，建立从现有 Go contract 与 OpenAPI 到 web TypeScript 的统一跨边界契约投影，消除 web 对 error code、message key、permission、enum、capability 和 feature/config key 的手工镜像。
- 当前状态：`active`；已完成 Work Intake、生成器基础与平台兼容层 pilot。
- 任务分类：`cross-boundary` 实施。
- Canonical authority：`openapi/**` 拥有 HTTP wire contract；`server/internal/contract/**`、`server/modules/*/contract/**`、module descriptor 与 `server/internal/moduleapi/**` 拥有非 HTTP server contract；web 仅为 derived consumer。

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: OpenAPI owns wire semantics; Go server contracts own non-HTTP values; generated web artifacts never become authority.

## Owned Scope

- `ai-plan/design/governance/platform/跨边界契约投影设计.md`
- `ai-plan/roadmap/跨边界契约投影实施计划.md`
- future generator, OpenAPI, server contract, web generated-consumer and CI slices required by the approved design

Out of scope:

- protobuf、共享运行时 package 或第二份手写 IDL
- 由 web 维护任何 server contract 的长期副本

## Locked Decisions

1. OpenAPI remains canonical for HTTP paths, wire schemas and public wire enums; Go contracts remain canonical for non-HTTP values.
2. Projection metadata references existing Go constants; `visibility=web` is mandatory before a non-HTTP value enters web artifacts.
3. Error code and message key remain open strings at API boundaries with fallback; runtime menu/permission/capability remains server authority.

## Current Recovery Point

- Platform pilot 已验证并提交：web API compatibility exports 已改为 generated platform 的薄 re-export，未改变 API wire envelope。
- Next step: wire the existing projection freshness check into the required CI paths.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in `todos/cross-boundary-contract-projection-tracking.md`.

## Pending Batch Direction

- CI integration
- broader migration and final archive readiness
