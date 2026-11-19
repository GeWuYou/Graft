Continue the Runtime Diagnostics Recovery topic only within its bounded optimize scope.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `ai-plan/public/runtime-diagnostics-recovery/README.md`
- recovery entry: `ai-plan/public/runtime-diagnostics-recovery/README.md`
- local execution truth: `server/AGENTS.md`, `web/AGENTS.md`, and `ai-plan/AGENTS.md`
- decision authority: `ai-plan/design/decisions/ADR-026-docker-runtime-agent-execution-boundary.md`
- contract authority: `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- worker context: delegated worker `track1-diagnostics` on branch `feature/runtime-diagnostics-projection`

Topic objective:

- Make execution diagnostics observable end-to-end: bounded execution stage attribution, failure classification, a
  sanitized diagnostic summary, and claim-admission tri-state readiness, with additive-only contract changes.

Locked decisions:

1. All OpenAPI changes are additive; the oasdiff breaking gate must pass without new approvals.
2. Diagnostic summaries are length-capped and sanitized; paths, credentials, and raw provider stderr never reach
   receipts, logs, or task logs.
3. `ExecutionAdmission` semantics in the task module stay unchanged; claim-admission evidence is projected from agent
   telemetry through Runtime Target readiness.

Implementation guardrails:

- Fix the canonical authority first, then regenerate derived artifacts in the same slice.
- Reuse the existing `ExecutionFailureClass` enum; do not create a parallel classification.
- Keep telemetry dedup/rate limiting bounded so admission reporting cannot spam the ledger.
