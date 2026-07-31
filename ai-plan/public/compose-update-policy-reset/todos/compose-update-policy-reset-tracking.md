# Compose Update Policy Reset Tracking

## Work Contract

```yaml
version: 1
kind: bug
scope: long-running
authority_summary: official Compose deployment configuration and verified release manifests own update image and tag-strategy semantics
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-batch
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

- The old repository-plus-digest Compose contract has been replaced without compatibility support.
- Compose, runner, server/OpenAPI, and Web now consume the resulting `GRAFT_IMAGE_TAG`-only contract. Unified SSE update progress and cross-boundary validation are complete; the topic is archive-ready.

## Task Checklist

- [x] Work Intake contract, active-topic bootstrap, ADR-007, and Compose configuration reset
- [x] runner tag-strategy handling, digest verification, and receipt failure preservation
- [x] server/OpenAPI tag initialization, history, and App Log failure evidence
- [x] Web tag-strategy rendering, fixed-release selection, and stage-progress UI
- [x] Deployment Runtime context ownership, canonical deployment keys, and shared Compose-root snapshot
- [x] Update operation snapshot rename from `update_mode` to `deployment_strategy`, including forward migration and affected-release recovery guidance
- [x] unified realtime SSE progress, cross-boundary validation and archive-readiness review

## Acceptance Conditions

- Official Compose uses `GRAFT_IMAGE_TAG` as its one shared image tag and update strategy, with no `GRAFT_UPDATE_POLICY` or alternate-key compatibility.
- `latest` tracks stable, `beta` tracks Beta, and fixed SemVer tags only select strictly newer verified releases in the same channel; server-side validation prevents downgrade and cross-channel upgrades.
- A tracking update resolves a verified manifest and digest at runtime without replacing `latest` or `beta` in `.env`.
- Automated upgrades only select verified releases and validate pulled digests before migration/recreation.
- A terminal update failure leaves both an actionable Update diagnostic and a request-correlated App Log record.
- Running UI never presents indeterminate progress as `100%`.
- `update_operations` ends at the canonical `deployment_strategy` column after the immutable `300001` then forward `300002` migration sequence; no `update_mode` API or storage alias remains.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "compose-contract-and-governance-reset",
    "runner-policy-and-receipt-reliability",
    "server-contract-and-app-log-evidence",
    "web-policy-selection-and-progress-rendering",
    "deployment-runtime-context-convergence",
    "unified-realtime-sse-update-progress",
    "cross-boundary-validation-and-archive-readiness"
  ],
  "pending_batches": [],
  "current_batch": "cross-boundary-validation-and-archive-readiness",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
