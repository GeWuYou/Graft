# User/Auth Transaction Ownership Trace

## 2026-07-25 Intake And Bootstrap

- Classified the work as a long-running server refactor because it changes user/auth lifecycle ownership and needs serial transaction-safe batches.
- Confirmed that auth owns two hidden Ent transactions, user owns none, and user creation currently uses compensation after independent commits.
- Locked the composite transaction owner as the user lifecycle workflow; auth will provide only a tx-bound participant.

## Locked Decisions

- Do not introduce a global UnitOfWork, TransactionManager, or context-hidden transaction propagation.
- Do not use post-commit compensation to simulate atomicity.
- Default-admin boot and development reset remain explicitly retryable initialization workflows unless separately redesigned.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": [
    "auth-native-transaction-ownership",
    "user-ownership-and-session-revocation",
    "user-auth-transaction-contract",
    "user-auth-transaction-adapter",
    "transaction-consistency-proof"
  ],
  "current_batch": "auth-native-transaction-ownership",
  "next_batch": "user-ownership-and-session-revocation",
  "closeout_status": "not-started"
}
```
