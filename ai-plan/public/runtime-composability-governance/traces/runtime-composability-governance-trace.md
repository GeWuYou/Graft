# Runtime Composability Governance Trace

## 2026-08-14 work-intake-design-bootstrap

- Classified the work as a long-running cross-boundary refactor: its eventual implementation can affect core Runtime, Modules, Providers, Agents, Task Runtime, realtime and typed capability boundaries.
- Created repository-wide design and roadmap authority plus the minimum active-topic recovery materials.
- Locked three principles: every runtime resource has creator/owner/disposer; capability visibility is bounded; composition units declare dependencies, exposed capabilities and cleanup responsibility.
- Rejected runtime Plugin Loader, nested dynamic configuration tree, Proxy Context hierarchy and HMR because Graft remains a compile-time Go modular monolith with persistent Task, Agent and external-resource authority.

## Locked Decisions

- A Resource Scope is only a narrow lifecycle tool for repeated cancellable resources; it cannot become a universal service context.
- Module dependencies, capability dependencies and resource ownership remain separate declarations.
- Process restart, configuration reconcile and Agent reconnect are preferred over runtime module reload.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-design-bootstrap"],
  "pending_batches": [
    "phase-0-resource-inventory",
    "phase-1-lifecycle-cleanup",
    "phase-2-narrow-resource-scope",
    "phase-3-capability-composition-declarations",
    "phase-4-controlled-change-evaluation"
  ],
  "current_batch": "phase-0-resource-inventory",
  "next_batch": "phase-1-lifecycle-cleanup",
  "closeout_status": "active"
}
```
