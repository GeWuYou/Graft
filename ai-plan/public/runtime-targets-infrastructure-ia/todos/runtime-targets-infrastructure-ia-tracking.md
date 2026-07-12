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

- Batch 4 replaced the container-list user-facing "source" filters with independent `deployment_type` and `runtime_target_id` contract fields.
- The Docker container page has no redundant Provider selector: deployment type is `Standalone` or `Compose`, and runtime target comes from the Runtime Target authority.
- The next bounded batch is Docker resource pages and Application/Compose integration; it must preserve Project ownership of Compose lifecycle.

## Task Checklist

- [x] work-intake-and-authority-foundation
- [x] menu-section-contract-and-sidebar-rendering
- [x] runtime-target-foundation
- [x] container-deployment-type-and-target-filter
- [x] docker-resources-and-application-integration
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
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering", "runtime-target-foundation", "container-deployment-type-and-target-filter"],
  "pending_batches": ["docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "docker-resources-and-application-integration",
  "next_batch": "cross-boundary-acceptance-and-archive-readiness",
  "closeout_status": "batch-5-complete"
}
```

## Batch 5 Completion

- Docker Resources exposes read-only Images, Networks, Volumes, and System APIs plus page-local Docker tabs; no provider selector, writes, cache, Registry, or credential persistence was introduced.
- Project remains the Compose lifecycle authority. Container and Project views only expose deployment context, target facts, and canonical Project navigation.

## Batch 4 Completion

- OpenAPI, generated server/web contracts, service mapping, and container UI now expose `deployment` plus `runtime_target` without retaining user-facing source-scope controls.
- Container filters serialize only deployment type, runtime target, status, health, and keyword. Compose, standalone, and unknown deployment presentation are covered by focused tests.
- Cache decision: no cache. The container list remains a live Docker runtime read; Runtime Target selection resolves persisted target identity through the narrow reader interface. Reconsider caching only after measured list-read latency or fan-out makes this path a hotspot.
- Validation: OpenAPI drift checks, focused Go and Vitest suites, `cd server && go run ./cmd/graft validate backend`, frontend format/type/i18n/lint/style/hygiene/build gates, and `git diff --check`. The full `bun run check` test wave had two unrelated timing failures; both files passed when retried directly.
