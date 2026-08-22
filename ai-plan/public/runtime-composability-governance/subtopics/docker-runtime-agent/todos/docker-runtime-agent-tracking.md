# Docker Runtime Agent Tracking

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: subtopic `runtime-composability-governance/docker-runtime-agent`
- authority summary: Task Runtime owns external execution lifecycle; Runtime Target owns Agent identity and capability
  binding; Docker Runtime Agent executes Provider side effects; Update Controller owns self-update state across replacement.

## Work Intake Result

- classification: long-running refactor
- existing topic reused: `runtime-composability-governance`
- artifacts required: repository design updates, ADR-026, roadmap extension, bounded subtopic recovery and phased implementation
- parallel top-level topic: rejected
- implementation engine: direct bounded batches with validation and scoped commit after each accepted batch

## Semantic Review Selection

- platform architecture: authority, runtime privilege and Task/Agent/Controller boundaries
- module architecture: narrow Provider/Gateway contracts and dependency direction
- domain model: Stage attempt, lease, receipt, cancellation and recovery invariants
- event contract: durable facts versus realtime notifications
- table design and SQL migration: Task-owned lease persistence and constraints
- test seam: repository/Runtime behavior, restart and expiry coverage
- consistency and delete review: stable vocabulary and removal of final-stage/CLI assumptions
- cross-boundary/OpenAPI/API DX: deferred until a batch changes published Agent or Web contracts

## Locked Decisions

1. One always-on `docker-runtime-agent`; direct rename from the experimental Builder Agent, no dual Agent or alias.
2. Agent pull is the sole work feed. Agent owns no queue, retry policy, Task state or business persistence.
3. Server ultimately has no Docker socket or Docker/Compose/buildx executable dependency.
4. Task Runtime owns external execution lease, bounded logs, cancellation observation, receipt and expiry recovery.
5. Update Controller remains separate and replaces Runtime Agent last.
6. Docker adapters converge on SDKs; any CLI adapter is a temporary bridge with owner and deletion trigger.

## Batch State

```json
{
  "completed_batches": [
    "batch-1-architecture-authority-and-recovery",
    "batch-2-task-runtime-external-execution-foundation",
    "batch-3-docker-runtime-agent-promotion",
    "batch-4-application-and-container-migration",
    "batch-5-build-sdk-migration",
    "batch-6-update-controller-launch-boundary",
    "batch-7-deployment-and-cli-deletion",
    "batch-8-ui-and-cross-boundary-convergence",
    "batch-9-runtime-boundary-closeout"
  ],
  "current_batch": null,
  "pending_batches": [],
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```

## Batch 1 Acceptance

- ADR-026 fixes the single Agent, Task-owned lease, server no-socket and independent Update Controller decisions.
- Compose, Build Provider SPI, Agent protocol, self-update and project-layout authority no longer prescribe the old
  server-local CLI or split Builder/Runtime Agent model.
- Parent and subtopic recovery materials identify the same current batch and authority.
- `git diff --check` and `python3 scripts/validate_ai_plan_structure.py` pass.

## Batch 2 Acceptance

- Task Runtime persists one fenced external execution lease per Stage attempt with database-enforced identity and state.
- Provider-neutral APIs support claim, renew, cancellation observation, bounded Stage logs and idempotent receipt.
- External receipt success can advance a non-final Stage without prematurely finalizing the Task.
- Expired running leases and interrupted external work enter `unknown`/`needs_attention`; no unsafe automatic replay.
- Migration has complete comments, preflight metadata and upgrade validation; focused Task tests and backend validation pass.

## Batch 3 Acceptance

- The experimental package, binary, image, configuration, development topology and conformance fixture are promoted
  directly to the single `docker-runtime-agent`; no old runtime entry, alias, process or optional Compose profile remains.
- Runtime Target persists an explicit capability binding and execution claim admits only the authenticated Agent's
  provider, capability and protocol version. The old single profile is copied once and is not an execution fallback.
- The Agent performs mTLS enrollment/reconnect, bounded long-poll claim, lease renewal, cancellation observation,
  fixed redacted logs, fenced receipt settlement, certificate rotation and local readiness without opening an inbound port.
- Application, Container, Build and Update Docker operations remain at their pre-Batch-3 owners for later migration.
- Focused Agent/transport/Runtime Target/Task behavior tests, Agent race tests, migration gates and full backend validation pass.

## Batch 4 Acceptance

