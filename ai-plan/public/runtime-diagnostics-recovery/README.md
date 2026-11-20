# Runtime Diagnostics Recovery

## Current Status Summary

- Topic objective: make execution diagnostics observable end-to-end with additive contract changes only.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: bounded optimize topic for the runtime diagnostics projection PR; created by the delegated worker
  `track1-diagnostics` in the `feature/runtime-diagnostics-projection` task worktree.
- Canonical authority:
  - `server/internal/moduleapi/**` shared Task contract
  - `server/modules/task/**`, `server/modules/runtime-target/**`
  - `server/internal/dockerexecution/**`, `server/agents/docker-runtime-agent/**`
  - `openapi/**`, `web/src/modules/task/**`, runtime-target web display and module locales

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: Task Runtime owns receipt settlement and projection, Runtime Target owns agent readiness and
  telemetry, Docker Runtime Agent owns claim admission observation, canonical OpenAPI owns schema closure, and the web
  modules own presentation with zh+en locales.

## Owned Scope

- `server/internal/moduleapi/`, `server/modules/task/`, `server/internal/dockerexecution/`
- `server/internal/httpx/agent_execution_routes.go` + `agent_ledger_routes.go` and their tests
- `server/modules/runtime-target/`, `server/agents/docker-runtime-agent/`
- `openapi/` additive schema closure with generated artifacts
- `web/src/modules/task/`, runtime-target-related web display and `web/src/locales/lang/` keys
- this topic directory and the `ai-plan/public/README.md` active entry

Out of scope:

- `.github/workflows/`, `Justfile`, `scripts/` automation
- any web layout redesign, provider stderr in receipts/logs/task logs, unrelated refactors

## Current Recovery Point

- Contract/backend slice, agent admission slice, provider redaction slice, and web presentation slice are planned in
  `todos/runtime-diagnostics-recovery-tracking.md`; consult the tracking checklist before resuming.

## Work Intake

- This topic was created through Work Intake before implementation commits.
- The full Work Contract is in `todos/runtime-diagnostics-recovery-tracking.md`.

## Validation Targets

```bash
just openapi-check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Loop Entry

- Preferred entry: `ai-plan/public/runtime-diagnostics-recovery/startup-prompt.md`
