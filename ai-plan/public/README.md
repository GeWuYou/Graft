# AI Plan Public Recovery Index

`ai-plan/public/README.md` is the shared recovery index used after `AGENTS.md` startup preflight. It should stay short,
list only active topics, and help the current branch or worktree land on the right recovery documents without scanning
every public artifact.

Overlay note:

- archived topic wording is historical evidence, not current normative governance
- if archived wording conflicts with current authority-first rules, follow root `AGENTS.md` and current design docs
- bounded scope continues to forbid unrelated expansion, but never forbids required authority repair

## Active Topics

- `observability-development-governance`
  - Status: `archive-ready`
  - Recovery status: all three phases completed locally with bounded server and web validation; no continuation required inside this topic unless a new bounded follow-up is opened.
  - Goal:
    - define backend observability development standards for app log / audit / security event / metric placeholder
    - inventory and fix bounded logging compliance drift
    - expose existing governance capability to frontend audit/access-control pages
  - Recovery source:
    - `ai-plan/public/observability-development-governance`
  - Authority summary:
    - `server/internal/logger/**` is backend app/error logger authority
    - `server/internal/httpx/**` is request correlation, access log, and HTTP security-event authority
    - `server/internal/audit/**` + `server/plugins/audit/**` are audit persistence and normalization authority
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. Treat observability-development-governance as archive-ready evidence and open a new bounded topic only if observability, metrics-governance, or audit-console follow-up is required.`

## Archived Topics

- `plugin-audit-correlation-governance`
  - Status: `archived`
  - Recovery status: completed the bounded server-only follow-up for plugin-owned domain audit correlation
    propagation; no continuation required unless a new bounded topic expands audit semantics again.
  - Archive reason: closed the `logging-unification-rollout` non-goal by moving plugin-domain audit correlation
    inheritance into the canonical `httpx -> context.Context -> audit plugin` path without introducing a second audit
    context model.
  - Final result:
    - plugin-owned domain audit events now inherit canonical `requestId`, `traceId`, `actorId`, `route`, `method`,
      `clientIp`, and `userAgent` semantics from request context when publishers omit them
    - explicit event payload fields still override inferred request-context values
    - legacy aliases such as `request_id` and `trace_id` remain unchanged in unified audit metadata
    - user/RBAC legacy request-id helper adapters remain preserved for compatibility but are no longer the only
      propagation path
  - Follow-up status: `new-topic-only`
  - Archived topic directory:
    - `ai-plan/public/archive/plugin-audit-correlation-governance`
  - Archive notes:
    - `traceId` still intentionally aliases `requestId` in MVP; no tracing platform was introduced
    - this topic did not widen into audit UI/query expansion, schema changes, or broader plugin refactors
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. Treat plugin-audit-correlation-governance as archive-ready evidence and open a new bounded topic only if audit correlation semantics need further expansion.`

- `logging-governance`
  - Status: `archived`
  - Recovery status: no continuation required; do not resume this topic as an active loop.
  - Archive reason: the read-only cross-boundary logging governance loop completed the planned server inventory,
    frontend inventory, and Batch 3 architecture assessment without widening into runtime refactor work.
  - Final result: the topic records the current zap-based backend baseline, the remaining access/request-id/frontend
    error-capture gaps, and a recommended split between `AppLogger`, `AccessLogger`, `AuditRecorder`, security events,
    and a future `MetricsEmitter`.
  - Follow-up status: `new-topic-only`
  - Archived topic directory:
    - `ai-plan/public/archive/logging-governance`
  - Archive notes:
    - any implementation follow-up should open a new bounded topic instead of reopening this design-only loop
    - preserve `zap` as the backend logging baseline unless future evidence justifies a change
    - keep `audit-plugin-mvp` archived and separate from logging implementation follow-up
    - OpenTelemetry, remote frontend telemetry, and request-log productization remain future scope, not archive
      blockers
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. Treat logging-governance as archived design evidence and open a new bounded implementation topic if logging changes are needed.`

- `logging-unification-rollout`
  - Status: `archived`
  - Recovery status: completed bounded rollout for MVP logging closure; no further continuation required inside this
    topic unless a new bounded follow-up is opened.
  - Archive reason: completed the remaining in-scope logging unification work after archived `logging-governance` and
    archive-ready `request-correlation-access-logging`, then passed bounded backend and web validation.
  - Final result:
    - backend CLI fatal paths now use the shared `zap` logger baseline through `server/internal/logger`
    - Ent debug defaults in owned scope no longer fall back to stdlib `log.Println`
    - security-event metadata now lands with a canonical request/trace/actor/route field dictionary while preserving
      current legacy audit aliases
    - frontend now has shell-owned global error sinks plus default route/request-correlation logger context
    - audit log UI now exposes request/trace troubleshooting ids, copy affordances, reason/source visibility, and URL-shareable filters without inventing fake backend contracts
    - access-control user/role/permission pages now link into related audit records and preserve correlation hints in owned success/error prompts
    - fake runtime audit-risk watch copy was removed from the overview surface; P2 summaries, trends, and timelines remain future scope
  - Follow-up status: `new-topic-only`
  - Archived topic directory:
    - `ai-plan/public/archive/logging-unification-rollout`
  - Archive notes:
    - `traceId` still intentionally aliases `requestId` in MVP; no separate tracing platform was introduced
    - plugin-owned domain audit paths still contain some manual request-id injection outside this topic's allowed
      scope; this remained a documented non-goal for the bounded rollout rather than a reopened authority drift inside
      owned scope
    - metrics stays an explicit future boundary; this topic did not backfill metrics by parsing logs
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. Treat logging-unification-rollout as archive-ready evidence; open a new bounded topic only if plugin-owned domain audit context or a future metrics/tracing platform needs work.`

