# Destructive Operation Contract Convergence Tracking

## Topic

Destructive Operation Contract Convergence

## Scope

Converge destructive API semantics, OpenAPI metadata, batch results, module behavior, and generated consumers in bounded authority-first batches.

## Repository Truth

- `AGENTS.md`
- `ai-plan/AGENTS.md`
- `ai-plan/design/governance/backend/服务端API边界与兼容治理规范.md`
- `ai-plan/design/governance/backend/后端安全与信任边界治理规范.md`
- `ai-plan/design/governance/backend/数据库表设计与迁移规范.md`
- `ai-plan/design/governance/backend/队列采用与任务异步化治理规范.md`
- `openapi/openapi.yaml`
- `server/AGENTS.md`
- `web/AGENTS.md`

## Work Contract

```yaml
version: 1
kind: refactor
scope: long-running
authority_summary: Backend API governance and canonical OpenAPI own destructive wire semantics; modules own domain effects and Task Runtime owns external execution.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: direct-specialized-skill
  dispatch_skill: graft-openapi-contract-review
bootstrap:
  targets:
    - topic
    - design
    - roadmap
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Startup, authority discovery, and idle worktree acquisition are complete.
- Contract-foundation, the user soft-delete pilot, and relationship/RBAC convergence are complete and fully validated.
- Existing operations remain unannotated until their behavior is migrated; full inventory coverage becomes blocking in the convergence-closeout batch.
- RBAC relationship removals use canonical DELETE operations without aliases; atomic security batches return the shared ordered result contract and reject empty, duplicate, or over-100-item requests.
- The next owned batch is external Docker/Compose destruction through canonical Task receipts.

## Task Checklist

- [x] `contract-foundation`
- [x] `soft-delete-pilot`
- [x] `relationship-and-rbac`
- [x] `hard-delete-commands`
- [ ] `external-destruction-tasks`
- [ ] `convergence-closeout`

## Acceptance Conditions

- Soft-deleted resources return 204 on an authorized repeat DELETE while GET/list hide tombstones.
- Irreversible database deletion uses POST deletion commands with persistent idempotency receipts.
- External destruction returns 202 with the canonical Task receipt and has queryable terminal state.
- Ordinary batch deletion has bounded partial results; security-sensitive batches commit atomically.
- All migrated operations carry validated `x-graft-destructive` metadata and use the shared result contracts.
- Legacy action paths and module-specific destructive batch envelopes are deleted without aliases.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "contract-foundation",
    "soft-delete-pilot",
    "relationship-and-rbac",
    "hard-delete-commands"
  ],
  "pending_batches": [
    "external-destruction-tasks",
    "convergence-closeout"
  ],
  "current_batch": "external-destruction-tasks",
  "next_batch": "convergence-closeout",
  "closeout_status": "batch-validated"
}
```
