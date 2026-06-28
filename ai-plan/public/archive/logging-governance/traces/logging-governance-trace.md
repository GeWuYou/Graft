# Logging Governance Trace

## 2026-05-28 Batch 0 started

- Opened the bounded cross-boundary topic `logging-governance` on branch `feat/logging-governance`.
- Re-ran startup preflight from root `AGENTS.md` before any topic-specific recovery work.
- Read the required startup and architecture sources:
  - root `AGENTS.md`
  - `.ai/environment/tools.ai.yaml`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/public/README.md`
  - `ai-plan/design/architecture/项目设计.md`
  - `ai-plan/design/插件与依赖注入设计.md`
  - `ai-plan/roadmap/MVP实施计划.md`
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`
- Confirmed this round stays read-only for runtime code and is scoped to exploration, design, and topic recovery docs.

## 2026-05-28 Batch 1 completed

- Recorded the authoritative server inventory supplied for this topic round instead of widening into runtime edits.
- Confirmed the backend baseline is already `zap` through `server/internal/logger/logger.go` plus runtime/plugin
  injection.
- Logged remaining server-side governance gaps:
  - stdlib `log` remains in `server/cmd/graft/main.go`, `server/cmd/graft-jwt-secret/main.go`, and
    `server/cmd/graft-signing-key/main.go`
  - Ent debug logging remains in `server/plugins/user/ent/client.go` and `server/plugins/rbac/ent/client.go`
  - access logging still relies on Gin default logger/recovery in `server/internal/httpx/server.go`
  - request IDs originate in `server/internal/httpx/response.go` but are not mounted globally for root routes
  - `traceId` currently collapses to request ID and audit propagation is only partly automatic
- Confirmed audit persistence and security-denial audit publishing already exist and should remain explicit rather than
  being absorbed into a generic logger abstraction.

## 2026-05-28 Batch 2 completed

- Recorded the authoritative web inventory supplied for this topic round.
- Confirmed the frontend baseline already uses `web/src/utils/logger/**` as a structured wrapper around browser
  console via `consola`.
- Logged remaining frontend governance gaps:
  - no global sink such as `app.config.errorHandler`, `window.onerror`, or `unhandledrejection`
  - request `traceId` is preserved in `ApiRequestError` but not promoted into default frontend logger metadata
  - error handling remains mostly page-local `logger.error/warn + MessagePlugin`
- Confirmed current audit UI semantics stay backend-owned and should not be reinterpreted as client logging evidence.

## 2026-05-28 Batch 3 completed and topic archived

- Synthesized the server and web findings into a design-only architecture recommendation.
- Preserved `zap` as the recommended backend logging implementation.
- Recommended explicit separation between:
  - `AppLogger`
  - `AccessLogger`
  - `AuditRecorder`
  - security events as the bridge between authn/authz behavior and audit persistence
  - `MetricsEmitter`
- Produced the requested Chinese assessment artifact at `temp/logging-governance-assessment.md`.
- Kept the round docs-only; no runtime code, schema, or product-scope changes were introduced.
- Topic status is now archived; future implementation must reopen as a new bounded topic.
