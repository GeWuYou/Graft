# Runtime Targets Infrastructure IA Tracking

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Runtime Target authority, Infrastructure Section Label navigation, Provider resource boundaries, Container deployment semantics, and Application/Compose integration.
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
    - topic
    - design
    - roadmap
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Batch 1 completed Work Intake and repaired the repository authority that formerly prohibited Docker/Kubernetes/Podman Provider menus.
- The current design preserves the existing side navigation and defines Section Labels as visual-only metadata.
- No server, web, OpenAPI, generated artifact, skill, or script change was made in this batch.

## Task Checklist

- [x] work-intake-and-authority-foundation
- [x] menu-section-contract-and-sidebar-rendering
- [x] runtime-target-foundation
- [ ] container-deployment-type-and-target-filter
- [ ] docker-resources-and-application-integration
- [ ] cross-boundary-acceptance-and-archive-readiness

## Acceptance Conditions

- Infrastructure sidebar renders Section Labels without route, permission, or menu-node identity and hides empty groups.
- Runtime Targets are the sole connection and capability authority; Docker, Kubernetes, and Podman consume them.
- Docker Container list uses independent deployment type and runtime target filters without a redundant Provider filter.
- Application/Project retains Compose lifecycle authority; shared Registry and Certificates do not become Docker-private resources.
- Kubernetes and Podman menus appear only for authorized, capability-bearing Targets.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering", "runtime-target-foundation"],
  "pending_batches": ["container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "runtime-target-foundation",
  "next_batch": "container-deployment-type-and-target-filter",
  "closeout_status": "batch-3-complete"
}
```

## Batch 3 Completion

- Runtime Target contract and Local Docker persisted discovery are complete.
- Remote mTLS needs a future credential-vault authority; no remote credential storage is introduced in this topic batch.
