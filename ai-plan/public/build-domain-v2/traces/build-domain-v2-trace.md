# Build Domain v2 Trace

## 2026-08-06 authority-bootstrap

- Work Intake classified Build Domain v2 as a long-running cross-boundary feature requiring repository-level design,
  roadmap, active topic recovery and loop-driven phased delivery.
- The canonical model is `Workspace -> Workspace Snapshot -> Execution Plan -> Task execution -> Artifact ->
  Publication -> Deployment`.
- Docker-first direct create fields are historical-only; future writes use the v2 resource model.
- Builder capability remains Runtime Target authority; Builder Profile/Instance/Pool are logical Build resources and
  do not create another connection directory.

## Locked Decisions

- A retry uses the frozen Snapshot and Execution Plan; rebuilding current source is explicit new submission.
- Artifact identity is immutable digest evidence; tag, manifest, promotion and copy are Publications.
- Registry Connection and Artifact Repository are Infrastructure resources; OCI Registry is only the first Artifact
  Destination provider, not the universal output model.

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
