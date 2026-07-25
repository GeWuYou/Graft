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

## 2026-07-25 Batch 2: User Ownership And Session Revocation Convergence

- Added a user-local `store.TransactionRunner`; the user service now owns begin, callback scope, commit, and rollback for profile creation, disabling, and deletion. The Ent-backed runner passes a transaction-scoped user repository and contains no hidden repository-owned transaction lifecycle.
- Removed the create-user soft-delete compensation path. Before the adapter exists, credential-provisioning failure returns an explicit `profile committed` error and the committed profile to the service caller. Disable/delete follow the same explicit failure model for session-revocation failure.
- Proved rollback with a real Ent transaction: a profile created before a callback error is not visible after the runner returns the error.
- Recorded a scope blocker instead of changing auth internals: user-facing `RevokeOtherSessionsByUserID` still loops through auth service calls despite auth store support for one collection update. Its stable contract and implementation belong to the later shared-capability/auth batch.
- Validation passed: `go test ./modules/user/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.

## 2026-07-25 Batch 3: Narrow Transaction Contract

- Added the stable `moduleapi.AuthTransactionAdapterFactory` contract, whose only input is a caller-owned `*sql.Tx`. Its returned auth participant can provision a password credential or revoke all sessions, but has no Begin, Commit, Rollback, or context propagation method.
- Did not register or implement the adapter. The current user runner owns an opaque Ent transaction and cannot prove that a separately created auth client shares it. Batch 4 must make the user lifecycle workflow own the raw transaction and construct both Ent module clients from it before binding the factory.
- Replaced auth's list-then-loop implementation for revoke-other sessions with its existing collection-write store path. The store now returns affected rows, preserving truthful `AuthSessionRevokeResult.Revoked` semantics.
- Validation passed: `go test ./modules/auth/... ./internal/moduleapi/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "auth-native-transaction-ownership",
    "user-ownership-and-session-revocation",
    "user-auth-transaction-contract",
    "user-auth-transaction-adapter",
    "transaction-consistency-proof"
  ],
  "pending_batches": [],
  "current_batch": "transaction-consistency-proof",
  "next_batch": null,
  "closeout_status": "archive-readiness-required"
}
```

## 2026-07-25 Batch 4: User-Owned Composite Transaction

- Replaced the opaque user Ent composite boundary with a raw `*sql.Tx` owned by the user lifecycle workflow. The runner binds a user Ent client to that transaction and passes the same handle to the narrow auth adapter factory.
- Registered auth's adapter factory during auth module setup. Its adapter uses auth's password-policy preparation callback and writes credentials or refresh-session revocations only through the caller-owned transaction.
- Added transaction-bound Ent drivers in each module's private `storeent` boundary. Ent's internal single-write transaction completion is a no-op there, so it cannot commit or roll back the outer lifecycle transaction; all SQL remains on the supplied raw transaction.

## 2026-07-25 Batch 5: Cross-Module Consistency Proof

- Added shared SQLite/Ent integration tests for the complete user/auth boundary. A successful create commits both user profile and auth credential.
- Injected real auth-table failures through SQLite triggers. Credential insert failure rolls back the newly created profile; session-revocation failure rolls back both disable and delete profile mutations and leaves the existing session active.
- Focused user/auth tests passed. Final lint, build, diff, and topic-structure validation are recorded in the batch closeout.
