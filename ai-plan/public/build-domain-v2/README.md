# Build Domain v2

## Current Status Summary

- Topic objective: replace the Docker-first Build Center authority with an immutable, Runtime Target-capability based
  Build Domain that can evolve across all four approved phases.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running cross-boundary feature requiring repository design, roadmap, active topic recovery and
  loop-driven phased delivery.
- Canonical authority:
  - `ai-plan/design/architecture/build-domain-v2.md`
  - `ai-plan/roadmap/build-domain-v2.md`
- Completed so far: Work Intake and authority/bootstrap are in progress in the current loop batch.
- Not started yet: Phase 1 runtime, infrastructure, server, contract and web implementation.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: Runtime Target owns build capability, Build owns immutable source/plan/artifact authority, and
  Task Runtime owns execution state.

## Owned Scope

- `ai-plan/design/architecture/build-domain-v2.md`
- `ai-plan/roadmap/build-domain-v2.md`
- `server/modules/build/**`, Runtime Target, Infrastructure registry owners, Task, OpenAPI and Build web contracts.

Out of scope:

- Implementing a Graft Registry server or exposing Runtime endpoint/credential details.
- Creating a second Build execution, queue, log or realtime runtime.

## Locked Decisions

1. Build consumes immutable Workspace Snapshots and Execution Plans; retry never refreshes source.
2. Artifact is first-class and immutable; Publication binds it to mutable tags, manifests, promotions or copies.
3. Runtime Target reports physical build capability; Builder Profile, Instance and Pool are Build-owned logical resources.
4. OCI Registry is the Phase 1 deployment-grade Destination, but Artifact Destination remains provider-neutral.

## Phase Plan

- `authority-bootstrap`
- `phase-1-single-builder`
- `phase-2-workspaces-templates-drivers`
- `phase-3-pools-scheduling-platforms`
- `phase-4-artifact-supply-chain-automation`

## Current Recovery Point

- Current loop batch establishes repository authority, roadmap and recovery materials only.
- Legacy Docker Build Center history is preserved, but its new-write model is superseded.
- Next step: the loop controller settles this bootstrap batch and selects the bounded Phase 1 implementation slice.

## Work Intake

- This topic was created through `Work Intake`.
- Persisted Work Contract: `ai-plan/public/build-domain-v2/todos/build-domain-v2-tracking.md`.

## Pending Batch Direction

- Phase 1 must establish upstream Runtime capability, Registry Connection/Repository and v2 contract authority before
  a Build UI migration.
- Later phases must be dispatched only after their upstream contracts and prerequisite execution capabilities exist.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/build-domain-v2/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
