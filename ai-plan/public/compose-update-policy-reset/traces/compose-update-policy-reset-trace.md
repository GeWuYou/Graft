# Compose Update Policy Reset Trace

## 2026-07-29 compose-contract-and-governance-reset

- Accepted ADR-007 without rewriting historical ADR-006: the shared `GRAFT_IMAGE_TAG` and `GRAFT_UPDATE_POLICY` now own the official Compose declaration; ADR-006 retains runner trust-boundary authority.
- Created the active-topic Work Contract and removed repository-plus-digest image variables from the official Compose example.

## 2026-07-29 runner-server-web-policy-batches

- Completed runner policy writes, digest verification, and receipt failure preservation against the new Compose authority.
- Completed server/OpenAPI policy initialization, release catalog and App Log failure evidence, plus Web policy selection, fixed-release selection, and truthful stage-progress rendering.
- The active recovery point is cross-boundary validation and archive-readiness review; no parent topic owns this recovery entry.

## Locked Decisions

- `stable`, `beta`, `fixed`, and `manual` are the only current policies; `nightly` is explicitly out of scope.
- The release manifest remains immutable identity authority; a shared explicit image tag is only executable by the runner after digest validation.
- No fallback, alias, or migration path exists for the removed legacy image configuration.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["compose-contract-and-governance-reset", "runner-policy-and-receipt-reliability", "server-contract-and-app-log-evidence", "web-policy-selection-and-progress-rendering"],
  "pending_batches": ["cross-boundary-validation-and-archive-readiness"],
  "current_batch": "cross-boundary-validation-and-archive-readiness",
  "next_batch": "",
  "closeout_status": "active"
}
```
