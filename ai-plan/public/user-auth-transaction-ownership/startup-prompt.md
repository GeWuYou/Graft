Continue the user/auth transaction ownership topic in `topic-completion-loop` mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `server`
- recovery source: `parent topic`
- recovery entry: `ai-plan/public/user-auth-transaction-ownership/README.md`
- local execution truth: `server/AGENTS.md`
- design authority:
  - `ai-plan/design/architecture/模块与依赖注入设计.md`
  - `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`
  - `$graft-multi-agent-batch`

Topic objective:

- Replace user/auth compensation-based and hidden transaction behavior with explicit, module-owned transaction lifecycles and a narrow composite adapter.

Locked decisions:

1. No handler-owned transaction lifecycle and no repository-hidden Begin/Commit/Rollback.
2. No generic UnitOfWork and no post-commit compensation as an atomicity substitute.
3. The user lifecycle workflow owns composite user/auth transaction commit; auth owns its tx-bound writer adapter.

Current batch plan:

1. Execute the batch recorded as `current_batch` in tracking.
2. Update tracking and trace, validate, then use `$graft-commit` for the confirmed owned scope.

Validation expectations:

```bash
cd server && go test ./modules/auth/...
cd server && go test ./modules/user/...
cd server && go run ./cmd/graft validate backend --stage lint
```

Use stronger direct validation when the active batch changes transaction semantics.
