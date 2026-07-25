Run the archive-readiness check for the user/auth transaction ownership topic.

Round context:

- governance source: root `AGENTS.md`
- task class: `docs/automation`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/user-auth-transaction-ownership/README.md`
- local execution truth: `ai-plan/AGENTS.md`
- design authority:
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- archive authority:
  - `ai-plan/design/governance/ai/AI任务追踪与恢复设计.md`

Topic objective:

- Confirm the completed topic's recovery evidence is archive-ready, then archive it through the repository's active-topic router flow.

Locked decisions:

1. No handler-owned transaction lifecycle and no repository-hidden Begin/Commit/Rollback.
2. No generic UnitOfWork and no post-commit compensation as an atomicity substitute.
3. The user lifecycle workflow owns composite user/auth transaction commit; auth owns its tx-bound writer adapter.

Archive-readiness checks:

1. Confirm Batch 5 remains the completed final implementation batch and that no pending batch exists.
2. Recheck the topic's acceptance evidence, recovery materials, and archive routing requirements.
3. Move the topic to the historical router only after the archive-readiness check passes; otherwise record the bounded blocker and retain the active topic.

Validation expectations:

```bash
cd server && go test ./modules/auth/...
cd server && go test ./modules/user/...
cd server && go run ./cmd/graft validate backend --stage lint
cd server && go build ./cmd/graft
```

Run these in the listed order. The topic has no remaining transaction-semantics implementation batch.
