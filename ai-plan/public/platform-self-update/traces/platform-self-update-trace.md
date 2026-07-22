# Platform Self Update Trace

## 2026-07-22 Work Intake And Bootstrap

- Classified as a long-running cross-boundary feature and selected the default `topic-completion-loop`.
- Created the minimum topic recovery material required by the Work Contract.
- Deferred all runtime upgrade execution until release authority, release manifest, and installation capability semantics are explicit.

## Locked Decisions

- The Compose executor is a future short-lived, receipt-writing runner rather than a persistent agent.
- A declared deployment mode is advisory and must be cross-checked against runtime detection.

## 2026-07-22 Release Authority And Manifest

- Accepted `ADR-006`: official Compose execution will use a one-shot receipt-writing runner, not a persistent agent or an in-container binary replacement.
- Added GitHub Release attachment `release-manifest.json`, generated only after server and web GHCR digests exist and validated before artifact upload.
- Defined stable/beta selection, binary manual-only boundary, independent backup capability, migration ownership, and installation capability matrix in release design truth.

## 2026-07-22 Read-Only Update Discovery

- Added the `platform-update` module with manifest validation, SemVer channel selection, installation-profile detection, and protected status/check APIs.
- Registered the daily check job and retained binary installation as a verification and manual-guidance path.

## 2026-07-22 Update Center UI

- Added `Platform -> Updates` as a module-owned dynamic route and exposed the running version under the side-nav logo.
- The page consumes the protected status/check endpoints, presents SemVer channel selection, release notes, installation evidence, and the capability matrix.
- The upgrade button remains disabled until the Compose executor, backup, and migration APIs exist; binary installations receive explicit manual guidance.
- Completed `update-center-ui`; the loop remains in progress with `backup-capability` next.

## 2026-07-22 Backup Capability

- Added the independent `platform-backup` module and narrow `BackupService` capability for Update to consume without accessing backup storage internals.
- Backup facts retain only controlled artifact references, SHA-256 integrity metadata, retention status, and recovery evidence; no backup content, `.env` secret, restore endpoint, or migration behavior is exposed.
- Validated the module, live migration SQL, and backend completion entrypoint before committing `40f61800`.

## 2026-07-22 Compose Protocol And Blocker

- Added the versioned, no-secret runner input/receipt and strict official Compose preflight in `2358b883`; migration-started failure is classified as `NEEDS_ATTENTION`, never an automatic database rollback.
- The executor cannot yet be implemented without violating authority boundaries: Task Runtime lacks durable external receipt settlement, Backup lacks a runner handoff contract, and release delivery has no pinned runner image identity.
- The prior loop stopped after its permitted retry. The user resumed the topic with an explicit dependency chain: Task receipt settlement, Backup runner handoff, runner digest authority, Compose fixture execution, then Update rollout.

## 2026-07-22 Task Receipt Settlement

- Added Task Runtime-owned, no-secret external receipt settlement in `ecb7ae41`, binding receipt protocol and operation identity to the frozen final Stage plan.
- Settlement persists idempotent Task-owned evidence, reconciles running or crash-recovered unknown stages, and appends immutable task events without allowing Update to write Task storage.
- Validated Task/Project/registry scopes, the live migration chain, and the backend completion entrypoint.

## 2026-07-22 Backup Runner Handoff

- Added the Backup-owned `backup_runner_handoffs` immutable execution-evidence table and public narrow capability in `70526fd1`; the handoff freezes operation/task identity, artifact root and refs before the one-shot runner starts.
- Target server settlement resolves paths beneath the frozen root, computes actual SHA-256 and byte counts, rejects forged metadata, and creates the Backup fact exactly once without exposing storage references through the completion receipt.
- Validated Backup package tests, migration hash and SQL/version gates, generated registry, diff hygiene, and the backend completion entrypoint.

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
    "backup-runner-handoff"
  ],
  "pending_batches": [
    "runner-digest-authority",
    "compose-fixture-execution",
    "compose-execution-and-recovery",
    "archive-readiness"
  ],
  "current_batch": null,
  "next_batch": "runner-digest-authority",
  "closeout_status": "in_progress"
}
```
