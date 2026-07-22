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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "release-authority-and-manifest"
  ],
  "pending_batches": [
    "read-only-update-discovery",
    "update-center-ui",
    "backup-capability",
    "compose-execution-and-recovery",
    "archive-readiness"
  ],
  "current_batch": null,
  "next_batch": "read-only-update-discovery",
  "closeout_status": "completed"
}
```
