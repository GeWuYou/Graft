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

## 2026-07-22 Runner Digest Authority

- Added the checksummed `runners.compose` immutable identity to the canonical release manifest in `20999c3f`; publishing validates an externally supplied digest and does not claim a local runner build context as release authority.
- Update manifest verification now requires official GHCR server, web and sibling runner image/reference/digest triples, and Compose preflight rejects a missing, mutable or non-official runner reference.
- Removed the release-stage reduced manifest overwrite; validated update tests, backend completion entrypoint, publish YAML parsing and diff hygiene.

## 2026-07-22 Compose Fixture Execution

- Added the hermetic local-version Compose runner fixture in `5a5ecddd`; it verifies the restricted runner action order, receipt persistence, immutable digest rejection and same-tag server/web target reconstruction.
- A migration-stage failure remains `NEEDS_ATTENTION` and exercises no database restore action. Compose configuration validation ran without starting containers, preserving the worktree runtime boundary.

## 2026-07-22 Archive Readiness Gap

- Archive review confirmed the prerequisite boundaries but rejected completion: Update operation state is not durable and no constrained Docker socket launcher exists to run the manifest-pinned runner or settle its receipt after server recreation.
- The next bounded rollout slice must add Update-owned history/API and the one-shot launcher without changing Task or Backup ownership; it must preserve the post-migration `NEEDS_ATTENTION` boundary.

## 2026-07-23 Rollout Delivery And Archive Readiness

- `1b2dda36` and `fd6ccc7c` added Update-owned operation history, confirmation/history APIs, a constrained Docker
  socket launcher, and receipt settlement without taking ownership of Task or Backup facts.
- `e4565581` made the one-shot runner a release-delivered, digest-pinned image and persisted only verified discovery
  results with explicit stale/error state.
- `53f125f4` and `01abcd32` added real Compose smoke coverage and final release authority gates for stale catalogs,
  minimum source versions, full binary guidance, manifest assets, and lifecycle phases.
- Final validation passed: backend completion entrypoint, `bun run check`, focused Task/Backup/Update tests, SQL
  migration and ai-plan structure checks, release-grade BuildInfo validation, publish workflow YAML parsing, and the
  local Docker Compose runner smoke. The smoke left no runner project or registry container.
- The topic reached `archive-ready`; this trace is historical evidence and must not replace current root governance.

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
    "compose-fixture-execution",
    "compose-execution-and-recovery"
  ],
  "pending_batches": [],
  "current_batch": "archive-readiness",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
