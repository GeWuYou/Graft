# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

## Active Topic

- `rbac-visibility-governance`
  - Status: `active`
  - Recovery source:
    - `ai-plan/public/rbac-visibility-governance/README.md`
    - `ai-plan/public/rbac-visibility-governance/todos/rbac-visibility-governance-tracking.md`
    - `ai-plan/public/rbac-visibility-governance/traces/rbac-visibility-governance-trace.md`
    - current RBAC implementation on branch `feat/wt-rbac-further-development`
  - Goal: govern the existing RBAC visibility chain end to end without expanding into menu CRUD or resource CRUD.
  - Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-rbac-further-development`
  - Branch: `feat/wt-rbac-further-development`
  - Owned scope:
    - `ai-plan/public/rbac-visibility-governance/**`
    - `ai-plan/public/README.md`
    - `server/plugins/rbac/**`
    - `server/internal/permission/**`
    - `server/internal/menu/**`
    - `server/internal/httpx/**`
    - `server/plugins/user/bootstrap.go`
    - `web/src/store/modules/permission.ts`
    - `web/src/utils/route/**`
    - `web/src/app/bootstrap/**`
    - `web/src/modules/rbac/**`
    - `web/src/modules/access-control/**`
    - bounded OpenAPI/generated contract files only if required
  - Current phase:
    - initialize governance topic
    - audit the current `permission -> bootstrap menus -> dynamic routes -> element visibility -> API guard` chain
  - Guardrails:
    - do not add menu CRUD
    - do not add resource CRUD
    - do not add a resource table
    - do not move menu canonical truth from registry/bootstrap into database-owned CRUD
  - Next-session prompt:
    - `governance source: root AGENTS.md`
    - `task class: cross-boundary`
    - `recovery source: ai-plan/public/README.md + ai-plan/public/rbac-visibility-governance/README.md + ai-plan/public/rbac-visibility-governance/todos/rbac-visibility-governance-tracking.md + ai-plan/public/rbac-visibility-governance/traces/rbac-visibility-governance-trace.md + current RBAC implementation`
    - `owned scope: ai-plan/public/rbac-visibility-governance/** + ai-plan/public/README.md + server/plugins/rbac/** + server/internal/permission/** + server/internal/menu/** + server/internal/httpx/** + server/plugins/user/bootstrap.go + web/src/store/modules/permission.ts + web/src/utils/route/** + web/src/app/bootstrap/** + web/src/modules/rbac/** + web/src/modules/access-control/** + bounded OpenAPI/generated contract files only if required`

## Archived Topics

- `localization-governance`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: final verification closed the last key-first error rendering gap and confirmed the localization governance baseline is stable enough to leave active recovery.
  - Final result: key-first localization governance is frozen with `messageKey` / `title_key` as canonical contracts, fallback text remains additive compatibility only, and no blocking baseline gaps remain.
  - Follow-up status: `superseded`
  - Superseded by:
    - operating rule `feature-delivery-with-key-first-localization-rule`
  - Archived topic directory:
    - `ai-plan/public/archive/localization-governance`
  - Archive notes:
    - future localization work should run as ordinary feature or contract slices instead of reopening a broad governance topic
    - permission `display_key` remains a future additive enhancement, not a baseline blocker
    - dynamic plugin locale loading remains intentionally deferred; the current static registration model is accepted as its compile-time equivalent
  - Next-session prompt: `No next-session prompt required.`

- `ARCHIVED_OPENAPI_GOVERNANCE_SERIES`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore these topics into the active recovery path.
  - Archive reason: final closeout for the completed OpenAPI / `oapi-codegen` / generated boundary / docs governance series.
  - Final result: implementation, audit, bundled-docs, monitoring-coverage, and closeout topics were either completed, superseded by later closeout topics, or absorbed into the final governance closeout.
  - Follow-up status: `superseded`
  - Superseded by:
    - `ai-plan/public/archive/openapi-governance-closeout-audit/traces/openapi-governance-closeout-audit.md`
    - operating rule `feature-delivery-with-contract-first-rule`
  - Archived topic directories:
    - `ai-plan/public/archive/oapi-codegen-types-only-spike`
    - `ai-plan/public/archive/oapi-generated-server-client-governance-spike`
    - `ai-plan/public/archive/openapi-codegen-governance-audit`
    - `ai-plan/public/archive/openapi-docs-bundled-spec-fix`
    - `ai-plan/public/archive/openapi-docs-mvp`
    - `ai-plan/public/archive/openapi-governance-closeout-audit`
    - `ai-plan/public/archive/openapi-monitoring-coverage-audit`
  - Archive notes:
    - `openapi-codegen-governance-audit` completed its read-first audit mission and was superseded by docs MVP, bundled-spec, generated-boundary, and final closeout work.
    - `openapi-docs-mvp` and `openapi-docs-bundled-spec-fix` completed their docs exposure mission and were absorbed by the final closeout state.
    - `openapi-monitoring-coverage-audit` completed its audit mission and its gap was absorbed by later generated-governance completion work.
    - `oapi-codegen-types-only-spike` and `oapi-generated-server-client-governance-spike` completed their guarded generated-boundary mission and now remain historical evidence only.
  - Operating rule:
    - future HTTP feature work follows `feature-delivery-with-contract-first-rule`
    - do not reopen a broad OpenAPI / `oapi-codegen` governance topic unless contract governance itself changes
  - Next-session prompt: `No next-session prompt required.`
