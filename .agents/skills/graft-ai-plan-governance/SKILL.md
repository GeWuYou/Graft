---
name: graft-ai-plan-governance
description: "Maintain Graft ai-plan routing, recovery materials, templates, catalog coverage, and their bounded validators."
---

# Graft AI Plan Governance

Use this skill when a `docs/automation` task changes the governance shape of `ai-plan/**`.

Treat root [AGENTS.md](../../../AGENTS.md) and [ai-plan/AGENTS.md](../../../ai-plan/AGENTS.md) as the governance sources. This skill does not replace startup,
validation, commit, closeout, or runtime authority.

## Read by Changed Authority

Reuse this task’s root receipt, environment inventory, and [ai-plan local rules](../../../ai-plan/AGENTS.md). Read missing evidence only:

- Directory/router/catalog changes: the affected section of [ai-plan README](../../../ai-plan/README.md) and the actual index entries or validator contract.
- Topic recovery or lifecycle changes: [tracking/recovery design](../../../ai-plan/design/governance/ai/AI任务追踪与恢复设计.md) plus that topic’s entry, current tracking point, and relevant trace. Use [the active router](../../../ai-plan/public/README.md) only to discover an unknown mapping or edit the router.
- Work Intake or bootstrap changes: [ADR-003](../../../ai-plan/design/decisions/ADR-003-work-intake-and-bootstrap-model.md) and the current contract; templates need only the affected template contract, not unrelated topic histories.
- Skill, tool, or helper governance: [AI tooling governance](../../../ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md); use `graft-ai-governance-audit` for its additional relevant detail.

A router/template/catalog edit without a topic does not require reading active-topic files.

## When To Use

Use this skill when the task changes any of:

- [ai-plan/AGENTS.md](../../../ai-plan/AGENTS.md)
- [ai-plan/README.md](../../../ai-plan/README.md)
- `ai-plan/catalog.json`
- `ai-plan/design/**` documents that define `ai-plan` governance or ADR convergence
- [ai-plan/public/README.md](../../../ai-plan/public/README.md)
- `ai-plan/public/<topic>/**` recovery materials
- `ai-plan/templates/**`
- `Work Intake`, `Work Contract`, or contract-driven bootstrap governance under `ai-plan/**`
- bounded `docs/automation` validators under `scripts/**` that guard `ai-plan` governance structure
- repo-local skills under `.agents/skills/**` only when they are part of the same `ai-plan` governance slice

## When Not To Use

Do not use this skill for:

- `server`, `web`, OpenAPI, or runtime implementation work
- whole-repo `ai-plan/**` retrofit, frontmatter rollout, or projection-model expansion
- tasks whose real authority lives outside `ai-plan/**`
- broader AI tooling or MCP adoption work that does not materially change `ai-plan` governance; use
  `graft-ai-governance-audit` for that

## Workflow

### 1. Authority And Router

- keep root [AGENTS.md](../../../AGENTS.md) as the only startup-governance source
- choose the narrowest `ai-plan` owner: `design/`, `design/decisions/`, `roadmap/`, `public/`, `public/archive/`,
  `lessons/`, or `templates/`
- treat `ai-plan/catalog.json` as supplementary, single-file, and bounded
- treat `Work Intake` as the only approved entry workflow for new long-running work
- treat `design`, `roadmap`, `topic`, and `ADR` as artifacts selected by `Work Contract`, not peer intake paths
- escalate outside `ai-plan/**` only when authority discovery proves it is required

### 2. Topic Safety

- preserve concurrent active topics in [ai-plan/public/README.md](../../../ai-plan/public/README.md)
- do not break another topic's startup entry while updating shared router docs
- keep `startup-prompt.md` aligned with root startup governance; do not invent a second receipt format
- when substantive work changes an active topic, update its tracking and trace files in the same change
- when a topic comes from `Work Intake`, keep the persisted `Work Contract` in tracking rather than inventing a new
  topic-level metadata file

### 3. Skill And Script Coupling

- add or modify repo-local skills only when the batch explicitly requires them
- keep new skills narrow and tied to existing authority docs
- if a validator or skill is added, sync only the minimum docs that must route future work to it
- keep validators as `docs/automation` guards, not runtime or CI substitutes

## Validation

For `ai-plan/**` governance work, always run:

```bash
git diff --check
```

Add:

```bash
python3 scripts/validate_ai_plan_structure.py
```

when the slice touches `ai-plan/catalog.json`, [ai-plan/public/README.md](../../../ai-plan/public/README.md), `ai-plan/templates/**`, active-topic
recovery structure, or the `ai-plan` structure guard itself.

Add:

```bash
python3 scripts/validate_ai_governance.py
```

when the slice touches `.agents/skills/**`, `scripts/**`, or
[ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md](../../../ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md).

Run:

```bash
python3 -m unittest scripts/test_validate_ai_plan_structure.py
```

only when `scripts/validate_ai_plan_structure.py` or its tests change.

## Guardrails

- do not create a second startup path, second validation truth, or hidden recovery store
- do not let `graft-work-intake` own independent intake rules outside root [AGENTS.md](../../../AGENTS.md), [ai-plan/AGENTS.md](../../../ai-plan/AGENTS.md), and
  the approved ADRs
- do not start `compose-project-management` implementation from `ai-plan` governance work
- do not mass-retrofit existing topics to match template wording
- do not widen bounded validators into a whole-repo `ai-plan` linter
- do not treat owned scope as permission to ignore upstream authority drift

## Closeout

Report:

```text
ai_plan_governance:
- authority_owner:
- router_paths_changed:
- concurrent_topics_preserved:
- validators_run:
- scope_expanded:
```
