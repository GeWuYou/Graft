# Queue Adoption Migrations

## Current Status Summary

- Topic objective: migrate selected high-value request-blocking operations to the Task Runtime.
- Current status: `active`
- Task class: `cross-boundary`
- Intake summary: long-running refactor using the existing Task Runtime and queue-adoption design.
- Canonical authority: Task Runtime, Container module, and `openapi/**` shared wire contract.
- Completed so far: Docker image pull and single-container start/stop/restart now submit Tasks.
- Not started yet: Container batch actions, container removal, and Backup execution entry.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- authority summary: repair Task Runtime, Container, and OpenAPI at their canonical owners before adapting consumers.

## Owned Scope

- Task Runtime submission, execution, and recovery behavior.
- Container image pull contract and UI adoption.
- Active-topic recovery materials for queue migration batches.

Out of scope:

- Dual NDJSON and Task API compatibility modes.
- Re-migrating Update operations that already submit Tasks.

## Locked Decisions

1. Docker image pull accepts a Task submission receipt rather than holding an NDJSON response open.
2. Each batch must stay below the repository file-change cap; split before the branch reaches 80 changed files.

## Current Recovery Point

- Batch 2 migrates single-container start/stop/restart while retaining their action-specific authorization boundary.
- Next step: complete cross-boundary validation, then select the batch-action or removal candidate without changing its synchronous partial-result contract implicitly.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is persisted in `todos/queue-adoption-migrations-tracking.md`.

## Pending Batch Direction

- Container batch actions and removal.
- Backup execution after a product-facing entry is available.

## Loop Entry

- Preferred entry: `ai-plan/public/queue-adoption-migrations/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
