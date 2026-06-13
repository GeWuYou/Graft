---
name: graft-multi-agent-loop
description: Repository-specific loop orchestrator for Graft multi-agent tasks. Use when one bounded task should run through repeated same-session serial worker-subagent rounds of `graft-multi-agent-task` under the default `topic-completion-loop` mode until the topic reaches `archive-ready`, becomes `blocked`, or no remaining batches stay in scope.
---

# Graft Multi-Agent Loop

Use this skill when a `Graft` task should run as a sequence of bounded delegated rounds under one main-agent session,
with the main agent acting as the long-lived loop owner and each implementation round delegated to one worker subagent
by default.

Treat root `AGENTS.md` as the only governance source. This skill is only an outer automation wrapper around
`graft-multi-agent-task`; it does not define a second startup path, a second validation contract, or a second commit
workflow.

## Loop Modes

`graft-multi-agent-loop` supports two loop modes:

* `topic-completion-loop`
  - default mode
  - use unless the caller explicitly sets `loop_mode=checkpoint-loop`
  - keeps the outer main agent responsible for batch-state maintenance, next-batch dispatch, topic recovery updates,
    scoped commit flow, and final `archive-ready` or `blocked` judgment
* `checkpoint-loop`
  - non-default compatibility mode
  - use only when the caller explicitly requests a checkpoint-driven governance task
  - must not be selected just because the user omitted `loop_mode`

If `loop_mode` is omitted, the loop must run as `topic-completion-loop`.

## When To Use

Use this skill when all of the following are true:

* the task should be executed through `graft-multi-agent-task`
* the task is best advanced as multiple bounded batches under one main-agent session
* you want the main agent to keep coordinating serial delegated rounds until the topic reaches an explicit terminal
  state or no safe in-scope batch remains

Typical triggers:

* `run this as a looped multi-agent task`
* `continue this multi-agent task automatically until it finishes`
* `use graft-multi-agent-loop for this bounded slice`

## Workflow

1. Ensure the current turn already has the startup receipt required by root `AGENTS.md`.
2. Confirm the loop mode before the first round:
   - if the caller omitted `loop_mode`, set `loop_mode=topic-completion-loop`
   - only use `checkpoint-loop` when the caller explicitly requested it
3. Confirm the owned scope, reference metrics, and any user-defined hard limits before starting the loop:
   - reference metrics are health signals used for checkpoints and acceptance review, not stop conditions by default
   - hard limits are explicit stop boundaries from the user, inherited prompt, or this skill's defaults
   - examples of hard limits: `max_rounds=3`, `max_commits=1`, `allowed_scopes=server/modules/scheduler`
   - examples of reference metrics: files changed, runtime, validation failures, soft timeout, and grace windows
   - `max_rounds`
   - `max_files_changed`
   - `max_commits`
   - `max_runtime_minutes`
   - `allowed_scopes`
   - validation failure policy
     - validation commands remain behavioral constraints for the delegated worker; ordinary fixable lint, type, style,
       or test failures normally stay with that same worker for diagnosis, in-scope repair, rerun validation, and then
       closeout or scoped commit
   - `checkpoint_budget` with default `1`
   - checkpoint cooldown
   - `soft_timeout_minutes`
     - default to `30` for deep implementation rounds unless the caller explicitly sets a smaller bound
   - `short_grace_window`
   - `default_grace_window`
     - default to `20` for deep implementation rounds unless the caller explicitly sets a smaller bound
   - `max_grace_window`
     - default to `30` for deep implementation rounds unless the caller explicitly sets a larger bound
   - treat `checkpoint_budget` as a hard limit by default; treat timeouts and grace windows as health metrics unless
     the caller explicitly defines them as hard limits
4. Establish the loop batch state in the outer main agent before dispatching Batch 1:
   - `completed_batches`
   - `pending_batches`
   - `current_batch`
   - `next_batch`
   - in `topic-completion-loop`, this state is mandatory and must be updated after every accepted closeout
5. Keep orchestration in the main agent and delegate each bounded implementation round to exactly one `worker`
   subagent by default:
   - build one round prompt that restates the inherited startup context, loop mode, owned scope, remaining budget,
     batch-state expectations, allowed scopes, validation expectations, health-check rules, and required closeout
     format
   - require the worker round to run the slice through `$graft-multi-agent-task`
   - use an `explorer` subagent instead of a `worker` only when the round is genuinely read-only
   - allow `graft-multi-agent-batch` only inside the delegated round when that round itself benefits from parallel
     subagent work; inside loop rounds, default sidecars to read-only `explorer` subagents unless a bounded write
     slice is clearly justified
6. During an active round, keep the outer main agent limited to orchestration work:
   - inspect repository state or returned artifacts as needed for acceptance
   - wait for the worker result
   - parse the closeout JSON and track remaining budget
   - decide whether to accept, retry, continue, or stop
   - do not edit repo-tracked implementation files for the active round
   - treat the round as a simple state machine: `running -> checkpoint_requested -> checkpoint_received ->
     waiting_for_final_closeout -> completed | retry_pending | blocked`
