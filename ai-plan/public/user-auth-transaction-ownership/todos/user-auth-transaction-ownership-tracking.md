# User/Auth Transaction Ownership Tracking

## Topic

User/Auth Transaction Ownership

## Scope

Refactor user and auth persistence workflows so each business transaction has one explicit owner, then add the minimum adapter needed for atomic user profile, credential, and session lifecycle writes.

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/architecture/模块与依赖注入设计.md`
- `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: user owns profile lifecycle, auth owns credential and session state, and moduleapi owns the narrow stable cross-module capability
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - topic
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Current state: bootstrap complete; no implementation batch started.
- Risk: auth credential persistence performs independent user identity reads and cannot participate in an uncommitted user transaction without a tx-bound adapter.
- Next step: Batch 1, auth-native transaction ownership.

## Task Checklist

- [ ] Batch 1: move auth multi-write transaction lifecycle out of repositories and prove local rollback.
- [ ] Batch 2: establish user ownership rules and remove partial multi-session revocation behavior.
- [ ] Batch 3: add the minimum transaction-scoped cross-module capability contract.
- [ ] Batch 4: bind auth to the user-owned composite transaction and remove compensation.
- [ ] Batch 5: add cross-module atomicity and failure-injection tests; run final validation.

## Acceptance Conditions

- Every user/auth multi-write use case has one documented transaction owner.
- Composite create, disable, and delete operations either commit all user/auth facts or persist none.
- No adapter begins, commits, rolls back, or uses a different transaction from its owner.
- No compensation path remains as a claimed atomicity mechanism.
- Focused transaction tests cover success and each write-side failure boundary.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": [
    "auth-native-transaction-ownership",
    "user-ownership-and-session-revocation",
    "user-auth-transaction-contract",
    "user-auth-transaction-adapter",
    "transaction-consistency-proof"
  ],
  "current_batch": "auth-native-transaction-ownership",
  "next_batch": "user-ownership-and-session-revocation",
  "closeout_status": "not-started"
}
```
