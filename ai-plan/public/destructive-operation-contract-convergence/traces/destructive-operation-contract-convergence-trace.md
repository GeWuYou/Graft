# Destructive Operation Contract Convergence Trace

## 2026-08-19 Contract Foundation

- Completed root startup preflight and classified the task as cross-boundary with no prior topic owner.
- Acquired idle numbered worktree `01` on `refactor/unified-destructive-operations`; the primary checkout remains untouched.
- Chose existing backend API governance plus canonical OpenAPI as authority instead of creating a generic deletion service or a second contract registry.
- Selected user soft deletion as the first runtime pilot after the contract foundation.

## 2026-08-19 Contract Foundation And Soft-Delete Pilot Closeout

- Added the canonical `x-graft-destructive` schema and validator for method, effect, execution, retry, result, authorization, audit, confirmation, batch, and MCP exposure consistency.
- Added one shared destructive batch envelope with `operation_id`, summary, ordered per-item results, and explicit partial/atomic metadata.
- Migrated user deletion from `POST /api/users/{id}/delete` to `DELETE /api/users/{id}` without an alias and regenerated server/web OpenAPI consumers.
- Added a narrow tombstone lookup seam so ordinary `GetByID` and list queries stay deleted-filtered while authorized repeated DELETE succeeds without duplicate session revocation or audit emission.
- Completion evidence:
  - `go run ./cmd/graft validate backend`: passed
  - `bun run check`: passed; 306 test files and 2139 tests passed, release build succeeded
  - `python3 -m unittest discover -s scripts -p 'test_*.py'`: 169 tests passed
  - `python3 scripts/validate_ai_governance.py`: passed
  - `python3 scripts/validate_ai_plan_structure.py`: passed
- `just` was unavailable in the numbered worktree environment; deterministic generation was run through the exact underlying OpenAPI bundle/runtime-paths, Go generation, TypeScript generation, and projection generation commands.

## Locked Decisions

- DELETE tombstones are idempotent success for the same authorized scope, but default reads remain 404/hidden.
- Irreversible deletion is a POST command with a stored receipt; external destruction is a 202 Task workflow.
- Ordinary resource batches may partially commit; security-sensitive batches are atomic.
- Metadata must describe implemented behavior, never a future target presented as current truth.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "contract-foundation",
    "soft-delete-pilot"
  ],
  "pending_batches": [
    "relationship-and-rbac",
    "hard-delete-commands",
    "external-destruction-tasks",
    "convergence-closeout"
  ],
  "current_batch": "relationship-and-rbac",
  "next_batch": "hard-delete-commands",
  "closeout_status": "batch-validated"
}
```
