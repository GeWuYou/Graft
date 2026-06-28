# Logging Unification Rollout

## Status

- Topic: `logging-unification-rollout`
- Status: `archived`
- Loop mode: `topic-completion-loop`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/logging-unification-rollout`
- Task class: `cross-boundary`
- Started: `2026-05-29`
- Closed: `2026-05-29`

## Goal

- close the remaining MVP logging-governance rollout work without reopening archived `logging-governance`
- keep `zap` as the only backend `AppLogger` backend and preserve the existing structured `AccessLogger` path
- align `SecurityEvent -> AuditRecorder` field semantics for request, trace, actor, route, plugin, and risk context
- add frontend global error sinks plus default route/request correlation context to the local structured logger
- leave `MetricsEmitter` as an explicit future boundary instead of implementing fake metrics from logs

## Recovery Inputs

- `ai-plan/public/README.md`
- archived `ai-plan/public/archive/logging-governance/**`
- archived `ai-plan/public/archive/request-correlation-access-logging/**`
- `temp/logging-governance-assessment.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`

## Scope

- Owned scope:
  - `ai-plan/public/archive/logging-unification-rollout/**`
  - `ai-plan/public/README.md`
  - `server/internal/logger/**`
  - `server/internal/httpx/**`
  - `server/internal/audit/**`
  - `server/cmd/graft/**`
  - `server/cmd/graft-jwt-secret/**`
  - `server/cmd/graft-signing-key/**`
  - `server/plugins/user/ent/**`
  - `server/plugins/rbac/ent/**`
  - `web/src/utils/logger/**`
  - `web/src/utils/request.ts`
  - `web/src/app/**`
  - bounded tests in directly affected packages
- Forbidden scope:
  - reopening archived `logging-governance`
  - keeping `request-correlation-access-logging` as the active topic
  - OpenTelemetry, Prometheus, exporter, or remote logging SaaS rollout
  - generic `Log(type, ...)`, event-bus-like logging abstraction, or audit UI product expansion
  - compatibility-layer patches that leave upstream authority drift unresolved

## Acceptance Targets

- backend app logging stays on `zap`, including CLI fatal paths and Ent debug output
- backend request correlation plus structured access logging remains the only long-term `AccessLogger` path
- security events and audit candidates share one stable field dictionary for request and actor semantics
- frontend logger gains global exception capture and automatic route/request correlation context
- closeout clearly records what is now closed, and what remains an intentional MVP non-goal

## Closeout

- Completed batches:
  - `batch-1-topic-open-and-authority-recheck`
  - `batch-2-backend-cli-and-ent-log-unification`
  - `batch-3-security-event-and-audit-correlation-alignment`
  - `batch-4-frontend-global-error-sinks-and-trace-context`
  - `batch-5-cross-boundary-validation-and-regression-audit`
  - `batch-6-closeout-archive-and-commit-evaluation`
- Validation:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./plugins/user/ent ./plugins/rbac/ent`
  - `cd web && bun run check`
- Accepted non-goals:
  - `traceId` continues to alias `requestId` in MVP
  - no remote logging, tracing, or metrics platform rollout
  - plugin-owned domain-event request-id adapters outside the allowed scope remain documented follow-up material, not a
    reopened blocker for this bounded rollout
