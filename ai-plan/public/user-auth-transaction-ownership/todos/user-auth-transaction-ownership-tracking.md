# User/Auth Transaction Ownership Tracking

## Topic

User/Auth Transaction Ownership

## Scope

Refactor user and auth persistence workflows so each business transaction has one explicit owner, then add the minimum adapter needed for atomic user profile, credential, and session lifecycle writes.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: user owns profile lifecycle, auth owns credential and session state, and moduleapi owns the narrow stable cross-module capability
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current state: Batch 5 complete and validated. User lifecycle workflows own one raw `*sql.Tx`, while the auth adapter binds only auth-owned writes to that same transaction.
- Risk: no user/auth composite transaction blocker remains. Default-admin boot and development reset remain deliberately retryable initialization workflows outside this lifecycle boundary.
- Next step: run the topic archive-readiness check and move this completed topic to the historical router.

## Task Checklist

- [x] Batch 1: move auth multi-write transaction lifecycle out of repositories and prove local rollback.
- [x] Batch 2: establish user ownership rules and record the remaining auth-owned multi-session revocation blocker.
- [x] Batch 3: add the minimum transaction-scoped cross-module capability contract.
- [x] Batch 4: bind auth to the user-owned composite transaction and remove compensation.
- [x] Batch 5: add cross-module atomicity and failure-injection tests; run final validation.

## Acceptance Conditions

- Every user/auth multi-write use case has one documented transaction owner.
- Composite create, disable, and delete operations either commit all user/auth facts or persist none.
- No adapter begins, commits, rolls back, or uses a different transaction from its owner.
- No compensation path remains as a claimed atomicity mechanism.
- Focused transaction tests cover success and each write-side failure boundary.

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

## Batch 1 Evidence

- Auth service now defines password/session reset, password change, development reset, and refresh rotation transaction scope through `store.TransactionRunner`.
- `storeent` creates one Ent transaction, defers rollback until successful commit, and passes transaction-scoped credential/session stores to the callback.
- Repository methods no longer create, commit, or roll back transactions.
- Validation passed: `go test ./modules/auth/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.

## Batch 2 Evidence

- User profile create, status transition, and deletion now run through a user-owned Ent transaction runner. The runner begins one transaction, passes only a transaction-scoped profile repository to its callback, rolls back callback failures, and commits only on callback success.
- Credential provisioning and session revocation intentionally remain outside the profile transaction pending Batch 4. Their failure errors now state that the profile write committed, return the committed profile where applicable, and never invoke a compensating delete or undo write.
- The user module cannot change the current auth `RevokeOtherSessionsByUserID` implementation because it is an auth-owned `moduleapi` capability. Although the auth store already has a collection update, the service capability still iterates sessions; Batch 3/4 must decide and implement its stable collection-oriented contract without user reaching into auth internals.
- Added an Ent-backed rollback proof for profile creation and a service-level assertion that credential-provision failure leaves the committed profile visible to the caller.
- Validation passed: `go test ./modules/user/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.

## Batch 3 Evidence

- Added `moduleapi.AuthTransactionAdapterFactory` and `AuthTransactionAdapter`. The factory receives a caller-owned `*sql.Tx`; its adapter surface contains only auth-owned credential provisioning and session revocation writes. Neither interface exposes Begin, Commit, Rollback, context propagation, or a generic UnitOfWork.
- The contract deliberately is not registered or implemented in this batch. The current user Ent transaction runner cannot prove it is binding the same raw SQL transaction, so an implementation here would violate lifecycle ownership.
- `RevokeOtherSessionsByUserID` and the current-user equivalent now use one auth collection update. The auth store returns the affected count so the stable `Revoked` result remains accurate without a list-then-loop write pattern.
- Validation passed: `go test ./modules/auth/... ./internal/moduleapi/...`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, and `git diff --check`.

## Batch 4 Evidence

- User's composite transaction runner creates the only raw `*sql.Tx` for create, disable, and delete workflows. It binds the user Ent client to that transaction, calls the auth adapter factory with the same handle, and alone commits or rolls back.
- The auth adapter constructs an auth-only Ent client bound to the caller-owned transaction. It receives prepared hashes from auth service policy and does not query the independent user identity provider, which cannot see an uncommitted profile.
- Ent's implicit transaction requests are converted to no-op commits only inside each module's transaction-bound private client. Statements still execute on the raw caller-owned transaction, so the user composite runner remains the sole completion owner.

## Batch 5 Evidence

- A real shared SQLite database now proves cross-module commit for profile plus credential creation.
- SQLite write triggers inject failures at the auth credential insert and refresh-session update boundaries. Create leaves neither profile nor credential; disable and delete leave the profile enabled and visible, with the session still active.
- Validation passed: `go test ./modules/user/... ./modules/auth/... -count=1`, `go run ./cmd/graft validate backend --stage lint`, `go build ./cmd/graft`, `git diff --check`, and `python3 scripts/validate_ai_plan_structure.py`.