- Application Compose lifecycle and finite Container mutations now submit provider-neutral intent as ordered external
  Stages and reach the Docker Runtime Agent only through Task-owned fenced leases. Application and Container retain
  domain intent and result interpretation; Task Runtime remains the sole owner of Task, Stage, lease, renewal,
  cancellation, logs, receipt, retry and recovery state.
- Runtime Target capability bindings are generation-scoped. Claim and every post-claim operation re-authorize the
  frozen provider, capability and version against the authenticated active generation, so old credentials, missing
  capabilities and version mismatches fail closed.
- The Agent strictly dispatches the Application and Container operation allowlists through the Moby and Compose SDKs,
  persists only a mode-0600 recovery journal, emits fixed redacted diagnostics and resolves workspace material through
  a lease-fenced transient endpoint that never writes paths, endpoints, credentials or commands to Task persistence.
- Server-local Application and Container mutation execution, CLI command contracts, adapters, fallbacks and
  compatibility aliases are removed. Container snapshot reads and interactive streams remain outside the Task lease
  protocol as required by ADR-026; Build SDK and Update Controller execution remain unmigrated, so the server Docker
  socket mount is intentionally retained.
- Focused Application, Container, Task, Runtime Target, HTTP transport and Agent behavior tests, related race tests,
  OpenAPI/generated-Web checks, migration gates, full Web validation and the complete backend validation entrypoint pass.

## Batch 6 Acceptance

### Launch-boundary inventory

| Entry | Batch 6 disposition | Task Stage / capability | Intent and result owner |
| --- | --- | --- | --- |
| Normal digest-pinned Update Controller start (`ComposeRunnerLauncher.Launch`) | migrated; server method deleted | `controller_launch`; `docker/update_controller@docker/v1`; `platform.update.controller.launch.v1` | Update freezes version/image/operation intent and maps Task failure or controller receipt; Task Runtime owns launch lease/receipt state; Agent owns Docker start |
| Controller terminal handshake | retained as the only final Stage, not a launch fallback | `controller_result`; `compose-runner/v2` final receipt | Controller state volume is durable evidence; Update interprets it; Task Runtime owns final receipt settlement |
| Pre-migration recovery controller (`LaunchRecovery`) | explicitly not migrated in this batch | no new Stage or capability added | Update recovery claim/state invariants remain authoritative; Batch 7 owns removal or migration of the retained Docker recovery boundary |
| Controller-internal Docker/Compose CLI sequence | not a controller startup entry and unchanged | none | Update Controller still owns backup/pull/migrate/recreate/health behavior; later CLI-deletion work must not reopen normal launch |
| Bootstrap `graft update cutover-v1` command | not a controller startup entry and unchanged | none | Deployment bootstrap/cutover compatibility authority remains outside Batch 6 |

- [x] Update submits a two-stage Task plan: `controller_launch` is a fenced Runtime Agent execution stage bound to the
  selected Runtime Target generation, while `controller_result` remains the sole final receipt handshake for the
  durable Update Controller state volume.
- [x] Update owns update intent, version/image semantics, transient material resolution and receipt interpretation;
  Task Runtime owns Task/Stage/lease renewal, cancellation, bounded logs, receipt settlement, retry and recovery.
- [x] `docker-runtime-agent` declares `update_controller` at `docker/v1`, claims only the exact frozen operation and
  protocol, and starts the digest-pinned controller with fixed mounts, no network, a read-only rootfs and stable
  failure codes. The Agent has no inbound listener and does not persist material, credentials, endpoints, host paths or
  commands.
- [x] Runtime Target local Docker capability declarations include `update_controller`; generation and post-claim
  re-authorization remain enforced by the shared Agent transport for version mismatch, capability removal, certificate
  rotation and reconnect recovery.
- [x] The server-side normal launch method was removed from the old Docker launcher. Container read/stream/interactive
  paths, Runtime Target discovery/summary and the bounded recovery observer remain on the retained server Docker socket
  boundary for later batches; no fallback, dual execution or compatibility alias was added.
- [x] Focused Update/Agent/Runtime Target tests, full Go tests, race coverage and backend validation evidence are
  recorded by the main agent.

Evidence: focused Update/Agent/Runtime Target/Task tests, the same packages under `go test -race`, production
`golangci-lint`, `python3 scripts/validate_ai_plan_structure.py`, `git diff --check`, and the complete
`go run ./cmd/graft validate backend` entrypoint all passed.

## Next Recovery Point

Batch 9 is archive-ready. There is no pending migration batch. Future work must start from a new authority-led slice and
use the recorded deletion trigger for the specific retained Update, Runtime Target, Container/Deployment observation or
interactive transport boundary; it must not reopen the completed Application/Container/Build/Update execution paths.

