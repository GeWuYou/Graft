# Phase D Log Explorer Authority Definition Tracking

- Topic: `phase-d-log-explorer-authority-definition`
- Status: `archive-ready`
- Task class: `cross-boundary`
- Goal:
  - define the authority model for future `Log Explorer`
  - keep the topic governance-only
- Allowed scope:
  - `ai-plan/design/**`
  - `ai-plan/public/phase-d-log-explorer-authority-definition/**`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
- Forbidden scope:
  - runtime implementation
  - OpenAPI runtime contract changes
  - metrics/tracing/otel expansion

## Completed

1. Startup preflight completed under root `AGENTS.md`.
2. Confirmed task class is `cross-boundary`.
3. Read current observability, logging, contract, server, and web governance truth.
4. Confirmed current authority facts:
   - `server/internal/logger/**` owns `AppLogger`
   - `server/internal/httpx/**` owns request correlation and `Access Log`
   - `server/internal/audit/**` + `server/plugins/audit/**` own audit persistence
   - `server/plugins/monitor/**` owns bounded short-retention monitor evidence
5. Confirmed current retention gaps:
   - no formal audit/access/app retention authority exists

## Final Decision

- Final verdict: `Archive Ready`
- Runtime readiness: `Partially Ready`

## Remaining Runtime Gaps

- define retention authority for audit/access/app logs
- define runtime storage/query authority for future log explorer
- define explicit shared wire contract only after the two items above
