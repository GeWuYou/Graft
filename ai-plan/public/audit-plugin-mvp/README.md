# Audit Plugin MVP

## Status

- Topic: `audit-plugin-mvp`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Archive date: `2026-05-28`

## Goal

- Deliver the audit plugin MVP as a bounded cross-boundary slice instead of widening Audit into a general access-log or
  SOC platform.
- Keep `Request / Access Log`, `Application / System Log`, and `Audit Log` boundaries explicit.
- Move audit inclusion and exclusion policy out of handler or middleware hardcoding and into plugin-owned persisted
  policy rules with bounded evaluator behavior.
- Keep audit overview and audit list surfaces focused on security-relevant events only.

## Final Recovery Summary

- Batch 0 through Batch 5 established the original plugin baseline:
  - backend audit domain and persistence
  - guarded audit read API, permission, menu, and OpenAPI contract closure
  - active audit emission on bounded user and RBAC success paths
  - frontend audit module, route, page, and generated-contract-backed read flow
  - cross-boundary regression on the settled `/audit/logs` closure path
- Batch 6 completed the backend authority repair for audit-policy-governed security logging:
  - added plugin-owned `audit_policy_rules` migration seed authority
  - introduced evaluator-based include and exclude policy decisions
  - removed the need for hardcoded `/healthz`, monitor, bootstrap, and audit-overview path skips in owned middleware
  - confirmed login success, login failure, authorization denial, RBAC change, and sensitive user actions can still
    enter `Audit Log` through candidate evaluation
- Batch 7 completed frontend semantics convergence without widening UI scope:
  - audit overview and audit log copy now explicitly describe security audit events
  - visible guidance now states that monitor polling, `healthz`, and bootstrap noise are excluded from the audit data
    set
  - drawer semantics now use `审计目标` / `目标对象` / `操作对象` terminology instead of generic resource wording
  - no audit-policy management UI was added in this MVP slice
- Final topic result:
  - `Access Log / Request Log -> Audit Policy -> Audit Log` is the accepted MVP model for owned scope
  - ordinary request traffic remains attributable to service-management logging rather than security audit
  - security-sensitive audit events remain bounded to authentication, authorization, RBAC, user-management, export,
    delete, and similar sensitive operations

## Scope

- Recovery docs:
  - `ai-plan/public/audit-plugin-mvp/**`
  - `ai-plan/public/README.md`
- Historical implementation ownership that this topic closed over:
  - `server/plugins/audit/**`
  - `server/internal/audit/**`
  - `server/internal/httpx/**`
  - `server/internal/pluginapi/**`
  - `server/plugins/auth/**`
  - `server/plugins/rbac/**`
  - `server/plugins/user/**`
  - `openapi/**`
  - `web/src/modules/audit/**`

## Batch History

- Batch 0: exploration and worktree/topic setup
- Batch 1: backend audit domain design and schema
- Batch 2: backend API, permission, menu, OpenAPI contract
- Batch 3: backend recording integration for user and RBAC actions
- Batch 4: frontend audit module and page
- Batch 5: cross-boundary integration and regression
- Batch 6: backend audit policy boundary convergence
- Batch 7: frontend audit semantics alignment
- Batch 8: topic closeout and archive-readiness

## Final Archive Record

- Status: `archived`
- Archive-ready decision: `archive-ready`
- Archive reason:
  - the planned bounded batches are complete
  - backend and frontend validation evidence for the extended topic is recorded
  - no remaining owned-scope blocker requires reopening runtime implementation just to state closeout truth honestly
- Final result:
  - `Request / Access Log`, `Application / System Log`, and `Audit Log` boundaries are explicitly separated in the
    topic outcome
  - ordinary requests such as `/healthz`, monitor polling, bootstrap loads, and audit overview reads no longer define
    the audit dataset by default
  - default audit include and exclude behavior is carried by plugin-owned SQL seed data plus evaluator logic instead of
    path-based hardcoding in request handlers or middleware
  - audit overview and audit log surfaces now describe and display security audit events rather than mixed traffic noise
- Final validations:
  - Batch 6 backend:
    - `cd server && go test ./plugins/audit/... ./internal/audit/... ./internal/httpx/...`
    - `cd server && go run ./cmd/graft validate backend`
  - Batch 7 frontend:
    - `cd web && bun run test:run src/modules/audit/pages/overview/index.test.ts src/modules/audit/pages/logs/index.test.ts`
    - `cd web && bun run check`
- Follow-up policy:
  - no continuation is required for this topic
  - later work should open a new topic if it needs audit-policy UI, request-log or system-log products, or broader
    security analytics
  - do not reopen this archived MVP line to build regex rule engines, SOC workflows, geo/IP risk profiling, or real-time alerting
- Remaining risks:
  - the current audit-policy evaluator is intentionally MVP-bounded to `enabled`, `priority`, `include|exclude`,
    `exact|prefix`, and method matching; more expressive rule systems remain future work
  - request-log and system-log products are still planning space outside this archive line and should not be inferred as
    completed just because Audit now has a clearer boundary
- Continuation:
  - no continuation required for this topic