## Batch 8 UI And Cross-Boundary Convergence

### Scope

- Reframe the Application detail route as the `list-form-detail` explain surface, retaining identity, Runtime Target,
  lifecycle state, service snapshot, configuration and Task history while removing Application-local CPU, memory and
  network dashboard placeholders.
- Present Runtime Target Agent status, per-capability readiness, implementation version and stable diagnostic codes from
  the canonical Runtime Target projection; keep resource metrics on Runtime Target or Container surfaces.
- Gate Application and Container actions on the capability projection and render stable localized disabled reasons.
- Map Task Agent/Provider failure codes to product copy; raw SDK errors remain only in redacted server diagnostics.
- Use `?url` for the Docker SVG asset and collapse header actions into an overflow menu in narrow containers.

### Acceptance

- [x] Application detail uses `list-form-detail` with primary `explain` intent and secondary `workflow`/`feedback`
  checks; loading, empty, error, disabled and destructive states retain stable layout and localized copy.
- [x] Application detail has no authoritative-less CPU, memory or network dashboard; service snapshot remains a
  bounded Container-owned observation and Runtime Target/Container own runtime metrics.
- [x] Runtime Target detail shows Agent status, each bound capability readiness, version and stable diagnostic code;
  endpoint/credentials/host paths and raw provider diagnostics never reach Task/API/UI.
- [x] Application and Container actions are disabled when the required capability is unavailable, with a stable local
  reason; no page-local authority or compatibility bridge is introduced.
- [x] Task detail renders stable failure-code product messages and keeps raw SDK/provider text out of API/UI payloads.
- [x] TDesign Vue Next Button, Dropdown, Tabs, Card, Alert and Tag usage is verified against official docs because MCP
  is unavailable; responsive overflow behavior is covered by tests.
- [x] Web typecheck, unit tests, `bun run check`, OpenAPI bundle/generated-Web freshness, relevant backend validation,
  `git diff --check` and `python3 scripts/validate_ai_plan_structure.py` pass.

### Verification plan

- `cd web && bun run check` plus focused Application, Runtime Target, Container and Task tests.
- OpenAPI bundle and generated projection freshness checks when the Runtime Target response contract changes.
- `cd server && go run ./cmd/graft validate backend` when server/OpenAPI authority or generated Go bindings change.
- `git diff --check` and `python3 scripts/validate_ai_plan_structure.py` for recovery/documentation integrity.

Verification evidence: `cd web && bun run check` (311 files / 2161 tests), focused Application, Runtime Target and Task
tests, `bun run typecheck`, `bun run openapi:types:check`, `python3 scripts/openapi_generated_freshness_check.py`,
`cd server && go test ./modules/runtime-target/...`, `cd server && go run ./cmd/graft validate backend`,
`python3 scripts/validate_ai_plan_structure.py`, and `git diff --check` all passed. The OpenAPI 3.1 warning from
`oapi-codegen` remains the repository's known non-blocking warning; generated artifacts are fresh.

### Recovery record

- Startup receipt: governance source `AGENTS.md`; task class `cross-boundary`; recovery source this Docker Runtime Agent
  subtopic; owned scope `batch-8-ui-and-cross-boundary-convergence`.
- Authority decision: Runtime Target owns Agent/capability readiness projection, Task Runtime owns failure/receipt facts,
  and pages only map stable wire values to localized product copy.
- TDesign MCP preflight: unavailable in this environment; official Vue Next documentation fallback is required and will
  be recorded again at closeout with the queried components and adoption decision.
- TDesign fallback evidence: official Vue Next Button, Dropdown, Tabs, Card, Alert and Tag documentation was queried;
  the installed Vue Next type/runtime sources were checked for the adopted APIs and DOM class conventions (including
  `DropdownItemTheme`). No callable Vue Next MCP tool was present, so this fallback is explicit rather than an
  unverified component assumption.

### Closeout

- Batch 8 is accepted and committed as the UI/cross-boundary convergence slice. The next recovery point is Batch 9;
  no Batch 9 implementation was started in this turn.

## Batch 5 Acceptance

- [x] Every Build Docker operation reaches the Agent through a Task-owned external execution lease and the exact
  provider/capability/version binding.
- [x] Material and normalized result are transient, fence-bound and redacted; Task retains only `result_sha256`, and
  Build alone owns Artifact/Publication/result interpretation.
