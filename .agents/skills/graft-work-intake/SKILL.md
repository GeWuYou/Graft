---
name: graft-work-intake
description: "Classify new long-running Graft work and create its minimal Work Contract and recovery structure."
---

# Graft Work Intake

Use this skill as the only workflow-level entry for new long-running work in `Graft`.

Treat root [AGENTS.md](../../../AGENTS.md), [ai-plan/AGENTS.md](../../../ai-plan/AGENTS.md), [ai-plan/README.md](../../../ai-plan/README.md), and the approved `ai-plan` ADRs as the authority
sources. This skill is a thin orchestration wrapper: it does not define independent business rules, startup rules, or
content-generation rules.

## Relevant Authority

Reuse the current startup receipt and [ai-plan local rules](../../../ai-plan/AGENTS.md). For classification and contract fields, use the relevant [ADR-003 Work Intake decision](../../../ai-plan/design/decisions/ADR-003-work-intake-and-bootstrap-model.md). Do not reread root or environment inventory solely because intake begins.

Consult [the active router](../../../ai-plan/public/README.md) only if no reliable ownership mapping is already known. When creating recovery materials, read the affected [tracking/recovery contract](../../../ai-plan/design/governance/ai/AI任务追踪与恢复设计.md); when choosing artifact locations, consult [ai-plan directory semantics](../../../ai-plan/README.md). Tooling governance is relevant only if intake itself changes skills, tools, or helper ownership.

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
