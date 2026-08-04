# Network Connectivity Diagnostics

## Current Status Summary

- Topic objective: deliver target-based platform connectivity diagnostics and a batch health-check surface.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running feature with design, roadmap, ADR, and loop-managed implementation phases.
- Canonical authority:
  - `server/modules/network/**`
  - `openapi/**`
  - `web/src/modules/network/**`
- Completed so far: Work Intake bootstrap and Phase 1 authority/probe-core contract.
- In progress next: Phase 2 persistence, APIs, and security boundary.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: the platform-network module, canonical OpenAPI source, and network web module jointly own the feature.

## Owned Scope

- `server/modules/network/**`, `server/internal/moduleapi/**`, and network-owned migrations
- `openapi/**`, generated contract closure, `web/src/modules/network/**`, network RBAC and i18n
- `ai-plan/design/domains/network/**`, `ai-plan/roadmap/**`, and the topic recovery materials

Out of scope:

- proxy-node, rule-graph, GeoIP, or proxy-client product behavior
- background runtime services or live migration execution from a numbered worktree

## Locked Decisions

1. Diagnostics routes identify a target: `/platform/network/connectivity/:targetId`; reports are data, not page identity.
2. Batch and diagnostics share one target registry, probe pipeline, report store, and web ConnectivityStore.
3. Probe results are capability-driven and extensible; Exit IP is masked by default and protected by a dedicated permission.

## Phase Plan

- Phase 1: design authority, target registry, capabilities, and extensible probe/report core.
- Phase 2: persistence, batch execution, diagnostics APIs, SSRF protection, and permissions.
- Phase 3: shared web ConnectivityStore, batch health UI, target diagnostics UI, and cross-boundary validation.

## Current Recovery Point

- Loop mode: `topic-completion-loop`.
- Current batch: controller-owned; consult the tracking document after startup preflight.
- Next step: the loop controller evaluates this round's closeout before dispatching the next bounded worker.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is in `todos/network-connectivity-diagnostics-tracking.md`.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/network-connectivity-diagnostics/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
