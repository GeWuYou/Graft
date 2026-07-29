# Compose Update Policy Reset Trace

## 2026-07-29 compose-contract-and-governance-reset

- Accepted ADR-007 without rewriting historical ADR-006: full image references and `GRAFT_UPDATE_POLICY` now own the official Compose declaration; ADR-006 retains runner trust-boundary authority.
- Created the active-topic Work Contract and removed repository-plus-digest image variables from the official Compose example.

## Locked Decisions

- `stable`, `beta`, `fixed`, and `manual` are the only current policies; `nightly` is explicitly out of scope.
- The release manifest remains immutable identity authority; a complete tagged image reference is only executable after digest validation.
- No fallback, alias, or migration path exists for the removed legacy image configuration.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["compose-contract-and-governance-reset"],
  "pending_batches": ["runner-policy-and-receipt-reliability", "server-contract-and-app-log-evidence", "web-policy-selection-and-progress-rendering", "cross-boundary-validation-and-archive-readiness"],
  "current_batch": "runner-policy-and-receipt-reliability",
  "next_batch": "server-contract-and-app-log-evidence",
  "closeout_status": "active"
}
```
