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

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["batch-0-contract-projection-intake"],
  "pending_batches": [
    "generator-foundation",
    "pilot-migration",
    "ci-integration",
    "broader-migration-and-final-archive-readiness"
  ],
  "current_batch": "batch-0-contract-projection-intake",
  "next_batch": "generator-foundation",
  "closeout_status": "committed"
}
```
