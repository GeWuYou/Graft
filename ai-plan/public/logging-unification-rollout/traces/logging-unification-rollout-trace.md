# Logging Unification Rollout Trace

## 2026-05-29 Batch 1 started

- Re-ran startup preflight from root `AGENTS.md`.
- Read:
  - `.ai/environment/tools.ai.yaml`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/public/README.md`
  - archived `ai-plan/public/logging-governance/**`
  - archive-ready `ai-plan/public/request-correlation-access-logging/**`
  - `temp/logging-governance-assessment.md`
  - `ai-plan/design/契约治理与魔法值治理规范.md`
- Renamed the worktree branch from `feat/request-correlation-access-logging` to
  `feat/logging-unification-rollout`.
- Reconfirmed current authority and bounded gaps:
  - `server/internal/logger/**` remains the backend `AppLogger` authority
  - `server/internal/httpx/**` remains the request correlation, access log, and security-event bridge authority
  - `server/internal/audit/**` plus `server/plugins/audit/**` remain the audit persistence authority
  - `web/src/utils/logger/**`, `web/src/utils/request.ts`, and `web/src/app/**` remain the frontend logging-context
    and global error-capture authority
- Batch 1 grep audit snapshot:
  - confirmed stdlib `log.Fatalf` remains in the three CLI entrypoints
  - confirmed Ent generated clients still default debug logging to `log.Println`
  - confirmed no direct `console.log/error/warn` drift exists under `web/src`
  - sensitive-word grep hits were dominated by tests, contract names, generated fields, and auth plumbing; no new
    in-scope plaintext secret logging issue was identified during startup
- Authority recheck also confirmed one bounded constraint:
  - request-id manual context threading still exists in some plugin-owned domain audit paths outside this topic's
    allowed scope, so this loop will focus on closing the `httpx` security-event bridge plus shared field semantics
    inside owned scope and will report residual plugin-path risk honestly if it remains

## 2026-05-29 Batch 2 completed backend app-logger closure

- Replaced the remaining CLI stdlib fatal logging paths in:
  - `server/cmd/graft/main.go`
  - `server/cmd/graft-jwt-secret/main.go`
  - `server/cmd/graft-signing-key/main.go`
- Added `server/internal/logger.NewBootstrap()` for early-process fatal paths before full runtime config is available.
- Installed the runtime logger as the `zap` global baseline from `server/internal/logger.New(cfg)` so owned low-level
  debug paths can reuse the same backend.
- Repointed owned Ent generated debug defaults to `zap.L().Debug(...)` instead of stdlib `log.Println` in:
  - `server/plugins/user/ent/client.go`
  - `server/plugins/rbac/ent/client.go`
- Validation accepted:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./plugins/user/ent ./plugins/rbac/ent`

## 2026-05-29 Batch 3 completed security-event and audit correlation alignment

- Canonicalized in-scope security-event metadata in `server/internal/httpx/authz.go` using:
  - `requestId`
  - `traceId`
  - `actorId`
  - `actorType`
  - `route`
  - `method`
  - `path`
  - `status`
  - `plugin`
  - `component`
  - `eventType`
  - `riskLevel`
  - `targetType`
  - `targetId`
- Updated `server/internal/audit/service.go` to persist the canonical field dictionary while still emitting the current
  legacy aliases such as `audit_source`, `request_method`, `request_path`, `status_code`, and `trace_id`.
- Kept `traceId == requestId` as the accepted MVP semantics; no separate tracing platform was introduced.

## 2026-05-29 Batch 4 completed frontend logger closure

- Added shell-owned global logger context merging in `web/src/utils/logger/index.ts`.
- Promoted backend response `traceId` into default frontend logger context through `web/src/utils/request.ts`.
- Added shell-owned global error sinks in `web/src/app/bootstrap/index.ts` for:
  - `app.config.errorHandler`
  - `window.onerror`
  - `window.unhandledrejection`
- Synced route context into the frontend logger baseline via router startup state plus `afterEach`.

## 2026-05-29 Batch 5 completed cross-boundary validation and grep audit

- Backend validation passed:
  - `cd server && go test ./internal/httpx ./internal/audit ./internal/logger ./cmd/graft ./cmd/graft-jwt-secret ./cmd/graft-signing-key ./plugins/user/ent ./plugins/rbac/ent`
- Frontend validation passed:
  - `cd web && bun run check`
- Required grep audit summary:
  - `grep -R "log\\." server/cmd server/internal server/plugins || true`
    - remaining hits were config keys, comments, and generated Ent migration comments rather than active stdlib log
      bypasses
  - `grep -R "fmt.Println\\|println" server || true`
    - remaining in-scope hits are the two key-generator stdout lines whose purpose is intentional CLI output, not app
      logging
  - `grep -R "console.log\\|console.error\\|console.warn" web/src || true`
    - no direct `web/src` runtime bypasses remained
  - `grep -R "Authorization\\|password\\|refresh_token\\|cookie" server web || true`
    - hits were dominated by tests, generated schema fields, auth contracts, and dependency trees; no new in-scope
      plaintext-secret logging issue was introduced by this topic

## 2026-05-29 Batch 6 completed archive-readiness evaluation

- Accepted this topic as `archive-ready`.
- Explicit non-goal retained:
  - plugin-owned domain-event request-id adapters outside the allowed scope still exist and should be handled only in a
    separate bounded follow-up topic if they become a priority
