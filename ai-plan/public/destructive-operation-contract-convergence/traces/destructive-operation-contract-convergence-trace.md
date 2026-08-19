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

## 2026-08-19 Relationship And RBAC Closeout

- Replaced RBAC relationship-removal action paths with canonical `DELETE` operations for role permissions, one user's roles, and multi-user roles; legacy action paths were removed without aliases.
- Kept atomicity, authorization, audit, and persistence in the RBAC service/store boundary while the HTTP layer performs bounded validation and maps ordered shared results.
- Returned the shared `operation_id + summary + results` envelope for role/permission and user-role atomic batches; results preserve request order and the runtime rejects empty, duplicate, or over-100-item requests.
- Added truthful `x-graft-destructive` metadata only to the migrated relationship-removal operations and fixed external-schema reference normalization in the validator without weakening the shared per-item conditional contract.
- Regenerated OpenAPI bundle, Go bindings, TypeScript schemas, runtime paths, and embedded server docs; updated server/web contract tests and freshness checks.
- Completion evidence:
  - `go run ./cmd/graft validate backend`: passed
  - `bun run check`: passed; 306 test files and 2140 tests passed, release build succeeded
  - focused RBAC server and web API tests: passed
  - canonical OpenAPI validation and generated freshness checks: passed
- Semantic review outcome: OpenAPI remains wire authority, RBAC remains atomic domain authority, no compatibility bridge or generic destructive service was introduced, and redundant handler wrappers were deleted instead of exempted from lint.

## 2026-08-20 Hard Delete Commands

- Replaced audit-log `batch-delete` and App Log single/batch delete surfaces with `POST /api/audit/logs/deletions` and `POST /api/app-log/deletions`.
- Both commands require `Idempotency-Key`, advertise truthful `hard_delete` metadata, and use the shared destructive result contract.
- Audit deletion continues to persist protected deletion receipts in the audit-owned transaction; App Log deletion now claims and replays logger-owned receipts in `app_log_deletion_receipts` within the deletion transaction.
- Removed the legacy OpenAPI paths and regenerated the embedded bundle, Go bindings, TypeScript schema, and runtime path projection. Web callers now send a fresh idempotency key for single and batch App Log deletion.
- Completion evidence:
  - focused server logger/audit/OpenAPI tests: passed
  - `go run ./cmd/graft validate openapi`: passed
  - `bun run typecheck`: passed
  - `git diff --check`: passed

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "contract-foundation",
    "soft-delete-pilot",
    "relationship-and-rbac",
    "hard-delete-commands"
  ],
  "pending_batches": [
    "external-destruction-tasks",
    "convergence-closeout"
  ],
  "current_batch": "external-destruction-tasks",
  "next_batch": "convergence-closeout",
  "closeout_status": "batch-validated"
}
```
