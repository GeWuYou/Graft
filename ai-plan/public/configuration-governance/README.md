# Configuration Governance

## Current Status Summary

- Topic objective: establish versioned, source-aware governance for deployment environment configuration and the official Compose topology.
- Current status: `active`
- Task class: `docs/automation with server impact`
- Intake summary: long-running platform governance feature requiring an authoritative design, ADR, implementation roadmap, and bounded delivery batches.
- Canonical authority:
  - `server/internal/config/**`
  - `compose.yml` and `compose.env.example`
  - `ai-plan/design/governance/platform/部署配置与运行时策略治理规范.md`
- Completed so far: architecture analysis, Schema v1, runtime preflight, official Compose contract and production gate, CI validation, and regression coverage.
- Future evolution: expand environment-rule coverage as config owners are added, and add a new immutable Schema snapshot for each published contract version.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation with server impact`
- recovery source: `none`
- authority summary: the embedded configuration schema owns deployment configuration lifecycle; runtime, Compose, templates, and CI consume that schema.

## Owned Scope

- `server/internal/config/**`
- `server/internal/cli/**`
- `server/internal/app/**`
- `compose.yml`, `compose.env.example`, and Compose validation tests
- `.github/workflows/**` configuration-validation wiring
- `ai-plan/**` configuration-governance design, ADR, roadmap, and recovery materials

Out of scope:

- Writing or automatically mutating operator `.env` files.
- System Config runtime-policy behavior, database schema changes, and web feature work.

## Locked Decisions

1. Versioned embedded YAML schema is the canonical source for deployment configuration lifecycle and defaults.
2. Production Compose uses a one-shot preflight gate; runtime commands use the same resolver before database or service initialization.
3. Configuration migration is diagnostics and patch generation only in the first release.

## Phase Plan

- Define the schema authority, lifecycle rules, ADR, and implementation roadmap.
- Implement source-aware environment resolution and `graft config validate`.
- Add Compose contract validation and the production preflight service.
- Wire CI, regression coverage, and operator-facing migration guidance.

## Current Recovery Point

- Current batch: initial implementation is complete.
- Risk: the Compose contract intentionally covers only the official production topology; alternate deployment profiles require an explicit future Schema profile.
- Next step: review this completed slice for release integration, then advance Schema versions through the documented lifecycle.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is in `todos/configuration-governance-tracking.md`.

## Pending Batch Direction

- Add new environment and Compose rules only by first updating the embedded Schema and its regression tests.
- Keep migration output diagnostic-only until an independently approved operator-file migration workflow exists.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
cd server && go test ./internal/config ./internal/cli
```

## Loop Entry

- Preferred entry: `ai-plan/public/configuration-governance/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-batch`
