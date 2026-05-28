# Logging Unification Rollout Tracking

## Topic

- Topic: `logging-unification-rollout`
- Status: `archive-ready`
- Goal:
  - unify the remaining MVP logging surfaces after request correlation and structured access logging landed
  - keep backend app logging on `zap`
  - close frontend global exception capture and default correlation context
- Recovery source:
  - `ai-plan/public/README.md`
  - archived `logging-governance`
  - archive-ready `request-correlation-access-logging`
  - `temp/logging-governance-assessment.md`
- Worktree: `feat/wt-audit-plugin-mvp`
- Branch: `feat/logging-unification-rollout`
- Task class: `cross-boundary`
- Loop mode: `topic-completion-loop`

## Startup Receipt

- Governance source: `root AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: `archived logging-governance evidence + archive-ready request-correlation-access-logging`
- Authority summary:
  - `server/internal/logger/**` is the backend `AppLogger` authority
  - `server/internal/httpx/**` is the request correlation, access logging, and security-event bridge authority
  - `server/internal/audit/**` plus `server/plugins/audit/**` own audit persistence semantics
  - `web/src/utils/logger/**`, `web/src/utils/request.ts`, and `web/src/app/**` own frontend logger context and
    global exception capture

## Scope

- Owned scope:
  - `ai-plan/public/logging-unification-rollout/**`
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
  - keeping `request-correlation-access-logging` active
  - OpenTelemetry, metrics-platform rollout, remote frontend telemetry, or generic logging abstraction work

## Batch State

- Completed batches:
  - `batch-1-topic-open-and-authority-recheck`
  - `batch-2-backend-cli-and-ent-log-unification`
  - `batch-3-security-event-and-audit-correlation-alignment`
  - `batch-4-frontend-global-error-sinks-and-trace-context`
  - `batch-5-cross-boundary-validation-and-regression-audit`
  - `batch-6-closeout-archive-and-commit-evaluation`
- Pending batches:
  - none
- Current batch:
  - none
- Next batch:
  - none

## Budget

- Max rounds: `6`
- Max files changed: `30`
- Max commits: `2`
- Max runtime minutes: `180`
- Validation failure policy: `stop-on-failure`
- Checkpoint budget per round: `1`

## Batch 1 Acceptance Snapshot

- active topic and branch rename are complete
- recovery index now points this worktree at `logging-unification-rollout`
- authority recheck confirmed:
  - in-scope backend bypasses are the three CLI entrypoints plus Ent generated debug defaults
  - `httpx` already owns security-event emission and is the correct place to align request/trace/actor field names
  - frontend still lacks any shell-owned global error sinks and default logger context injection
- residual risk noted for later closeout:
  - some plugin-owned domain audit paths still manually inject request IDs outside this topic's allowed scope, so
    this topic can only close the in-scope authority surfaces and report remaining drift precisely

## Validation

- Passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./plugins/user/ent ./plugins/rbac/ent`
  - `cd web && bun run check`

## Final Status

- Result: `archive-ready`
- Commit eligibility:
  - owned scope is clear
  - validation is complete for directly changed code
  - no mixed ownership was detected in the worktree at task start
- Archive notes:
  - `traceId` intentionally remains equal to `requestId` in MVP
  - metrics stayed out of scope
  - plugin-owned domain-event request-id helpers outside the allowed scope remain a separate future topic, not a
    blocker inside this bounded rollout
