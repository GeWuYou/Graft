# Audit Plugin MVP Tracking

## Topic

- Topic: `audit-plugin-mvp`
- Status: `archived`
- Goal: deliver and close the audit plugin MVP as a bounded cross-boundary slice with explicit logging boundaries and
  policy-governed security audit inclusion.
- Recovery source:
  - `ai-plan/public/README.md`
  - `ai-plan/public/audit-plugin-mvp/README.md`
  - `ai-plan/public/audit-plugin-mvp/todos/audit-plugin-mvp-tracking.md`
  - `ai-plan/public/audit-plugin-mvp/traces/audit-plugin-mvp-trace.md`
  - archived `ai-plan/public/archive/backend-rbac-contract-audit`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/wt-audit-plugin-mvp`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Scope

- Owned recovery-doc scope:
  - `ai-plan/public/audit-plugin-mvp/**`
  - `ai-plan/public/README.md`
- Historical implementation scope closed by this topic:
  - `server/plugins/audit/**`
  - `server/internal/audit/**`
  - `server/internal/httpx/**`
  - `server/internal/pluginapi/**`
  - `server/plugins/auth/**`
  - `server/plugins/rbac/**`
  - `server/plugins/user/**`
  - `openapi/**`
  - `web/src/modules/audit/**`
- Forbidden continuation scope:
  - reopening runtime implementation without a new topic
  - broad request-log or system-log product work inside this archived line
  - regex or expression-based policy engines
  - SOC, real-time alerting, geo/IP risk analysis, or other out-of-MVP security expansion

## Startup Receipt

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: `subtopic`
  - `ai-plan/public/README.md`
  - `ai-plan/public/audit-plugin-mvp/README.md`
  - `ai-plan/public/audit-plugin-mvp/todos/audit-plugin-mvp-tracking.md`
  - `ai-plan/public/audit-plugin-mvp/traces/audit-plugin-mvp-trace.md`
- Authority summary:
  - canonical authority for this closeout round is the topic recovery documentation itself
  - runtime truth for the completed implementation remained in the settled `server` and `web` owned slices validated in
    prior batches

## Current Recovery Point

- Batch 8 closeout completed.
- The topic is archive-ready and no further same-session batches remain.
- Topic docs now reflect the real extended history:
  - Batch 6 backend audit policy boundary convergence completed and validated
  - Batch 7 frontend audit semantics alignment completed and validated
  - Batch 8 archive-readiness review completed without reopening implementation

## Batch State

- Current batch: `Batch 8 - Topic closeout and archive-readiness`
- Completed batches:
  - `Batch 0 - Exploration and worktree/topic setup`
  - `Batch 1 - Backend audit domain design and schema`
  - `Batch 2 - Backend API, permission, menu, OpenAPI contract`
  - `Batch 3 - Backend recording integration for user and RBAC actions`
  - `Batch 4 - Frontend audit module and page`
  - `Batch 5 - Cross-boundary integration and regression`
  - `Batch 6 - Backend audit policy boundary convergence`
  - `Batch 7 - Frontend audit semantics alignment`
  - `Batch 8 - Topic closeout and archive-readiness`
- Pending batches:
  - none

## Final Topic Outcome

- Accepted MVP logging boundary:
  - `Access Log / Request Log -> Audit Policy -> Audit Log`
- Accepted scope split:
  - ordinary request traffic belongs to service-management logging
  - internal runtime and background behavior belongs to application or system logging
  - only security-relevant or sensitive business events belong to audit logging
- Accepted audit-policy governance:
  - default include and exclude rules are seeded through plugin-owned SQL migration data
  - owned middleware and handlers no longer need path-based hardcoded skip rules for `healthz`, monitor polling,
    bootstrap, or audit overview reads
  - candidate events from request flow, auth, and authorization still converge through evaluator-based final policy
    decision
- Accepted frontend semantics:
  - audit overview and audit logs now describe security audit events rather than generic traffic
  - audit drawer language uses `审计目标` / `目标对象` / `操作对象`
  - no audit-policy UI was added in this MVP line

## Validation Record

- Batch 6 backend validation:
  - `cd server && go test ./plugins/audit/... ./internal/audit/... ./internal/httpx/...`
  - `cd server && go run ./cmd/graft validate backend`
- Batch 7 frontend validation:
  - `cd web && bun run test:run src/modules/audit/pages/overview/index.test.ts src/modules/audit/pages/logs/index.test.ts`
  - `cd web && bun run check`
- Batch 8 closeout validation:
  - docs-only review of the recorded Batch 6 and Batch 7 evidence
  - `git diff --check` should be run by the integrating orchestrator if needed for the final docs diff

## Follow-up Policy

- No continuation required for this topic.
- Future work should open a new topic instead of reviving this archive if it needs:
  - audit-policy management UI
  - dedicated request-log product surfaces
  - dedicated system-log product surfaces
  - more expressive policy matching such as regex or condition expressions
  - risk scoring, anomaly detection, or SOC-style workflows

## Commit Plan

- Batch 8:
  - `docs(audit-plugin-mvp): archive audit policy topic`

## Final Decision

- Final closeout status: `archive-ready`
- Decision basis:
  - the planned implementation batches are complete
  - Batch 6 and Batch 7 validation evidence is explicit and internally consistent
  - the remaining notes are future-scope product or rule-engine work, not blockers to MVP archive readiness
- Remaining risks:
  - audit policy remains intentionally limited to a small rule vocabulary
  - request-log and system-log productization remains out of scope for this archived topic
