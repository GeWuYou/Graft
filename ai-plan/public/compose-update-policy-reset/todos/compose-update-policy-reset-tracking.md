# Compose Update Policy Reset Tracking

## Work Contract

```yaml
version: 1
kind: bug
scope: long-running
authority_summary: official Compose deployment configuration and verified release manifests own update image and policy semantics
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
- Compose, runner, server/OpenAPI, and Web now consume the resulting `GRAFT_IMAGE_TAG` and `GRAFT_UPDATE_POLICY` contract; the remaining batch is cross-boundary validation and archive readiness.

## Task Checklist

- [x] Work Intake contract, active-topic bootstrap, ADR-007, and Compose configuration reset
- [x] runner image/policy write, digest verification, and receipt failure preservation
- [x] server/OpenAPI policy initialization, history, and App Log failure evidence
- [x] Web policy selection, fixed-release selection, and stage-progress UI
- [ ] cross-boundary validation and archive-readiness review

## Acceptance Conditions

- Official Compose uses one shared `GRAFT_IMAGE_TAG` and one explicit policy value with no alternate-key compatibility.
- Automated upgrades only select verified releases and validate pulled digests before migration/recreation.
- A terminal update failure leaves both an actionable Update diagnostic and a request-correlated App Log record.
- Running UI never presents indeterminate progress as `100%`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "compose-contract-and-governance-reset",
    "runner-policy-and-receipt-reliability",
    "server-contract-and-app-log-evidence",
    "web-policy-selection-and-progress-rendering"
  ],
  "pending_batches": [
    "cross-boundary-validation-and-archive-readiness"
  ],
  "current_batch": "cross-boundary-validation-and-archive-readiness",
  "next_batch": "",
  "closeout_status": "active"
}
```
