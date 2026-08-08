# Build Domain v2 Loop Startup Prompt

Continue the `build-domain-v2` topic using `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/build-domain-v2/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - `ai-plan/AGENTS.md` when changing topic/design/roadmap materials
- design authority:
  - `ai-plan/design/architecture/build-domain-v2.md`
  - `ai-plan/design/architecture/build-domain-v2-credential-and-telemetry-authority.md`
  - `ai-plan/design/architecture/build-domain-v2-provider-sdk-spi.md`
  - `ai-plan/roadmap/build-domain-v2.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-batch`
  - `$graft-validation-runner`

Topic objective:

- Deliver the approved staged Build Domain v2 without reintroducing Docker-first input, duplicate Runtime Target
  connections, or a second execution runtime.

Work contract summary:

- Long-running cross-boundary feature; repository-level design, roadmap and recoverable topic are required; execution
  uses `$graft-multi-agent-loop` and may use bounded `$graft-multi-agent-batch` slices.

Locked decisions:

1. Submission freezes Workspace Snapshot and Execution Plan before Task Runtime submission.
2. Runtime Target owns physical build capability, Task Runtime owns execution, and Artifact is an immutable first-class
   resource independent of Build Job.
3. Registry Connection selects a credential reference only; `CredentialProvider` issues short-lived credentials and
   `RuntimeExecutionAdapter` injects and removes them in an isolated provider context. Default Docker credential stores
   and environment-default authentication are forbidden for new execution.
4. Builder/Registry-local failure is local Build capability failure, not global platform availability. Build-owned
   Reservation is a fenced capacity lease, never a second Task state machine.

Implementation guardrails:

- Repair the highest available authority first and promote the task to cross-boundary when shared contracts change.
- Do not introduce compatibility fields for Docker-first writes, arbitrary host paths, endpoint details or credentials.
- Follow the RFC's four release gates: manual single Builder; completed Build intent/materialization; static Pool
  placement; then real provider telemetry, dynamic policy and Task Runtime-owned distributed Build.
- `RuntimeTargetBuilderTelemetryReader` is a facade, not a telemetry source. UI, Monitor, Docker/host metrics and Task
  JSON are diagnostic only. `least_load`, `capacity`, `affinity` and `region` remain disabled until Phase 4 conformance.
- A `build-snapshot` locality declaration alone never permits a host path, local Docker fallback or remote execution.
- The user has pre-authorized normal in-scope `execute_repair` actions for this topic. Do not pause for repeated repair
  confirmation; retain the normal authority, ownership and validation checks.
- Treat Build-owned selector read APIs and the controlled create flow as completed Phase 10; do not reintroduce manual
  opaque ID inputs or direct Runtime Target frontend API imports.
- Consume the Work Contract already persisted in tracking; do not repeat Work Intake during normal batches.

Current batch plan:

1. Phase 1, Phase 2 and Phase 3 release gates are complete. Phase 4a has registered a durable Builder
   Agent/control-plane ingress and telemetry provider; signed reports are verified against target-bound Agent public
   keys before persistence. Reservation recovery and Task Runtime distributed-leg gaps still block Phase 4 acceptance.
2. Do not promote the telemetry provider alone, historical Pool/Docker adapter work, or a persisted observation into
   dynamic-policy acceptance evidence. Preserve immutable Plan/Placement authority throughout each bounded phase.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the batch and authority owner changed this round.
- Update tracking and trace in the same change.
- Run the smallest required validation and scoped commit evaluation.
- Do not emit a next-session startup prompt except for a controller-selected terminal state.