- [x] Agent SDK tests cover capability admission, Moby/OCI build/push/manifest/copy behavior, cancellation, retry and
  recovery; duplicate result replay is idempotent and conflicting replay is rejected.
- [x] No Build server-local Docker/CLI execution, fallback, old adapter or compatibility alias remains.
- [x] Deployment/conformance docs describe the shared snapshot volume and no Agent inbound port; server socket retention
  is explicitly limited to unmigrated Update/Container read-stream boundaries.
- [x] Focused, race, migration, AI-plan and complete backend validation evidence is recorded by the main agent.

Evidence: focused and race Go tests, `go run ./cmd/graft validate backend` (including lint), SQL migration/version
checks, generated module registry freshness, AI-plan structure validation and `git diff --check` all passed.

## Batch 7 Deployment and CLI Deletion Evidence

- [x] Official Compose bootstrap and smoke overlay retain `graft update cutover-v1` as the one-time legacy update-state
  authority before `graft migrate up`; cutover is not a runtime launch or compatibility execution path.
- [x] Deployment and conformance documentation state the only retained server Docker socket consumers: Update
  observation/recovery, Runtime Target discovery/summary, and Container snapshot/stream/interactive transport. Build,
  Update Controller launch, and finite mutation side effects remain Agent-owned.
- [x] Official migration guidance fixes the ordered Update rollout: replace `server`/`web`, verify server health,
  replace the Runtime Agent last, then verify mTLS identity, active generation, and capability readiness.
- [x] Conformance fixture documentation and Compose comments preserve the outbound-only Agent/no inbound port boundary
  and the retained server socket list.

Validation evidence for this documentation/configuration slice: `git diff --check` and
`python3 scripts/validate_ai_plan_structure.py`.

## Batch 9 Runtime Boundary Closeout

### Scope and authority decision

Batch 9 is a closeout/evidence slice. It does not redesign Runtime Target, Task Runtime, the Agent protocol, or the
Update Controller. The authority chain remains:

- Task Runtime owns the external execution lease, renewal, cancellation observation, bounded logs, transient result
  digest, receipt settlement, retry and recovery.
- Runtime Target owns Agent identity, generation, target binding and capability admission.
- `docker-runtime-agent` owns all Docker/Moby/OCI side effects reached through a Task lease.
- Update Controller owns durable self-update state and its survivor/recovery protocol.
- Container owns resource semantics and authorization; Deployment Runtime owns deployment-context interpretation.

### Final server Docker socket consumer inventory

The inventory was cross-checked against handwritten Go, Compose/deployment fixtures, focused tests and the normative
design/deployment documents. The server socket is retained only for bounded observation/projection/read transport:

| Consumer and code authority | Evidence | Disposition | Retention reason, risk and deletion trigger |
| --- | --- | --- | --- |
| Update runner receipt/progress/failure observation and settled-runner cleanup (`server/modules/update/launcher.go`, `RolloutService`) | `NewDockerComposeRunnerLauncher`; `ReadRunnerReceipts`, `ReadRunnerProgress`, `ReadRunnerFailures`, `RemoveRunner`; `launcher_test.go` and `operation_test.go` | Retain temporarily | Update must reconcile the short-lived Controller after server recreation and remove a container only after a settled receipt. High socket privilege; delete when receipt/state projection is Agent-initiated or durable without server Docker access. |
| Runtime Target local Docker discovery (`server/modules/runtime-target/discovery.go`) | `discoverLocalDocker`, `pingLocalDocker`; `discovery_test.go`, `audit_transaction_test.go` | Retain temporarily | System-managed local target bootstrap and availability fact. Delete when Agent enrollment/readiness becomes the authoritative target discovery input. |
| Runtime Target target summary (`server/modules/runtime-target/summary.go`) | `dockerTargetSnapshotCollector`; `summary_test.go` and summary collection tests | Retain temporarily | Bounded version/info/workload/image/volume/network/host-usage projection. Delete when Agent telemetry supplies the equivalent bounded summary. |
| Container snapshot/read, stats, runtime events, logs and stream (`server/modules/container/docker_runtime*.go`, `docker_resources.go`) | `docker_runtime_test.go`, `docker_resources_test.go`, `runtime_events_test.go`, service route tests | Retain temporarily | Container module owns read semantics, redaction, authorization and realtime transport; these are not Task stages. Delete when the Agent-initiated transport-only channel covers the same API/read semantics. |
| Container interactive/exec (`server/modules/container/docker_exec_session.go`, `docker_runtime.go`) | `docker_runtime_test.go`, shell route/websocket tests | Retain temporarily | Interactive transport must not be disguised as a Task or Agent queue. Delete when an Agent-initiated interactive channel preserves the current authorization and session lifecycle. |
| Current server container facts used by Deployment Runtime (`server/modules/container/docker_facts_provider.go`) | `docker_facts_provider_test.go`, `server/modules/deployment/runtime.go` | Retain temporarily | Deployment Runtime owns deployment-context interpretation while Container exposes the raw inspect projection. Delete when an explicit provider/Agent deployment projection replaces server-local inspect. |