7. In `topic-completion-loop`, batch success must continue by default:
   - after an accepted worker closeout, the outer main agent must:
     - verify owned scope stayed bounded
     - verify validation and commit results for the current batch
     - update `completed_batches`
     - update `pending_batches`
     - update topic recovery materials such as trace and todos when the loop owns them
     - automatically choose `next_batch`
     - when `pending_batches` is not empty, dispatch the next worker unless a terminal stop condition applies
     - when `pending_batches` becomes empty, do not stop immediately; run one final archive-readiness check first
   - the final archive-readiness check must verify the topic-level acceptance conditions before the loop may stop
   - after the final archive-readiness check:
     - if all acceptance conditions pass, mark the loop `archive-ready` and commit any owned archive or closeout docs
     - if acceptance conditions fail but more bounded work is clear, generate new `pending_batches`, choose
       `next_batch`, and continue
     - if acceptance conditions fail and no safe next batch can be defined without user help, stop as `blocked`
   - do not end the loop after ordinary batch success
   - do not emit a `Next-session startup prompt:` for ordinary batch success
8. Treat `timeout != stalled`:
   - exceeding one wait window or one soft timeout is not enough on its own to declare the worker stalled
   - absence of visible `git diff` or repo-tracked file changes is not, by itself, evidence of no progress; design,
     read-only dependency mapping, validation setup, or edit preparation may still be active
   - before any checkpoint request, first distinguish:
     - `no visible diff yet`
     - `no new visible output evidence`
     - `closeout not started`
   - when the current tool surface does not expose a direct activity query, do not rewrite "cannot observe tool
     activity" into "no tool activity"
   - if the worker still shows recent visible output or other signs that an edit wave is about to start, keep waiting
     instead of interrupting
   - stalled judgment requires all of the following:
     - the round has exceeded soft timeout
     - there has been prolonged lack of new visible output evidence
     - the worker has not reached closeout
     - a checkpoint request still fails to return a usable health response
9. Use bounded checkpoint requests instead of ad-hoc remote control:
   - every round starts with `checkpoint_budget=1` unless the round budget explicitly raises it to `2` or `3`
   - checkpoint requests use `interrupt=true`
   - in `topic-completion-loop`, checkpoint is exceptional only for:
     - `blocked`
     - architecture decision required
     - unsafe worktree
     - validation failed after reasonable worker self-repair attempts, or repair is unsafe/out of scope
     - retry exhausted
     - explicit user intervention required
   - checkpoint requests are health checks only and must not change the task goal, broaden scope, or append new
     implementation requirements
   - checkpoint responses are not closeouts and must not be interpreted as implicit stop signals
   - do not send a checkpoint just because one or more `wait_agent` windows elapsed without a closeout
   - the default trigger for a first checkpoint is: the round is at or beyond `soft_timeout`, has no usable closeout
     yet, and the main agent has reason to believe both output and tool activity have gone quiet for a prolonged period
   - when the only signal is “still no diff”, prefer waiting; use checkpoint only after the stronger stalled signals
     above are also present
   - enforce checkpoint cooldown; do not send frequent back-to-back interrupts
   - the worker must respond with a structured status containing:
     - `current_phase`
     - `changed_files`
     - `last_validation`
     - `next_action`
     - `can_continue`
     - `estimated_remaining_minutes`
     - `eta_confidence`
     - `risks_or_blockers`
   - a checkpoint response must begin with `Checkpoint status:`, must not include `Next-session startup prompt:`, and
     must not append the final closeout JSON block
10. After a usable checkpoint, set the next wait window from ETA while respecting any user-defined hard limit:
   - `eta_confidence=high`: wait `estimated_remaining_minutes`, capped by `max_grace_window`
   - `eta_confidence=medium`: wait `min(estimated_remaining_minutes, default_grace_window)`
   - `eta_confidence=low`: wait only `short_grace_window`, then checkpoint again or move to retry/block
   - ETA is advisory only; it must not override an explicit hard runtime limit
   - reference budget overruns alone are not blocking grounds; stopping requires an explicit hard limit or a
     substantive validation, safety, scope, closeout, retry, risk, or user-stop reason
   - if the checkpoint reports the worker is in an active pre-write or early-write phase and `can_continue=true`,
     treat that as positive health evidence; prefer another wait window over retry escalation
   - after any usable checkpoint with `can_continue=true`, explicitly continue the same worker round and resume waiting
     for that worker's final closeout; if the worker was closed by the interrupt handling path, reopen it first, then
     send a resume message that preserves the same goal, scope, budget, and current batch before the next wait window
   - do not close, replace, or mark the round malformed merely because the most recent message was a checkpoint
   - before classifying a round as missing closeout, perform at least one post-checkpoint `wait_agent` window sized
     from ETA or the default grace rule above
   - if a worker later emits a valid final closeout after a prior checkpoint, accept that final closeout as the round
     result rather than freezing the earlier checkpoint as terminal state
   - incomplete checkpoint content alone is not retry justification; first use the post-checkpoint grace rule unless
     the worker explicitly cannot continue
