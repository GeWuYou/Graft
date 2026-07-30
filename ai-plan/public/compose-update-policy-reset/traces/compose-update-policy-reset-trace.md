# Compose Image Tag Strategy Reset Trace

## Superseded Historical Decision: 2026-07-29 compose-contract-and-governance-reset

- The earlier two-key `GRAFT_IMAGE_TAG` plus `GRAFT_UPDATE_POLICY` declaration was accepted without rewriting historical ADR-006; ADR-006 retains runner trust-boundary authority. This declaration is superseded and is not current guidance.
- Created the active-topic Work Contract and removed repository-plus-digest image variables from the official Compose example.

## 2026-07-29 image-tag-strategy reset

- ADR-007 now makes `GRAFT_IMAGE_TAG` the only Compose image and update-strategy declaration: `latest` tracks stable, `beta` tracks Beta, and SemVer tags are fixed stable or Beta releases.
- A runner resolves verified manifest identity and digest only for the current operation. It preserves tracking tags in `.env`; a fixed upgrade writes only the confirmed higher same-channel tag.
- Server reads the injected tag, not a host `.env`, and enforces official release, digest, channel, and version-order rules without relying on the UI.

## 2026-07-29 runner-server-web-tag-strategy batches

- Completed runner tag-strategy handling, digest verification, and receipt failure preservation against the Compose authority.
- Completed server/OpenAPI tag initialization, release catalog and App Log failure evidence, plus Web strategy rendering, fixed-release selection, and truthful stage-progress rendering.
- At that point, the next recovery step was cross-boundary validation and archive-readiness review; no parent topic owned this recovery entry.

## 2026-07-30 deployment-runtime-context convergence

- ADR-008 moves Compose-root discovery and deployment-fact interpretation out of Update into the shared Deployment Runtime boundary. `DeploymentContext` is immutable; `Current` is read-only and `Freeze` creates the operation-scoped snapshot used by controlled execution.
- Container remains a provider of raw Docker inspect facts only. Update, Backup, Restore, diagnostics, and future Compose management must consume the Deployment Runtime capability and must not inspect labels, read deployment environment variables, or derive host paths independently.
- `GRAFT_DEPLOYMENT_RUNTIME` and optional `GRAFT_DEPLOYMENT_COMPOSE_ROOT` replace Update-prefixed deployment configuration without aliases, dual reads, or fallback keys. `GRAFT_IMAGE_TAG` remains the official Compose image/update strategy.
- The next implementation batch owns server wiring, migration of Update consumers, and regression coverage before the topic returns to cross-boundary validation and archive readiness.

## Locked Decisions

- `GRAFT_IMAGE_TAG` is the only current image and update-strategy declaration; `GRAFT_UPDATE_POLICY` is removed without compatibility. `nightly` is explicitly out of scope.
- The release manifest remains immutable identity authority; runner-scoped resolution of an explicit target and digest does not rewrite `latest` or `beta` in `.env`.
- No fallback, alias, dual-read, or migration path exists for the removed legacy image configuration.
- Deployment context is the single source of truth for deployment interpretation; every controlled operation consumes an immutable manager-produced context or frozen snapshot.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["compose-contract-and-governance-reset", "runner-policy-and-receipt-reliability", "server-contract-and-app-log-evidence", "web-policy-selection-and-progress-rendering"],
  "pending_batches": ["deployment-runtime-context-convergence", "cross-boundary-validation-and-archive-readiness"],
  "current_batch": "deployment-runtime-context-convergence",
  "next_batch": "cross-boundary-validation-and-archive-readiness",
  "closeout_status": "active"
}
```
