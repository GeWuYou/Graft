# Graft Multi-Agent Loop

`graft-multi-agent-loop` is the repository loop orchestrator for repeated delegated rounds under one long-lived main
agent session.

## Default Mode

The default mode is `topic-completion-loop`.

- If `loop_mode` is omitted, the main agent must run `topic-completion-loop`.
- `checkpoint-loop` is available only when the caller explicitly requests it.
- The loop must not fall back to `checkpoint-loop` just because the user did not specify `loop_mode`.

## Main-Agent Responsibilities

In `topic-completion-loop`, the outer main agent owns:

- batch-state maintenance
- closeout acceptance
- next-batch selection
- next worker dispatch
- trace and todo updates
- scoped commit flow
- final `archive-ready` or `blocked` judgment

After every accepted worker closeout, the main agent must:

1. verify owned scope and validation
2. update `completed_batches`
3. update `pending_batches`
4. choose `next_batch`
5. if `pending_batches` is not empty, dispatch the next worker
6. if `pending_batches` is empty, run the final archive-readiness check instead of stopping immediately

## Final Archive-Readiness Check

When `pending_batches` becomes empty, the main agent must run a final archive-readiness check.

- If all acceptance conditions pass, mark the topic `archive-ready` and commit any owned archive or closeout docs.
- If acceptance conditions fail but a bounded next slice is clear, generate new `pending_batches` and continue.
- If acceptance conditions fail and no safe next batch can be defined without user help, enter `blocked`.

## Worker Responsibilities

The worker owns only one bounded batch:

- implement the batch within owned scope
- run the required validation for that batch
- report risks or blockers
- return a usable closeout JSON block

The worker does not own loop lifecycle, topic completion, or `archive-ready` decisions.

## Stop Rules

In `topic-completion-loop`, stop only when one of these conditions applies:

- `archive-ready`
- `blocked`
- validation-failure stop policy
- retry exhausted
- owned-scope conflict
- explicit user stop

Ordinary batch success must continue by default.
An empty `pending_batches` set is not, by itself, a terminal state.

## Handoff Rule

`Next-session startup prompt:` is a terminal handoff artifact only.

Use it only for:

- `blocked`
- `archive-ready`
- explicit user stop

Do not use it as the normal success path for another batch.
