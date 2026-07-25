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
- Completed so far: transaction audit, implementation design, and auth-native transaction boundary.
- Current focus: user-owned profile lifecycle and session revocation convergence.

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

## Phase Plan

- Establish auth and user module-local transaction boundaries.
- Freeze transaction ownership and add the minimum shared capability contract.
- Bind the auth adapter to the user-owned composite transaction.
- Prove commit and rollback behavior with focused integration tests.

## Current Recovery Point

- Batch 1 established auth-native transaction ownership and rollback proof.
- Adapter implementation remains blocked until user module boundaries and the shared contract are accepted.
- Next step: establish user ownership and session revocation convergence.

## Work Intake

- This topic was created through `Work Intake`.
- The full Work Contract is in the tracking file.

## Pending Batch Direction

- Batch 1: auth native transaction ownership.
- Batch 2: user ownership and session revocation convergence.
- Batch 3: narrow user/auth transaction contract.
- Batch 4: transaction adapter and composite atomic writes.
- Batch 5: rollback and commit proof suite.

## Loop Entry

- Preferred entry: `ai-plan/public/user-auth-transaction-ownership/startup-prompt.md`
- Preferred execution mode: `$graft-multi-agent-loop`
