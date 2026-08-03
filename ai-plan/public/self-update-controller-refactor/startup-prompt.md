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

1. A named runner-owned state volume stores the sole active operation state as versioned atomic snapshot plus sparse
   append events. Schema v2 uses `lease_epoch`, a 30-second heartbeat and five-minute expiry; server reads it only
   after integrity, binding and lease validation.
2. The runner never receives PostgreSQL credentials or exposes a public API; server projects verified terminal results
   into business history and relays sanitized snapshots through the existing realtime boundary.
3. `runner_lost` is lease-derived rather than Docker-derived. A non-terminal pre-migration lost operation requires an
   authorized manual recovery runner; server does not fabricate a phase, including when first state is missing.

Implementation guardrails:

- Repair this controller ownership boundary before adding any lower-layer compatibility mapping.
- Preserve frozen Deployment Runtime input, official Compose-root trust boundaries, manifest verification, and
  forward-only migration rules.
- Do not retain server-owned running lifecycle fields, Task receipt settlement, runner log receipts as status truth,
  old lifecycle API aliases, or direct browser-to-runner transport.

Current batch plan:

1. Completed: runner state-volume schema, atomic persistence, operation mutual exclusion, phase controller, and
   recovery path.
2. Completed: server request admission, read-only projection, terminal history migration, API/realtime contract
   convergence, Compose state-volume wiring, and Update Center recovery rendering.
3. Current: deliver durable lease/fencing (`lease_epoch`, 30-second heartbeat, five-minute expiry, `runner_lost`, and
   recovery fencing), then rerun cross-boundary validation and perform the archive-readiness review. Do not restart
   the completed implementation batches.

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
