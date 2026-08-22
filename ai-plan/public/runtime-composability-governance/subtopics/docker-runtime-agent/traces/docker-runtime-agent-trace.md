# Docker Runtime Agent Trace

## Loop Batch State

```json
{
  "current_batch": "docker-runtime-agent-batch-8-ui-and-cross-boundary-convergence",
  "next_batch": "docker-runtime-agent-batch-9-runtime-boundary-closeout",
  "closeout_status": "active"
}
```

## 2026-08-21 intake-and-batch-1-start

- Reproduced the architectural split: Runtime Target health uses the Moby SDK, while Application/Build lifecycle paths
  may invoke a missing `docker` executable. Host Docker availability does not imply CLI presence inside server.
- Reused the active `runtime-composability-governance` topic and created a bounded subtopic rather than a parallel topic.
- Accepted ADR-026: a single pull-based Runtime Agent owns always-on Docker socket access; Task Runtime owns external
  execution state; the short-lived Update Controller remains the survivor across server/Agent replacement.
- Selected platform/module/domain/event/table/test/consistency/delete semantic review layers before implementation.
- No server, web, OpenAPI, migration or generated code was changed in the Batch 1 design step.

## 2026-08-21 batch-1-accepted

- Converged project-layout, Compose, Build Provider SPI, Agent protocol, Task Runtime, self-update and active-topic
  recovery authority on ADR-026.
- Preserved ADR-006/009 host-path, durable state, fencing and survivor invariants while superseding CLI/launcher details.
- Validation passed: `git diff --check` and `python3 scripts/validate_ai_plan_structure.py`.

## 2026-08-22 batch-7-deployment-and-cli-deletion-accepted

- Recovery launch now reuses the existing Task-owned `controller_launch` Stage and single Runtime Agent pull
  boundary; server-local recovery Docker launch helpers are deleted.
- Recovery material and Agent state-volume-only isolation are covered by focused tests. The Compose runner uses Moby
  and official Compose SDKs, and the runner image no longer includes Docker CLI or Compose plugin.
- The one-time `cutover-v1` bootstrap authority and retained socket consumer inventory remain explicit. Recovery point
  advances to Batch 8 UI and cross-boundary convergence.

## 2026-08-22 batch-8-ui-and-cross-boundary-convergence-started

- Reconciled the subtopic Next Recovery Point from Batch 6 -> Batch 7 to Batch 8 before touching UI code.
- Locked the page brief: Application detail is `list-form-detail` with `explain` primary intent and
  `workflow`/`feedback` secondary checks; Runtime Target/Container own runtime metrics; Task Runtime, Runtime Target,
  Provider, Agent and Update Controller retain their existing fact boundaries.
- TDesign MCP is not callable in this environment; official Vue Next documentation is the approved fallback for the
  Button, Dropdown, Tabs, Card, Alert and Tag components used in this batch.

## 2026-08-22 batch-8-ui-and-cross-boundary-convergence-accepted

- Application detail now uses the `list-form-detail` explain surface, removes non-authoritative resource dashboards,
  preserves identity/runtime/lifecycle/service/configuration/task facts, and collapses narrow actions into overflow.
- Runtime Target projects Agent identity, version, generation, per-capability readiness and stable diagnostics; UI action
  gating and Task failure-code copy remain localized presentation over those canonical facts.
- OpenAPI and generated Web/Go artifacts are fresh. Full Web check (311 files, 2161 tests), focused Web tests,
  Runtime Target Go tests, backend validation, AI-plan structure and diff checks passed.
- Current batch remains Batch 8 for receipt continuity; next recovery point is
  `docker-runtime-agent-batch-9-runtime-boundary-closeout`. Batch 9 was not started.

## 2026-08-21 batch-2-accepted

- Added a provider-neutral `RuntimeAgentExecutionGateway` owned by Task Runtime for claim, renew, cancellation
  observation, bounded logs, receipt settlement and expiry.
- Persisted `external_execution` on Stage rows and made Agent claim atomically transition Stage/Task state while creating
  the fenced lease; local workers exclude external Stage rows at the database claim boundary.
- Extended the existing external receipt authority instead of creating a second receipt store. Lease and receipt
  uniqueness include attempt so an operator-approved retry can reuse the frozen operation identity safely.
- Preserved a still-valid lease across server restart; expired leases converge to `unknown`/`needs_attention`, while a
  fully matching late receipt remains reconcilable and stale fences are rejected.
- Classified migration `202608210001` as L4 because it replaces existing receipt constraints; preflight metadata records
  historical assumptions, upgrade order, recovery rationale and MIG-002 evidence.
- Retained the final-stage receipt writer only as the explicitly temporary Update Controller terminal-state handshake;
  normal controller launch no longer uses it and Batch 7 owns its retirement decision.
- Validation passed: focused Task tests, Task race tests, production/test lint, the complete `graft validate backend`
  entrypoint, SQL migration/version gates, AI-plan structure guard and `git diff --check`.

## 2026-08-21 batch-3-accepted

- Promoted the experimental build-only Agent package, binary, image, configuration, development deployment and
  conformance fixture directly to `docker-runtime-agent`; removed the optional root Compose profile and all old entries.
- Added Runtime Target's explicit identity-scoped capability binding. Enrollment writes it transactionally, migration
  copies the former single profile once, and execution admission reads only the new binding plus mTLS identity.
- Connected the Agent to `RuntimeAgentExecutionGateway` through bounded long polling, renewal, cancellation observation,
  fixed redacted logs and fenced receipts while preserving Task Runtime as the sole lifecycle authority.
