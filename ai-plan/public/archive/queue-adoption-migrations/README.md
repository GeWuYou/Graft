# Queue Adoption Migrations

## Current Status Summary

- Topic objective: migrate selected high-value request-blocking operations to the Task Runtime.
- Current status: `archive-ready`
- Task class: `cross-boundary`
- Intake summary: long-running refactor using the existing Task Runtime and queue-adoption design.
- Canonical authority: Task Runtime, Container module, and `openapi/**` shared wire contract.
- Completed so far: Docker image pull, single-container lifecycle actions, all Container batch lifecycle actions, and the Backup Task executor authority repair.
- Completed: the Backup public API, Platform navigation, and web history page.

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
2. Compare every batch with `origin/main`: reassess and split at 80 changed files, and never exceed 90 changed files.

## Current Recovery Point

- Batch 5 repairs the Backup executor authority: `backups.task_id` binds each recorded Backup to its Task, the record Stage only verifies frozen artifacts, and Task-backed records are idempotent.
- Batch 6 completed the bounded Backup public surface: the canonical API returns Task receipts, safe history remains artifact-free, and the Platform page observes the shared Task drawer. User-requested Backup Tasks accept `1d`, `7d`, or `30d` retention, defaulting to `30d`; Update pre-backup remains an independent fixed 30-day policy.

## Work Intake

- This topic was created through Work Intake.
- The full Work Contract is persisted in `todos/queue-adoption-migrations-tracking.md`.

## Topic State

- The current planned migration batches are complete. The topic is `archive-ready` after its validated scoped commit.

## Loop Entry

- Preferred entry: `ai-plan/public/queue-adoption-migrations/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
