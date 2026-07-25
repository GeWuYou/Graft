# User/Auth Transaction Ownership

## Current Status Summary

- Topic objective: make user/auth writes use explicit, provable transaction ownership without compensation-based atomicity.
- Current status: `active`
- Task class: `server`
- Intake summary: long-running refactor executed through the topic-completion loop.
- Canonical authority:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
- Completed so far: transaction audit, module-local ownership boundaries, the narrow cross-module transaction contract, the user-owned composite transaction adapter, and the Batch 5 commit/rollback proof suite.
- Current focus: archive readiness for the completed topic.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `none`
- authority summary: user owns profile lifecycle; auth owns credential and session state; `internal/moduleapi` owns the narrow cross-module capability.

## Owned Scope

- `server/modules/user/**`
- `server/modules/auth/**`
- `server/internal/moduleapi/**`
- `ai-plan/public/user-auth-transaction-ownership/**`

Out of scope:

- schema, migration, OpenAPI, web, and unrelated module changes.
- a global UnitOfWork or generic transaction framework.

## Locked Decisions

1. Handlers never begin, commit, or roll back transactions; repositories do not hide transaction lifecycle creation.
2. Cross-module user/auth writes use one explicitly owned SQL transaction only after both modules expose transaction-scoped stores.
3. No post-commit Delete/Undo compensation may claim atomicity.

## Current Recovery Point

- Batch 1 established auth-native transaction ownership and rollback proof.
- Batch 2 established user profile ownership without compensation.
- Batch 3 froze the transaction-scoped cross-module contract.
- Batch 4 bound auth to the user-owned composite transaction and removed compensation-based atomicity claims.
- Batch 5 completed focused cross-module commit and rollback proof.
- Next step: perform the archive-readiness check, then move the completed topic to the historical router when its acceptance evidence remains current.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in the tracking file.

## Pending Batch Direction

- No implementation batches remain. The pending direction is archive readiness.

## Loop Entry

- Preferred entry: `ai-plan/public/user-auth-transaction-ownership/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
