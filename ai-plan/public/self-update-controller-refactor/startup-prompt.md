# Self Update Controller Refactor Startup Prompt

Continue the same `topic-completion-loop` after rerunning root startup preflight.

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `none`
- recovery entry: `ai-plan/public/self-update-controller-refactor/README.md`
- local execution truth: `server/AGENTS.md` and `web/AGENTS.md` for corresponding implementation slices
- design authority: `ai-plan/design/decisions/ADR-009-self-update-controller-state-authority.md`,
  `ai-plan/design/release/platform-self-update.md`, and
  `ai-plan/design/decisions/ADR-008-deployment-runtime-context.md`
- AI skills: `$graft-multi-agent-loop`, `$graft-validation-runner`, `$graft-cache-governance`, and
  `$graft-localization-governance` when their trigger applies

Topic objective:

- make `graft-compose-runner` the independent self-update controller so service replacement cannot interrupt the
  authoritative upgrade state.

Work contract summary:

- long-running refactor with ADR, repository release design, roadmap, active topic, and cross-boundary implementation
  batches.

Locked decisions:

1. A named runner-owned state volume stores the sole active operation state as versioned atomic snapshot plus append
   events; server reads it only after integrity and binding validation.
2. The runner never receives PostgreSQL credentials or exposes a public API; server projects verified terminal results
   into business history and relays sanitized snapshots through the existing realtime boundary.
3. A non-terminal stale operation requires an authorized manual recovery runner; server does not fabricate a phase.

Implementation guardrails:

- Repair this controller ownership boundary before adding any lower-layer compatibility mapping.
- Preserve frozen Deployment Runtime input, official Compose-root trust boundaries, manifest verification, and
  forward-only migration rules.
- Do not retain server-owned running lifecycle fields, Task receipt settlement, runner log receipts as status truth,
  old lifecycle API aliases, or direct browser-to-runner transport.

Current batch plan:

1. Implement runner state-volume schema, atomic persistence, operation lease, phase controller, and recovery runner.
2. Replace server lifecycle ownership with request admission, read-only projection, terminal history migration, and
   API/realtime contract convergence.
3. Update Compose, Update Center recovery rendering, and cross-boundary tests; then complete archive-readiness review.

Loop instructions:

- Default `loop_mode=topic-completion-loop`; advance exactly one bounded batch per round.
- Update this topic's tracking and trace files in the same change as substantive work.
- Validate both server and web when shared contracts, menu/route, lifecycle, or realtime semantics change.
- Evaluate `$graft-commit` only after validation and only for confirmable owned scope.

Validation expectations:

```bash
git diff --check
```

Required closeout:

- State the current batch and authority owner changed this round.
- State validation performed and any inability to run the full server/web entrypoints.
- Update loop batch state and use `Next-session startup prompt:` only for terminal states.
