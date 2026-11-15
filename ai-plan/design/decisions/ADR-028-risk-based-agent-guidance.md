# ADR-028: Risk-Based Agent Guidance

- Status: accepted
- Date: 2026-09-13
- Scope: four AGENTS.md files, 48 repository skills, 5 plugin skills, coupled governance and validators
- Decision authority: user-approved implementation plan; root AGENTS.md remains the execution authority

## Context

The root instruction file exceeds the default 32 KiB project instruction budget. Repeated preflight, mandatory broad
reading, exact-hunk repair approvals and model-tier uncertainty add avoidable pauses. Valuable module, contract,
validation, worktree and licensing constraints must survive a smaller instruction surface.

## Decision

1. Keep root AGENTS.md as the sole startup, authorization, delegation and completion policy owner. Child AGENTS own
   domain execution. Root, server and web each have an 8 KiB UTF-8 budget; ai-plan has 4 KiB. These are project budgets,
   not vendor recommendations. Do not increase personal client limits to solve instruction growth.
2. Route detailed guidance by changed behavior and material risk. Reuse a trustworthy same-turn startup receipt,
   previously read context and still-valid check evidence. High-risk concerns remain mandatory; one review may cover
   several concerns. Small repairs do not require full design reading or a new topic.
3. Authorized implementation includes related files, regression tests, diagnosed validation repairs and revalidation.
   A file/hunk list is an ownership aid, not the authorization boundary. Confirm only unknown/mixed ownership, changed
   goals or newly required external side effects. Workers ask the coordinator to reassign cross-worker scope.
4. A child model may be the same as or below its immediate parent. Choose autonomously from trustworthy host/official
   capability evidence; when another model's relation is unknown, inherit the parent and continue. Never infer rank
   from names or reasoning effort. This applies to nested delegation, replacement and retry. Remove exactly-one-tier-
   lower comment generation. Host API spellings live in the skill reference, not universal policy.
5. Preserve all 53 skill names and invocation surfaces. Descriptions state purpose and precise trigger in one or two
   sentences. Do not enforce a minimum character count. Move conditional protocols/templates to focused references.
6. Read-only PR review may finish with open/reported findings carrying evidence, risk and next step. Remediation still
   uses fixed/delegated/blocked/stale/noise final disposition. The remote managed ledger retains its final enum/schema.
7. Check structure, size, references and real data contracts; do not require repeated policy sentences. Retain all loop
   state-machine/DAG/controller protections and existing hooks, CI and full domain completion entrypoints.

## Implementation Interfaces

- Root stable headings: Startup Governance; Authority and Risk Routing; Authorized Work; Validation and Completion;
  Git and Worktree Safety; Delegation; License and Text Hygiene.
- AI code generation/review design owns detailed repair and review evidence; root owns permission policy.
- AI tooling/MCP design owns skill description/reference guidance and structural validation scope.
- Task tracking/recovery design owns loop and recovery detail; root owns inherited startup and model constraints.
- TDesign governance owns conditional API/DOM/version lookup and reuse evidence; web AGENTS owns frontend completion.
- Existing root/subdomain design documents retain domain detail through conditional links. No second startup chain,
   validator entrypoint, model ranking table, recovery store or roadmap is introduced.

## Consequences and Evaluation

Smaller default context requires precise routing and review of rule retention. Compare baseline and revised guidance
with isolated evaluators using the same model configuration and eight scenarios: small UI edit, local code repair,
cross-boundary API, read-only PR review, validation repair/new related file, mixed ownership/external action, model
selection, and loop continuation. Record qualitative results and loaded bytes; do not claim statistical performance.
