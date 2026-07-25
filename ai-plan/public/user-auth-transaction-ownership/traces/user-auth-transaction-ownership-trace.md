# User/Auth Transaction Ownership Trace

## 2026-07-25 Intake And Bootstrap

- Classified the work as a long-running server refactor because it changes user/auth lifecycle ownership and needs serial transaction-safe batches.
- Confirmed that auth owns two hidden Ent transactions, user owns none, and user creation currently uses compensation after independent commits.
- Locked the composite transaction owner as the user lifecycle workflow; auth will provide only a tx-bound participant.

## Locked Decisions

- Do not introduce a global UnitOfWork, TransactionManager, or context-hidden transaction propagation.
- Do not use post-commit compensation to simulate atomicity.
- Default-admin boot and development reset remain explicitly retryable initialization workflows unless separately redesigned.

## 2026-07-25 Batch 1: Auth-Native Transaction Ownership

- Replaced repository-hidden Ent transactions for refresh rotation and password/session changes with an auth-local `store.TransactionRunner`.
- Auth service now owns the multi-write use-case scope; the runner creates the single Ent transaction, defers rollback, commits only after callback success, and supplies transaction-scoped stores.
- Development default-admin reset now uses the same local transaction boundary for password and refresh-session writes.
- Added Ent-backed rollback proof for a failed rotation and for failed password-plus-session work; neither leaves partial durable state.
- Validation passed: `go test ./modules/auth/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.
- Remaining blocker: user profile lifecycle and auth identity reads do not yet participate in one shared transaction, so no cross-module adapter is introduced in this batch.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "auth-native-transaction-ownership"
  ],
  "pending_batches": [
    "user-ownership-and-session-revocation",
    "user-auth-transaction-contract",
    "user-auth-transaction-adapter",
    "transaction-consistency-proof"
  ],
  "current_batch": "user-ownership-and-session-revocation",
  "next_batch": "user-ownership-and-session-revocation",
  "closeout_status": "batch-1-completed"
}
```
