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

- Phase 1 accepted: `ca21f69a` established the canonical registry, typed capabilities, extensible sanitized report
  envelope, and Platform Update adapter without changing persistence, APIs, routes, RBAC, migrations, or web consumers.
- No unresolved authority escalation is known.
- Next step: Phase 2 replaces the legacy persistence/API boundary with target/check/report storage and secure batch execution.

## Task Checklist

- [x] Phase 1: authority, registry, capabilities, and probe/report core
- [x] Phase 2: persistence, batch APIs, route explanation, and security boundary
- [ ] Phase 3: ConnectivityStore, batch UI, target diagnostics UI, and integrated validation

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
    "phase-2-persistence-api-security"
  ],
  "pending_batches": [
    "phase-3-web-experience-integration"
  ],
  "current_batch": "phase-3-web-experience-integration",
  "next_batch": null,
  "closeout_status": "phase-2-settled"
}
```