- `audit-plugin-mvp`
  - Status: `archived`
  - Recovery status: no continuation required; do not restore this topic into the active recovery path.
  - Archive reason: the dedicated audit plugin MVP loop completed its planned backend and frontend batches, recorded
    the Batch 6 and Batch 7 validation evidence, and reached archive-ready closeout without widening into request-log
    products, system-log products, or SOC-style security capabilities.
  - Final result: the topic accepted `Access Log / Request Log -> Audit Policy -> Audit Log` as the MVP model; normal
    request traffic no longer defines the audit dataset by default; persisted audit policy rules now own default
    include/exclude behavior; the frontend audit overview and log copy now describe policy-filtered security audit
    events instead of generic traffic noise.
  - Follow-up status: `future-topic-only`
  - Archived topic directory:
    - `ai-plan/public/archive/audit-plugin-mvp`
  - Archive notes:
    - future request-log or system-log product work should open separate topics instead of reviving this archived audit
      MVP line
    - future audit-policy UI should open a new bounded topic if the repository needs operator-facing rule management
    - regex rule engines, dynamic expressions, risk analytics, geo/IP profiling, and SOC workflows remain intentionally
      out of scope for this archive line
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. If follow-up is needed, open a new topic instead of resuming audit-plugin-mvp.`

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
    - `ai-plan/public/archive/backend-rbac-contract-audit`
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

- `request-correlation-access-logging`
  - Status: `archived`
  - Recovery status: no continuation required; the bounded server-only slice completed and should not remain an active recovery entry.
  - Archive reason: completed the planned request-correlation and structured access-logging implementation, recorded bounded validation, and was absorbed as historical evidence by the later logging rollout closeout.
  - Final result:
    - global request correlation now covers root and plugin HTTP routes through one canonical middleware path
    - Gin default access logging was replaced by structured `zap`-backed access logging
    - access-log severity routing and bounded backend validation evidence were recorded before closeout
  - Follow-up status: `superseded`
  - Superseded by:
    - `logging-unification-rollout`
  - Archived topic directory:
    - `ai-plan/public/archive/request-correlation-access-logging`
  - Archive notes:
    - later logging work should continue as a new bounded topic instead of reopening this completed Phase 1 slice
    - the later `logging-unification-rollout` topic is the cross-boundary closeout line for broader logging follow-up
  - Next-session prompt: `Re-run startup preflight from root AGENTS.md. Open a new bounded topic instead of resuming request-correlation-access-logging.`

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
    - `ai-plan/public/archive/frontend-permission-code-cleanup`
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
    - `ai-plan/public/archive/user-page-permission-governance`
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
    - `ai-plan/public/archive/rbac-visibility-governance`
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
- Worktree: `feat/wt-audit-plugin-mvp`
  - Branch: `feat/wt-audit-plugin-mvp`
  - Active topic: none
  - Archived topic history: `audit-plugin-mvp`
  - Role: retained historical worktree for the archived audit plugin MVP recovery materials; future audit follow-up must
    start as a new topic instead of resuming this worktree as an active recovery entry
  - Hotspot policy: no standing feature ownership; only archived recovery docs under `ai-plan/public/archive/audit-plugin-mvp/**`
    remain as historical evidence
- Worktree: `feat/wt-audit-plugin-mvp`
  - Branch: `feat/logging-governance`
  - Active topic: none
  - Archived topic history: `logging-governance`
  - Role: retained design-only worktree state for the archived logging governance assessment; any implementation
    follow-up must start as a new bounded topic instead of resuming this archived loop
  - Hotspot policy:
    - no standing feature ownership; archived governance evidence remains under `ai-plan/public/archive/logging-governance/**`
      and the temporary assessment output under `temp/logging-governance-assessment.md`
- Worktree: `feat/wt-audit-plugin-mvp`
  - Branch: `feat/logging-unification-rollout`
  - Active topic: none
  - Archived topic history:
    - `logging-governance`
    - `logging-unification-rollout`
  - Recovery dependency:
    - archived `request-correlation-access-logging`
  - Role: retained worktree state for the archive-ready MVP logging closure rollout after request correlation and
    structured access logging landed
  - Hotspot policy:
    - no standing feature ownership; `ai-plan/public/archive/logging-unification-rollout/**` remains historical
      recovery evidence
    - archived governance evidence remains under `ai-plan/public/archive/logging-governance/**`
