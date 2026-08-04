# Network Connectivity Diagnostics Tracking

## Topic

Network Connectivity Diagnostics

## Scope

Cross-boundary platform-network diagnostics, batch health checks, extensible target/probe contracts, and the Network module web experience.

## Repository Truth

- `AGENTS.md`
- `server/modules/network/**`
- `openapi/**`
- `web/src/modules/network/**`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: platform-network module, canonical OpenAPI source, and network web module own connectivity diagnostics.
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
    - design
    - roadmap
    - adr
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Phase 3 implementation is complete in the current branch worktree: the shared ConnectivityStore serves batch health,
  target diagnostics, history, trace, export, and custom-target management through the canonical OpenAPI paths.
- Phase 4 implementation adds a nullable, sanitized HTTP-status projection to each persisted connectivity check. The
  projection comes only from the final valid HTTP `ProbeResult`, remains absent for non-HTTP/no-response targets, and
  is available to latest, history, batch, and single-run consumers without widening the batch list into a trace view.
- No unresolved authority escalation is known. Full web release build remains blocked by the pre-existing missing
  `monaco-editor` Dockerfile registration module; focused Network validation and all non-build frontend governance gates pass.
- Archive readiness is satisfied; no further product implementation batch is pending.
- The canonical Network entry is `/platform/network` for Connectivity; outbound policy remains reachable through the
  secondary `/platform/network/outbound` workflow, avoiding an undiscoverable batch-health page.

## Task Checklist

- [x] Phase 1: authority, registry, capabilities, and probe/report core
- [x] Phase 2: persistence, batch APIs, route explanation, and security boundary
- [x] Phase 3: ConnectivityStore, batch UI, target diagnostics UI, and integrated validation
- [x] Phase 4: HTTP-status summary projection for batch/latest/history checks

Phase 4 implementation evidence is recorded in the trace for controller settlement; controller-owned loop state is
unchanged in this worker round.

## Acceptance Conditions

- Diagnostics pages remain target-addressed across reruns and history selection.
- Batch and diagnostics use one target/report execution and persistence model.
- Registry capabilities allow protocol-specific probes without database shape changes.
- Exit IP is masked by default, permission gated when revealed, and never persisted unmasked.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "phase-1-authority-registry-probe-core",
    "phase-2-persistence-api-security",
    "phase-3-web-experience-integration",
    "phase-4-http-status-summary-projection"
  ],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
