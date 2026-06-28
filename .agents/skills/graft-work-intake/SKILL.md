---
name: graft-work-intake
description: Repository-specific workflow orchestrator for new long-running work. Use when startup discovers a request that may need design, topic, roadmap, ADR, bootstrap, or loop dispatch, and the repository must classify it through one Work Intake path instead of special-case entry flows.
---

# Graft Work Intake

Use this skill as the only workflow-level entry for new long-running work in `Graft`.

Treat root `AGENTS.md`, `ai-plan/AGENTS.md`, `ai-plan/README.md`, and the approved `ai-plan` ADRs as the authority
sources. This skill is a thin orchestration wrapper: it does not define independent business rules, startup rules, or
content-generation rules.

Guardrail summary:

- do not define independent business rules

## Read First

1. Complete the root `AGENTS.md` startup preflight.
2. Read `ai-plan/AGENTS.md`.
3. Read `ai-plan/README.md`.
4. Read `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`.
5. Read `ai-plan/design/governance/ai/AI工具与MCP接入治理规范.md`.
6. Read `ai-plan/design/decisions/ADR-003-work-intake-and-bootstrap-model.md`.
7. Read `ai-plan/public/README.md` only after startup preflight and only to confirm whether an active topic already
   owns the work.

## When To Use

Use this skill when all of the following are true:

- the request introduces new work rather than resuming an already-owned active topic
- the work may be long-running, multi-batch, or recovery-sensitive
- the repository needs to decide whether it requires `design`, `topic`, `roadmap`, `ADR`, bootstrap, or loop dispatch

Typical triggers:

- `design a new capability`
- `add a multi-batch feature`
- `plan a refactor that may need topic tracking`
- `create the long-running work entry for this request`

## Workflow

1. Confirm whether the request is `short-lived` or `long-running`.
2. If the work is `short-lived`, do not create a new topic or persisted `Work Contract`; hand off directly to normal
   implementation flow.
3. If the work is `long-running`, produce one `Work Contract`.
4. Decide the contract fields from documented rules only:
   - `kind`
   - `scope`
   - `authority_summary`
   - `requires.design`
   - `requires.topic`
   - `requires.roadmap`
   - `requires.adr`
   - `execution.engine`
   - `execution.dispatch_skill`
   - `bootstrap.targets`
   - `closeout.archive`
   - `closeout.lessons_review`
5. Apply contract-driven minimal bootstrap:
   - create only the minimal skeleton required by the contract
   - do not pre-create artifacts the contract did not request
   - do not fill in domain content beyond starter structure
6. Persist the `Work Contract` only when the contract creates an active topic:
   - store it in the topic tracking file
   - do not create a standalone `work-contract.yaml`
   - do not put the full contract in topic `README.md`
   - do not put the full contract in `ai-plan/catalog.json`
7. Dispatch the work to specialized skills after bootstrap:
   - design content goes to the relevant design or governance skill
   - long-running execution normally goes to `graft-multi-agent-loop`
   - closeout still goes through `graft-task-closeout`

## Guardrails

- do not create a second startup path, second validation truth, or second recovery store
- do not treat `design`, `roadmap`, `topic`, or `ADR` as peer entry workflows
- do not author domain content inside this skill
- do not persist a `Work Contract` for short-lived work
- do not let bootstrap guess future artifacts beyond what the contract explicitly requires

## Closeout

Report:

```text
work_intake:
- work_classification:
- work_contract_created:
- persisted_contract:
- bootstrap_targets:
- dispatch_skill:
- existing_topic_reused:
```
