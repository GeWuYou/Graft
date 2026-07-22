# Platform Self Update Tracking

## Topic

Platform Self Update

## Scope

Build controlled update discovery first, then manual Compose self-update with independent backup, migration, health evidence, and recovery records.

## Repository Truth

- `AGENTS.md`
- `ai-plan/design/release/`
- `.github/workflows/publish.yml`
- `compose.yml`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Release policies and publication workflows own release identity; Compose, server module contracts, and web navigation own deployment and product integration.
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
    - ai-plan/public/platform-self-update
    - ai-plan/design/release/platform-self-update.md
    - ai-plan/roadmap/platform-self-update.md
    - ai-plan/design/decisions/ADR-006-platform-self-update-compose-runner.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Work Intake completed and the topic was bootstrapped.
- Batch 1 established release-manifest, release authority, roadmap, and Compose runner ADR.
- Batch 2 added read-only release discovery, the installation profile, and the protected status/check API.
- Batch 3 added the front-end Update Center and the logo version affordance.
- Batch 4 added the independent `platform-backup` capability with retained artifact metadata and recovery evidence.
- Task Runtime now settles bound, no-secret external receipts exactly once after a runner handoff.
- Backup now owns a runner handoff that freezes operation/task-bound artifact paths and re-verifies the resolved files, SHA-256, and byte counts before an idempotent Backup fact is created.
- Release manifest verification now includes a checksummed, official, immutable Compose runner identity.
- The hermetic Compose fixture proves digest rejection, same-tag reconstruction and fixed runner sequencing; the next batch is `compose-execution-and-recovery`.

## Task Checklist

- [x] `release-authority-and-manifest`
- [x] `read-only-update-discovery`
- [x] `update-center-ui`
- [x] `backup-capability`
- [x] `compose-runner-preflight-contract`
- [x] `task-receipt-settlement`
- [x] `backup-runner-handoff`
- [x] `runner-digest-authority`
- [x] `compose-fixture-execution`
- [ ] `compose-execution-and-recovery`
- [ ] `archive-readiness`

## Acceptance Conditions

- Update discovery identifies a compatible release by SemVer channel and exposes an installation capability matrix.
- Only supported official Compose installations can execute the MVP update; binary installations receive verified manual guidance.
- Update execution uses an independent backup capability, explicit migration, health evidence, durable history, and audit records.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "release-authority-and-manifest",
    "read-only-update-discovery",
    "update-center-ui",
    "backup-capability",
    "compose-runner-preflight-contract",
    "task-receipt-settlement",
    "backup-runner-handoff",
    "runner-digest-authority",
    "compose-fixture-execution"
  ],
  "pending_batches": [
    "compose-execution-and-recovery",
    "archive-readiness"
  ],
  "current_batch": null,
  "next_batch": "compose-execution-and-recovery",
  "closeout_status": "in_progress"
}
```
