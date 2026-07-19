# Cross-Boundary Contract Projection Tracking

## Topic

Cross-Boundary Contract Projection

## Scope

Create an authority-aware, deterministic projection from existing OpenAPI and Go server contracts to web generated artifacts. The work covers metadata, generation, migration and drift gates, while preserving API compatibility and server runtime authority.

## Repository Truth

- `AGENTS.md`
- `ai-plan/design/governance/platform/跨边界契约投影设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: OpenAPI owns HTTP wire contracts; existing Go server contracts own non-HTTP values; web consumes generated artifacts only.
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
    - ai-plan/public/cross-boundary-contract-projection
    - ai-plan/design/governance/platform/跨边界契约投影设计.md
    - ai-plan/roadmap/跨边界契约投影实施计划.md
closeout:
  archive: true
  lessons_review: true
```

ADR is not required: this applies the existing authority-first and contract-governance rules rather than introducing a competing architectural baseline.

## Current Recovery Point

- Batch 0 established the active topic, repository-wide design and phased roadmap.
- Batch 1 added the platform contract projection foundation. Its descriptor metadata references existing typed Go constants, filters `visibility=web`, and emits deterministic TypeScript without changing canonical ownership.
- Next action: `pilot-migration`.

## Task Checklist

- [x] batch-0-contract-projection-intake
- [x] generator-foundation
- [ ] pilot-migration
- [ ] ci-integration
- [ ] broader-migration-and-final-archive-readiness

## Acceptance Conditions

- A web-visible non-HTTP server contract can be projected by referencing an existing Go constant, without a duplicate literal.
- OpenAPI remains the only wire authority and existing OpenAPI generation stays intact.
- Generated web values cannot expose server-only data, grant runtime authorization, or close API error/message value sets.
- CI detects derived-output drift and duplicate/visibility lifecycle violations through the existing validation chain.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["batch-0-contract-projection-intake", "generator-foundation"],
  "pending_batches": [
    "pilot-migration",
    "ci-integration",
    "broader-migration-and-final-archive-readiness"
  ],
  "current_batch": "generator-foundation",
  "next_batch": "pilot-migration",
  "closeout_status": "committed"
}
```
