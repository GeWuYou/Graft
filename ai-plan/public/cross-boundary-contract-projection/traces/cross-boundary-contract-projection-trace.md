# Cross-Boundary Contract Projection Trace

## 2026-07-19 Batch 0: intake, authority and roadmap

- Work Intake classified the effort as long-running `refactor`; it needs an active topic, repository-wide design and roadmap, but not an ADR.
- Locked the authority split: OpenAPI owns HTTP wire schemas, paths and public wire enums; existing Go server contracts own non-HTTP stable values; web consumes generated artifacts only.
- Locked projection metadata as an index that references existing typed Go constants. It includes explicit owner/lifecycle/visibility and does not repeat values.
- Locked API compatibility: error code and message key stay open strings and web must retain fallback for unknown future values.
- Locked runtime boundary: generated permission/menu/capability values cannot replace server bootstrap or runtime availability decisions.
- Defined Phase 1 generation/drift baseline, Phase 2 pilot migration and Phase 3 broad convergence.
- Committed this intake batch after `git diff --check` and the bounded ai-plan structure guard passed.

## Locked Decisions

- Do not introduce protobuf, a shared runtime package, a new hand-written IDL, or a second contract authority.
- Preserve existing OpenAPI generation; projection follows it and reuses its artifacts.
- Run generation in authority order and block uncommitted derived output, duplicate semantic ownership, invalid visibility and lifecycle drift.

## 2026-07-19 Batch 1: generator foundation

- Added `server/internal/contract/projection` as a metadata index and renderer, not a new contract authority. `Registry` values reference existing `errorcode.Code`, `message.Key`, `httpheader.Name`, and `auth.Scheme` constants.
- Added an explicit Go generator with write and `--check` modes. Check mode renders in memory and compares the tracked `web/src/contracts/generated/platform.ts` artifact without writing it.
- Generated platform error code, message key, HTTP header, and auth-scheme literal objects plus union types. An internal-only header descriptor proves visibility filtering and is absent from web output.
- Added focused validation for deterministic generation order, duplicate/lifecycle metadata rejection, visibility filtering, and AST verification that registry `Value` fields are existing typed constant selectors rather than copied literals.
- Updated `just generate` to run OpenAPI bundle before Go bindings, then web OpenAPI types and cross-boundary projection. `just openapi-check` now includes projection freshness; PR CI wiring remains the later `ci-integration` batch.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["batch-0-contract-projection-intake", "generator-foundation"],
  "pending_batches": [
    "pilot-migration",
    "ci-integration",
    "broader-migration-and-final-archive-readiness"
  ],
  "current_batch": "generator-foundation",
  "next_batch": "pilot-migration",
  "closeout_status": "committed"
}
```
