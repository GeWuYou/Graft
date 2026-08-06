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

Implementation guardrails:

- Repair the highest available authority first and promote the task to cross-boundary when shared contracts change.
- Do not introduce compatibility fields for Docker-first writes, arbitrary host paths, endpoint details or credentials.
- Treat cross-Runtime Builder Pool placement as Phase 8 work. Phase 6/7 now own coordinated leg execution and manifest
  finalization; Phase 8 freezes placements but must not invent load, region or affinity evidence.
- Treat provider-backed Snapshot delivery and Remote Builder execution as Phase 9 work. Docker now supports validated
  Unix-socket and TCP/SSH provider paths; a `build-snapshot` locality declaration alone is never permission to use a
  host path or local Docker process for another Runtime Target. Phase 9C now includes aggregate Docker adapter registration,
  private connection authority and persisted conformance evidence; it still does not claim a concrete non-Docker provider.
  Kubernetes/BuildKit/Kaniko remain concrete Phase 9D adapters.
- The user has pre-authorized normal in-scope `execute_repair` actions for this topic. Do not pause for repeated repair
  confirmation; retain the normal authority, ownership and validation checks.
- Treat Build-owned selector read APIs and the controlled create flow as completed Phase 10; do not reintroduce manual
  opaque ID inputs or direct Runtime Target frontend API imports.
- Consume the Work Contract already persisted in tracking; do not repeat Work Intake during normal batches.

Current batch plan:

1. Preserve immutable Plan/Placement authority while completing the selected bounded Phase.
2. Phase 10 selector/read-model work is complete; continue with Phase 8A telemetry authority or Phase 9D provider proof.
   Phase 8A has a provider-neutral freshness/provenance contract but no source implementation; `least_load`, `region`
   and `affinity` remain fail-closed. Phase 9C's aggregate Docker foundation, private connection authority and evidence
   persistence are implemented. Phase 9B's Docker adapter may not be represented by a local fallback and must retain
   target-scoped provider authority.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the batch and authority owner changed this round.
- Update tracking and trace in the same change.
- Run the smallest required validation and scoped commit evaluation.
- Do not emit a next-session startup prompt except for a controller-selected terminal state.