11. Let the main agent decide whether to continue based on:
   - closeout JSON
   - loop mode
   - batch state
   - scope expansion
   - risk level
   - remaining reference metrics and any explicit hard limits
   - explicit terminal conditions
12. If a delegated worker round stalls, omits closeout, or returns contradictory closeout:
   - degrade worker reliability when ETA repeatedly misses, there is no substantive progress, or no closeout arrives
   - do not classify a round as stalled while the latest evidence still shows recent visible output or a credible
     near-term next action
   - if the worker gives no response, a malformed final closeout after the post-checkpoint grace handling above,
     `can_continue=false`, or exhausts checkpoint budget, enter `retry_once_then_blocked`
   - retry the same bounded round once with a fresh worker subagent
   - the retry worker must inherit the partial diff, relevant logs, validation evidence, and the previous worker
     failure reason
   - if the second worker still fails to emit a usable closeout, stop the loop as `blocked`
   - do not recover the implementation locally and do not silently continue outside the loop contract
   - keep the stop reason explicit in the final closeout
   - if the first worker already produced substantive owned-scope changes, preserve that fact in the retry context and
     do not describe the round as diff-free unless Git still confirms there are no relevant changes
13. Stop when:
   - the topic reaches `archive-ready`
   - the loop becomes `blocked`
   - a user-defined hard limit is exhausted
   - the worker reports a non-recoverable validation failure after reasonable in-scope self-repair attempts, or explains
     that repair is unsafe or out of scope
   - a worker closeout fails twice under the retry-once policy
   - the delegated round expands scope or reports high risk
   - the worktree becomes unsafe for scoped worker continuation
   - the user explicitly stops the loop
   - a reference metric overrun combines with a substantive safety, validation, scope, closeout, retry, risk, or
     user-stop reason
14. Use `Next-session startup prompt:` only for terminal handoff states:
   - `blocked`
   - `archive-ready`
   - `explicit stop`
   - do not use it as the normal continuation mechanism for `topic-completion-loop`

## Output Contract

Every delegated round run through this loop must end with:

1. a concise human-readable closeout
2. `Next-session startup prompt: <prompt>` only when the round ends in a terminal handoff state that requires a future
   turn
3. a fenced JSON block containing:
   - `closeout_status`
   - `continue`
   - `loop_mode`
   - `current_batch`
   - `completed_batches`
   - `pending_batches`
   - `next_batch`
   - `next_batch_prompt`
   - `next_prompt`
   - `stop_reason`
   - `validation`
   - `commit`
   - `consumed_budget`
   - `remaining_budget`
   - `scope_expanded`
   - `risk_level`

The main agent treats the JSON block as the primary control surface. In `topic-completion-loop`, `continue=true`
means the outer main agent must keep the same-session loop alive. It must not downgrade that signal into a mere hint or
fall back to `checkpoint-loop` because `loop_mode` was omitted.

In `topic-completion-loop`:

* `continue=true` requires:
  - `loop_mode=topic-completion-loop`
  - non-empty `current_batch`
  - updated `completed_batches`
  - updated `pending_batches`
  - a non-empty `next_batch` and `next_batch_prompt` when batches remain
  - `next_prompt=null`
* `continue=false` requires one explicit terminal reason such as:
  - `archive-ready`
  - `blocked`
  - `validation failed`
  - `retry exhausted`
  - `owned scope conflict`
  - `explicit user stop`
* `pending_batches=[]` alone is not a stop condition:
  - the outer main agent must run the final archive-readiness check first
  - only that check may convert an empty pending set into `archive-ready`, regenerated `pending_batches`, or
    `blocked`

Checkpoint responses are not a second closeout format. They are bounded health reports used only to decide the next
wait window or whether to enter `retry_once_then_blocked`.

## Boundaries

* do not use this skill as a substitute for `graft-boot`
* do not bypass `graft-multi-agent-task`; this skill only orchestrates repeated delegated rounds of it
* do not let the loop broaden ownership beyond the declared `allowed_scopes`
* do not treat the loop as permission to skip closeout, validation, scoped commit rules, or batch-state updates
* do not let checkpoint interrupts turn the loop into real-time remote control of the worker
* do not let a stalled or malformed delegated round silently downgrade into untracked main-agent execution
* do not assume a delegated round can inherit unstated governance; the round prompt must restate the inherited context
* do not reintroduce `run_loop.py`, `test_run_loop.py`, or `codex exec --ephemeral` style external fresh-session
  runners as part of this skill
