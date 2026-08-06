# Build Domain v2 Tracking

## Topic

Build Domain v2

## Scope

Replace the Docker-first Build Center with a platform Build Domain covering immutable source snapshots, execution plans,
Runtime Target build capability, artifact delivery, and the approved four-phase evolution.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/roadmap/build-domain-v2.md`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Runtime Target owns build capability; Build owns immutable Snapshots, Execution Plans, Artifacts and Publications; Task Runtime owns execution state.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/design/architecture/build-domain-v2.md
    - ai-plan/roadmap/build-domain-v2.md
    - ai-plan/public/build-domain-v2/README.md
    - ai-plan/public/build-domain-v2/startup-prompt.md
    - ai-plan/public/build-domain-v2/todos/build-domain-v2-tracking.md
    - ai-plan/public/build-domain-v2/traces/build-domain-v2-trace.md
    - ai-plan/public/README.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: `authority-bootstrap`; it has not yet been settled by the loop controller.
- Current authority work establishes the v2 design/roadmap, retires Docker-first new-write semantics and creates active
  recovery materials.
- Current risk: later implementation must widen from Build to Runtime, Infrastructure, Task, OpenAPI and Web whenever
  an upstream shared contract changes.

## Task Checklist

- [ ] Settle authority/bootstrap design, roadmap and topic recovery materials.
- [ ] Phase 1: Runtime build capability, Application Snapshot adapter, single Builder and OCI Registry publication.
- [ ] Phase 2: reusable Workspaces/Snapshots, Templates, Drivers and materialization.
- [ ] Phase 3: Builder Profiles/Instances/Pools, scheduling and multi-platform fan-out.
- [ ] Phase 4: promotion, OCI supply-chain evidence, remote/distributed builders and deployment/pipeline handoff.

## Acceptance Conditions

- Every submitted build has exactly one immutable Workspace Snapshot and Execution Plan.
- Retries retain their original source and plan; rebuild-current creates a new plan.
- Artifacts are digest-addressed, reusable outside Build Job and published through provider-neutral destinations.
- Runtime endpoint and credential details, arbitrary host paths and a second Build execution runtime never enter v2.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": [
    "authority-bootstrap",
    "phase-1-single-builder",
    "phase-2-workspaces-templates-drivers",
    "phase-3-pools-scheduling-platforms",
    "phase-4-artifact-supply-chain-automation"
  ],
  "current_batch": "authority-bootstrap",
  "next_batch": null,
  "closeout_status": "in-progress"
}
```
