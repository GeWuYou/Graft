# Docker Runtime Agent Trace

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
- Retained the legacy final-stage receipt writer only as an explicitly temporary bridge until Update Controller migration.
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
