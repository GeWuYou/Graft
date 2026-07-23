# Migration Governance

## Current Status Summary

- Topic objective: establish the Phase 1 SQL Schema Governance and Migration Contract Gate.
- Current status: `active`
- Task class: `docs/automation with server impact`
- Intake summary: long-running governance feature with an active topic and multi-agent batch execution.
- Canonical authority:
  - `server/internal/moduleregistry`
  - `server/internal/cli/migrate.go`
  - `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
- Completed so far: architecture audit and implementation plan.
- Not started yet: Phase 1 implementation.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: `none`
- authority summary: the module registry defines the default migration chain; Atlas executes it and PostgreSQL catalog state proves its semantics.

## Owned Scope

- `server/internal/cli/**`
- `scripts/**`
- `.github/workflows/**`
- `Justfile`
- `ai-plan/public/migration-governance/**`
- `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`

Out of scope:

- Phase 2 naming/index debt remediation.
- Rewriting historical live migrations or changing module migration ownership.

## Locked Decisions

1. Schema semantic checks use PostgreSQL catalog queries, not SQL text parsing.
2. Phase 1 blocks semantic/bootstrap failures and reports naming/FK-index debt.
3. Destructive migrations remain a review requirement until a PostgreSQL-aware automated analyzer is available.

## Phase Plan

- Phase 1: validate the default chain, bootstrap PostgreSQL, and check catalog contracts.
- Phase 2: baseline and gradually enforce naming and FK-index conventions.
- Phase 3: full contract enforcement and schema drift fingerprinting.

## Current Recovery Point

- Worktree branch: `feature/migration-governance`.
- Current batch: Phase 1 implementation, split across CLI, catalog checker, and automation wiring.
- Next step: implement the default-chain export boundary before workflow integration.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in `todos/migration-governance-tracking.md`.

## Pending Batch Direction

- Add catalog contract enforcement and disposable bootstrap orchestration.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
python3 scripts/validate_ai_governance.py
```

## Loop Entry

- Preferred entry: `ai-plan/public/migration-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`
