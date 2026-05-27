# Localization Governance Tracking

## Topic

- Topic: `localization-governance`
- Status: `closeout-ready`
- Goal: close the cross-boundary localization governance baseline after verifying the key-first runtime path and recording the remaining additive-only follow-ups.
- Recovery source: `none`
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-localization-governance`
- Branch: `feat/wt-localization-governance`

## Scope

- Owned scope:
  - `server/internal/i18n/**`
  - `server/internal/httpx/**`
  - `server/internal/contract/**`
  - `server/plugins/**` when changing localization contract registration or error/message key ownership
  - `web/src/locales/**`
  - `web/src/modules/**` when changing key-first locale consumption
  - `web/src/contracts/**`
  - `web/src/utils/request.ts`
  - `web/src/utils/route/**`
  - `openapi/**` when aligning key-field semantics only
  - `ai-plan/public/**`
  - related design docs when governance truth changes
- Task class: `cross-boundary`

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`

## Current Recovery Point

- The prior worktree and branch name were still tied to the archived OpenAPI governance topic and were no longer valid recovery truth for this task.
- The active implementation workspace has now been renamed to:
  - Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-localization-governance`
  - Branch: `feat/wt-localization-governance`
- This topic is the new active recovery entry for the project-wide localization governance task.
- Current governance decisions frozen by the completed audit:
  - locale sources must not be assumed to come only from the host app source tree
  - locale keys require owner namespaces
  - `menu title_key`, `permission display_key`, and error `messageKey` are stable key contracts
  - frontend display must consume `key + fallback`
  - backend registry is registration/validation/fallback only, not a UI copy center
  - OpenAPI describes key fields and semantics only, not multilingual copy
  - current static plugin registration is the compile-time equivalent of a future dynamic plugin register API

## Audit Conclusion

- No material gap was found that justified a runtime code change inside the bounded slice.
- Frontend error rendering in the scanned `web/src/modules/**`, `web/src/app/**`, and shared helper path already routes API-localized failures through `messageKey + message fallback`:
  - `web/src/modules/shared/localized-api-error.ts`
  - `web/src/app/bootstrap/route-guards.ts`
  - `web/src/modules/auth/pages/components/Login.vue`
  - `web/src/modules/user/pages/index.vue`
  - `web/src/modules/rbac/pages/index.vue`
  - `web/src/modules/rbac/pages/permissions/index.vue`
  - `web/src/modules/monitor/pages/overview/index.vue`
- Dynamic menu and route title handling already treats `title_key` as canonical and `title` as fallback:
  - `web/src/utils/route/bootstrap.ts`
  - `web/src/utils/route/title.ts`
  - `web/src/store/modules/permission.ts`
- Breadcrumb and tab title rendering consume localized route meta produced by the bootstrap transformer rather than reading backend `title` as long-term truth:
  - `web/src/layouts/components/Breadcrumb.vue`
  - `web/src/layouts/index.vue`
- Backend message registration already uses owner namespaces and freeze-aware registration:
  - `server/internal/i18n/service.go`
  - `server/plugins/user/plugin_registration.go`
  - `server/plugins/rbac/plugin_registration.go`
  - `server/plugins/monitor/plugin.go`
- Current permission `display` / `description` fields remain fallback-only; no `display_key` contract exists yet, and that remains an additive future contract instead of a missing baseline fix:
  - `server/plugins/rbac/README.md`
- OpenAPI remains limited to key-field semantics and fallback/locale behavior:
  - `openapi/components/schemas/error-response.yaml`
  - `openapi/components/schemas/api-envelope.yaml`
  - `openapi/components/schemas/bootstrap-menu.yaml`

## Shared Hotspots

- `ai-plan/public/README.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/design/前端架构设计.md`
- `ai-plan/design/项目设计.md`
- `server/internal/contract/**`
- `server/internal/i18n/**`
- `web/src/locales/**`
- `web/src/contracts/**`

## Remaining Follow-up

- Optional future cleanup only if a new slice explicitly owns it:
  - add registry-level diagnostics for `server/internal/menu` if the project later wants duplicate-path or missing-`title_key` enforcement at registration time
  - evolve permission payloads additively with `display_key` only when a stable frontend-facing permission copy contract is actually needed
  - remove the current access-control compatibility normalization in `web/src/utils/route/bootstrap.ts` only after backend menu hierarchy and `title_key` payloads become fully canonical for that subtree
