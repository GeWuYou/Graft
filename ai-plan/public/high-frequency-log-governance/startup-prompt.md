Continue the `high-frequency-log-governance` topic in the same `topic-completion-loop` unless the caller explicitly changes loop mode.

Round context:

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/high-frequency-log-governance/README.md`
- local execution truth:
  - `server/AGENTS.md`
  - `web/AGENTS.md` when OpenAPI or web consumers change
  - `ai-plan/AGENTS.md` when topic or governance documents change
- design authority:
  - `server/internal/logger/**`
  - `server/internal/config/**`
  - `ai-plan/design/domains/audit/日志治理开发规范.md`
- AI skills:
  - `$graft-multi-agent-loop`
  - `$graft-multi-agent-task`
  - `$graft-validation-runner`
  - `$graft-comment-governance`

Topic objective:

- Implement `TRACE + typed Category + Registry + thin CategoryLogger + lazy fields` for high-frequency Graft runtime logs without replacing existing `*zap.Logger` injection.

Locked decisions:

1. Categories are typed constants registered by the logger owner; business code must not pass category string literals.
2. `GRAFT_LOG_CATEGORIES` is the only category configuration surface.
3. TRACE diagnostics are process-output-only; durable App Log category support is a later bounded batch.

Current batch plan:

1. Complete the logger Category and TRACE foundation with normative docs and tests.
2. Migrate high-frequency call sites and add static governance coverage.
3. Evaluate and implement durable App Log/OpenAPI/web category propagation.

Loop instructions:

- Default `loop_mode=topic-completion-loop`.
- Advance exactly one bounded batch this round.
- Update tracking and trace files in the same change.
- Run the smallest required validation before closeout.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.