- Added reconnect, certificate-rotation-window and local readiness behavior without an Agent inbound listener. Error
  output and Task logs use fixed diagnostics and never include endpoints, credential paths, host paths or commands.
- Kept Application, Container, Build and Update Docker operations unchanged for their explicitly later batches.
- Validation passed: focused behavior and conformance-tag tests, Agent race tests, SQL/version/ai-plan gates,
  `git diff --check` and the complete `graft validate backend` entrypoint.

## 2026-08-21 batch-4-accepted

- Assigned Application Compose lifecycle and finite Container mutations to domain-owned provider-neutral intents and
  Task-owned external Stages. Task Runtime remains the only owner of lease, fence, renewal, cancellation, bounded logs,
  receipt, retry and recovery transitions.
- Extended Runtime Target binding authority from identity-only rows to immutable generation snapshots. Claim and all
  post-claim endpoints compare the authenticated certificate and the frozen provider/capability/version with the active
  generation, covering capability expansion, removal, version mismatch and reconnect recovery without credential drift.
- Added strict Moby/Compose SDK dispatch for the Application and Container operation allowlists, fair capability polling,
  fixed redacted logs and a mode-0600 execution journal. Application workspace/config/env material is resolved only
  after a fenced claim and is neither embedded in Stage input nor persisted in Task, logs or receipts.
- Removed Application and Container server-local mutation executors, Docker CLI command assembly, command-bearing
  OpenAPI/Web contracts, legacy adapters, fallbacks and aliases. Kept snapshot reads and interactive streams on their
  ADR-026 transport boundaries and retained the server Docker socket for Build SDK, Update Controller and remaining
  explicitly unmigrated runtime transports.
- Validation passed: focused and race tests across Agent, HTTP transport, Task, Application, Container and Runtime
  Target; OpenAPI and generated-Web freshness checks; SQL/version/AI-plan/shared-asset gates; full Web validation;
  `git diff --check`; and the complete backend validation entrypoint.

## 2026-08-21 batch-5-build-sdk-accepted

- Build SDK execution is converging on the existing Task Runtime lease seam: provider `docker`, capability `oci-build`,
  capability version `docker/v1`, protocol `build-execution/v1` and operations `build.image.local.v1`,
  `build.image.publish.v1`, `build.manifest.publish.v1` and `build.artifact.copy.v1`.
- Build material and normalized result are transient and fence-bound. Material is resolved after claim; result is sent
  before terminal receipt. Task persists only the result digest and remains sole lease/receipt/retry/recovery owner;
  Build remains sole Artifact/Publication/result interpretation owner.
- Server-local Build Docker/CLI execution is being removed. Server and Agent share the named
  `/tmp/graft-build-snapshots` volume. Agent remains outbound-only with no published port. Server socket retention is
  limited to unmigrated Update Controller, Runtime Target discovery/summary and Container read/stream/interactive
  boundaries.
- Validation passed: focused and race Go tests for Agent, HTTP transport, Task, Build and Runtime Target; complete
backend validation including lint; SQL migration/version checks; generated module registry freshness; AI-plan
structure validation; and `git diff --check`. Build server-local Docker/CLI paths and duplicate adapters were deleted.

## 2026-08-21 batch-6-update-controller-launch-boundary-accepted

- Inventoried the Update Controller Docker/CLI launch path. The normal server-side `Launch` method was deleted; the
  retained observer/recovery capability remains only for the explicitly unmigrated recovery/observation boundary.
- Update now submits `controller_launch` as a Task-owned external stage bound to the selected Runtime Target ID,
  provider `docker`, capability `update_controller`, capability version `docker/v1`, protocol
  `platform-update-controller/v1`, and a digest of the provider-neutral launch intent. A final
  `controller_result` receipt stage remains the single terminal handshake for the controller's durable state volume.
- Update resolves digest-pinned controller reference, Compose root, socket path, state volume and encoded runner input
  only after a valid fenced material request. The material is kept in memory for reconnect/re-authorization and is
  removed when the Update operation settles; it is not written to Task, Agent journal, logs, receipts or durable Update
  state.
- Agent capability admission and post-claim transport checks continue to enforce active generation identity, provider,
  capability and version. The Agent launches with fixed mounts, `none` network, read-only rootfs, `ALL` capabilities
  dropped plus `CHOWN`, no inbound listener and stable redacted failure codes.
- Runtime Target local Docker declarations now include `update_controller`; Container read/stream/interactive and
  Runtime Target discovery/summary remain on the retained server socket boundary. No fallback, dual execution or
  compatibility alias was introduced.
- Validation passed: focused Update/Agent/Runtime Target/Task tests, matching race tests, production lint, AI-plan
  structure validation, `git diff --check`, and the complete backend validation entrypoint.

## 2026-08-22 batch-7-deployment-and-cli-deletion-docs

- Kept `graft update cutover-v1` explicit in the official Compose bootstrap and smoke overlay as the one-time legacy
  update-state cleanup authority before forward-only migrations; it is not a second runtime launch path.
- Synchronized deployment and conformance guidance on the frozen server socket consumer list: Update
  observation/recovery, Runtime Target discovery/summary, and Container snapshot/stream/interactive reads. Agent-owned
  Update Controller launch, Build, and finite mutations do not use the server socket path.
- Documented the required self-update rollout order: server/web replacement, server health verification, Runtime Agent
  replacement last, then mTLS identity, generation, and capability-readiness verification.
- Validation passed: `git diff --check` and `python3 scripts/validate_ai_plan_structure.py`.