The explicit development CLI (`server/internal/cli/dev_docker_runtime_agent.go`) invokes `docker compose`/`docker cp` only
to prepare the local development fixture. It is not a production `serve` launch path, does not execute Application,
Container, Build or Update mutations, and is therefore recorded as a retained repository CLI entrypoint rather than a
server runtime socket consumer. Its deletion trigger is a future developer-topology change that provides equivalent
fixture preparation without a local Docker CLI; it is not a runtime fallback.

The short-lived `server/cmd/graft-compose-runner` and `server/runner/compose` remain Update Controller code and use the
Moby/official Compose SDK inside the Agent-launched process. A repository-wide search found no `exec docker`,
`docker compose`, `docker-compose` or Buildx invocation in the migrated server mutation paths. The only remaining
`exec.CommandContext(..., "docker", ...)` calls are the explicit development CLI above; shell scripts and conformance
fixtures are test/deployment harnesses, not server runtime paths.

### Mutation and duplicate-path audit

- Application Compose lifecycle stages are created by `server/modules/project/service_lifecycle.go` and claimed only by
  Task Runtime external execution; no server Docker client is reachable from that mutation path.
- Container finite mutations and image/network/volume changes are submitted by `task_lifecycle.go` and
  `task_docker_*.go`; Agent provider dispatch owns the Moby side effect. No server-local mutation executor, CLI command
  assembler, fallback or compatibility alias remains.
- Build operations are submitted by `server/modules/build/v2_submission.go`; Build material/result remain transient
  and Task persists only the result digest. Build SDK calls exist only under `server/agents/docker-runtime-agent`.
- Update Controller normal launch is submitted as `controller_launch` through the Task lease. The retained Update
  launcher is observation/recovery/settled cleanup only; `cutover-v1` is a one-time migration/bootstrap authority and
  cannot start runtime work.
- No second scheduler, Task state machine, Agent queue, server push path or hidden launch fallback was found.

### Batch 9 acceptance and verification

- [x] Code/deployment/test/document inventory is complete and the retained consumers above have explicit authority,
  lifecycle, risk and deletion triggers.
- [x] All finite Application, Container, Build and normal Update Controller mutations cross a Task-owned external
  execution lease and Runtime Agent capability; no server-local mutation path remains.
- [x] Positive retained-boundary tests cover Update receipt/progress/failure observation, Runtime Target discovery and
  summary, Container read/log/stream/exec/resource snapshots, and Deployment Docker facts.
- [x] Static searches confirm no migrated server path invokes Docker/Compose/Buildx CLI or a second launch path.
- [x] Deployment and conformance fixtures keep the Agent outbound-only/no-inbound-port boundary and document the
  retained server socket consumers.
- [x] `git diff --check` and `python3 scripts/validate_ai_plan_structure.py` pass; focused server and race tests plus
  the complete backend validation entrypoint pass.

### Closeout decision

Batch 9 is `archive-ready`: retained socket consumers are intentional, bounded and evidence-backed rather than
unresolved migration work. The cleanup triggers above are the next authority-led migration points; no new Batch 10 is
created by this closeout. The parent topic may enter the normal archive workflow after this scoped commit is reviewed.

## 2026-08-22 Batch 9 Recovery Record

- Startup receipt was rerun from root `AGENTS.md` with task class `cross-boundary`, recovery source
  `runtime-composability-governance/docker-runtime-agent`, owned scope
  `docker-runtime-agent-batch-9-runtime-boundary-closeout`, and the ADR-026 authority summary.
- HEAD was confirmed as `b72dcb1108d8dfe0fd5de899b72c28fe09fa1e84`. The checkout contained pre-existing file-mode-only
  changes in `.claude/skills`, Husky scripts, smoke scripts and conformance shell fixtures; they were preserved and are
  excluded from this Batch 9 scope.
- Batch 8 receipt continuity remains recorded as `current_batch=Batch 8` in the trace; Batch 8 is accepted. Batch 9 is
  the only implementation/closeout batch in this turn.
