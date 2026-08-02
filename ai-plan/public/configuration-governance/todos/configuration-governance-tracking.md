# Configuration Governance Tracking

## Topic

Configuration Governance

## Scope

Introduce a versioned configuration contract for `.env`, resolved process configuration, official Compose topology, startup preflight, and CI validation.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md`
- `server/internal/config/**`
- `compose.yml`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: "An embedded versioned configuration schema owns deployment configuration lifecycle; runtime, Compose, templates, and CI consume its resolved contract."
requires:
  design: true
  topic: true
  roadmap: true
  adr: true
execution:
  engine: graft-multi-agent-batch
  dispatch_skill: graft-multi-agent-batch
bootstrap:
  targets:
    - ai-plan/public/configuration-governance/README.md
    - ai-plan/public/configuration-governance/startup-prompt.md
    - ai-plan/public/configuration-governance/todos/configuration-governance-tracking.md
    - ai-plan/public/configuration-governance/traces/configuration-governance-trace.md
    - ai-plan/design/governance/platform/配置治理与迁移规范.md
    - ai-plan/design/decisions/ADR-020-configuration-governance-schema.md
    - ai-plan/roadmap/配置治理实施路线.md
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current batch: initial implementation and CI regression coverage complete.
- The embedded Schema remains the repository-wide configuration-contract authority.
- Next step: release review, then evolve Schema snapshots through the documented deprecation and removal lifecycle.

## Task Checklist

- [x] Publish configuration-governance design, ADR, and roadmap.
- [x] Implement embedded schema, source-aware resolver, structural Compose CLI validation, and runtime CLI preflight.
- [x] Implement Compose contract checker and production dependency gate.
- [x] Add CI validation, patch-only migration suggestions, and regression coverage.

## Acceptance Conditions

- Invalid configuration prevents `serve` and `migrate up` before database or runtime initialization.
- Production Compose prevents bootstrap and server execution when contract validation fails.
- CI validates the official example environment and Compose template from the same schema authority.
- Operators receive source-aware, secret-safe migration guidance and deterministic nonzero exits.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "schema-authority-and-runtime-preflight",
    "compose-contract-and-deployment-gate",
    "ci-and-operator-regression-coverage"
  ],
  "pending_batches": [],
  "current_batch": "none",
  "next_batch": "release-integration-review",
  "closeout_status": "implementation-complete"
}
```
