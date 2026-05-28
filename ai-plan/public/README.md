# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

Overlay note:

- archived topic wording is historical evidence, not current normative governance
- if archived wording conflicts with current authority-first rules, follow root `AGENTS.md` and current design docs
- bounded scope continues to forbid unrelated expansion, but never forbids required authority repair

## Active Topics

- `audit-plugin-mvp`
  - Purpose: hold the dedicated recovery entry for the audit plugin MVP loop spanning `server` and `web`, starting
    from exploration and then closing through bounded implementation batches.
  - Tracking: `ai-plan/public/audit-plugin-mvp/todos/audit-plugin-mvp-tracking.md`
  - Trace: `ai-plan/public/audit-plugin-mvp/traces/audit-plugin-mvp-trace.md`
  - Recovery note: this topic now runs from dedicated worktree
    `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp` on branch `feat/wt-audit-plugin-mvp`; standing
    ownership is centered on `server/plugins/audit/**` and `web/src/modules/audit/**`, while shared-hotspot touches
    remain serialized exceptions. This owned scope guides default responsibility, not canonical authority; if future
    audit drift is traced to upstream authority, escalate and repair there rather than adding downstream compatibility.

## Archived Topics

- `backend-rbac-contract-audit`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: the cross-boundary RBAC contract audit completed all planned batches, passed final backend and web
    validation, and confirmed the current MVP contract closure is stable enough to archive without widening into new
    runtime capability work.
  - Final result: current MVP RBAC scope is `mvp-stable-with-risks`; backend permission registry, backend guards,
    backend menu declarations, frontend permission constants, bootstrap route registrations, and page/action visibility
    remain aligned for the audited `/access-control/*`, role-permission, and user-role surfaces.
  - Follow-up status: `bugfix-only`
  - Archived topic directory:
    - `ai-plan/public/backend-rbac-contract-audit`
  - Archive notes:
    - RBAC no longer takes proactive feature expansion in this topic line; later work should be bugfix-only unless a
      new topic is opened
    - data permission / row-level permission remains a future topic, not a follow-up inside this archive line
    - organization or department permission model remains a future topic, not a follow-up inside this archive line
    - tenant permission model remains a future topic, not a follow-up inside this archive line
    - permission observability or dashboard work remains a future topic, not a follow-up inside this archive line
    - registry and menu closure still rely on canonical ownership plus tests rather than runtime duplicate/reference
      enforcement; this is a non-blocking hardening risk, not a reopen trigger by itself
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. If follow-up is needed, open a new bugfix-only or new-scope topic instead of resuming backend-rbac-contract-audit.`

- `frontend-permission-code-cleanup`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: the frontend-only permission-code cleanup loop completed all planned batches and removed the last
    RBAC symbolic alias drift without widening into backend contract, OpenAPI, or permission-system redesign work.
  - Final result: owned frontend RBAC permission usage now converges on canonical
    `RBAC_PERMISSION_CODE.ROLE_PERMISSION_ASSIGN -> role.permission.assign`; the historical
    `ROLE_PERMISSION_MANAGE` alias is removed from owned scope; RBAC and user page visibility behavior remains
    unchanged because the underlying canonical permission value did not change.
  - Follow-up status: `follow-up-needed`
  - Archived topic directory:
    - `ai-plan/public/frontend-permission-code-cleanup`
  - Archive notes:
    - future backend RBAC contract work should run as a separate cross-boundary topic if canonical permission semantics
      ever need to change
    - future permission observability work should stay a separate frontend or cross-boundary topic instead of
      reopening this cleanup loop
  - Next-session prompt: `No continuation required. Re-run startup preflight from root AGENTS.md before any new topic.`

- `user-page-permission-governance`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: the user-management page permission-governance loop completed all planned batches and removed the
    remaining page-local permission drift without widening into backend, contract, or global UI changes.
  - Final result: user-management action visibility now follows the existing `permission -> v-permission -> runtime
    guard` closure path; permission-only visible-disabled dropdown semantics were removed from the page; privileged
    handlers retain local runtime guards; business-state disabled behavior remains intact.
  - Follow-up status: `follow-up-needed`
  - Archived topic directory:
    - `ai-plan/public/user-page-permission-governance`
  - Archive notes:
    - future frontend permission-code cleanup can remove the `ROLE_PERMISSION_MANAGE` alias if the RBAC module adopts a
      clearer canonical name without changing backend permission values
    - if future user-management behavior needs a permission not expressible by current backend codes, open a separate
      RBAC contract topic instead of reopening this frontend-only governance loop
  - Next-session prompt: `No continuation required.`

- `rbac-visibility-governance`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: the RBAC visibility governance loop completed all planned batches and reached a stable Option A baseline without requiring menu CRUD, resource CRUD, or new backend observability contracts.
  - Final result: the repository now has a governed `permission -> bootstrap menus -> dynamic routes -> element visibility -> API guard` closure path with canonical `/access-control/*` routing, owned-scope `v-permission` coverage improvements, verified backend guard consistency, and a documented decision to keep capability snapshot observability design-only for now.
  - Follow-up status: `superseded`
  - Superseded by:
    - operating rule `feature-delivery-with-existing-rbac-visibility-chain`
  - Archived topic directory:
    - `ai-plan/public/rbac-visibility-governance`
  - Archive notes:
    - future RBAC work should extend the existing visibility chain through ordinary feature or contract slices rather than reopening broad governance
    - any future capability snapshot should stay frontend-only and read-only unless a new bounded slice explicitly introduces a justified cross-boundary observability contract
    - generalized hidden-state `missing permission reason` semantics remain intentionally out of scope until a canonical denial-reason model is designed
  - Next-session prompt: `No next-session prompt required.`

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

## Branch / Worktree To Active Topic Map

- Worktree: repository root
  - Branch: `main`
  - Active topic: none by default
  - Role: shared coordination point for active-topic governance only; feature recovery should enter through an explicit
    startup prompt naming an active topic instead of assuming root carries feature state
  - Hotspot policy: shared hotspots such as `ai-plan/public/README.md` remain serialized governance slices and do not
    grant standing feature ownership to the root worktree
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-audit-plugin-mvp`
  - Branch: `feat/wt-audit-plugin-mvp`
  - Active topic: `audit-plugin-mvp`
  - Role: dedicated audit plugin MVP worktree and recovery entry for slices centered on `server/plugins/audit/**` and
    `web/src/modules/audit/**`
  - Hotspot policy: no standing shared-hotspot ownership; current serialized exception is limited to public recovery
    docs at `ai-plan/public/README.md` and `ai-plan/public/audit-plugin-mvp/**` while the topic baseline is being
    established
