---
name: graft-multi-agent-loop
description: Repository-specific loop orchestrator for Graft multi-agent tasks. Use when one bounded task should run through repeated same-session serial worker-subagent rounds of `graft-multi-agent-task` until closeout stops emitting a next-session startup prompt or an execution budget stops the loop.
---

# Graft Multi-Agent Loop

Use this skill when a `Graft` task should run as a sequence of bounded delegated rounds under one main-agent session,
with the main agent acting as the loop orchestrator and each implementation round delegated to one worker subagent by
default.

Treat root `AGENTS.md` as the only governance source. This skill is only an outer automation wrapper around
`graft-multi-agent-task`; it does not define a second startup path, a second validation contract, or a second commit
workflow.

## When To Use

Use this skill when all of the following are true:

* the task should be executed through `graft-multi-agent-task`
* the task may require multiple future-session handoffs before it is actually complete
* you want the main agent to keep coordinating serial delegated rounds until closeout says to stop or a budget is exhausted

Typical triggers:

* `run this as a looped multi-agent task`
* `continue this multi-agent task automatically until it finishes`
* `use graft-multi-agent-loop for this bounded slice`

## Workflow

1. Ensure the current turn already has the startup receipt required by root `AGENTS.md`.
2. Confirm the owned scope and explicit budget before starting the loop:
   - `max_rounds`
   - `max_files_changed`
   - `max_commits`
   - `max_runtime_minutes`
   - `allowed_scopes`
   - validation failure policy
   - `checkpoint_budget` with default `1`
   - checkpoint cooldown
   - `soft_timeout_minutes`
     - default to `30` for deep implementation rounds unless the caller explicitly sets a smaller bound
   - `short_grace_window`
   - `default_grace_window`
     - default to `20` for deep implementation rounds unless the caller explicitly sets a smaller bound
   - `max_grace_window`
     - default to `30` for deep implementation rounds unless the caller explicitly sets a larger bound
3. Keep orchestration in the main agent and delegate each bounded implementation round to exactly one `worker`
   subagent by default:
   - build one round prompt that restates the inherited startup context, owned scope, remaining budget, allowed scopes,
     validation expectations, health-check rules, and required closeout format
   - require the worker round to run the slice through `$graft-multi-agent-task`
   - use an `explorer` subagent instead of a `worker` only when the round is genuinely read-only
   - allow `graft-multi-agent-batch` only inside the delegated round when that round itself benefits from parallel
     subagent work; inside loop rounds, default sidecars to read-only `explorer` subagents unless a bounded write slice
     is clearly justified
4. During an active round, keep the outer main agent limited to orchestration work:
   - inspect repository state or returned artifacts as needed for acceptance
   - wait for the worker result
   - parse the closeout JSON and track remaining budget
   - decide whether to accept, retry, continue, or stop
   - do not edit repo-tracked implementation files for the active round
   - treat the round as a simple state machine: `running -> checkpoint_requested -> checkpoint_received ->
     waiting_for_final_closeout -> completed | retry_pending | blocked`
5. Treat `timeout != stalled`:
   - exceeding one wait window or one soft timeout is not enough on its own to declare the worker stalled
   - absence of visible `git diff` or repo-tracked file changes is not, by itself, evidence of no progress; design, read-only dependency mapping, validation setup, or edit preparation may still be active
   - before any checkpoint request, first distinguish:
     - `no visible diff yet`
     - `no new visible output evidence`
     - `closeout not started`
   - when the current tool surface does not expose a direct activity query, do not rewrite "cannot observe tool activity" into "no tool activity"
   - if the worker still shows recent visible output or other signs that an edit wave is about to start, keep waiting instead of interrupting
   - stalled judgment requires all of the following:
     - the round has exceeded soft timeout
     - there has been prolonged lack of new visible output evidence
     - the worker has not reached closeout
     - a checkpoint request still fails to return a usable health response
6. Use bounded checkpoint requests instead of ad-hoc remote control:
   - every round starts with `checkpoint_budget=1` unless the round budget explicitly raises it to `2` or `3`
   - checkpoint requests use `interrupt=true`
   - checkpoint requests are health checks only and must not change the task goal, broaden scope, or append new
     implementation requirements
   - checkpoint responses are not closeouts and must not be interpreted as implicit stop signals
   - do not send a checkpoint just because one or more `wait_agent` windows elapsed without a closeout
   - the default trigger for a first checkpoint is: the round is at or beyond `soft_timeout`, has no usable closeout yet, and the main agent has reason to believe both output and tool activity have gone quiet for a prolonged period
   - when the only signal is “still no diff”, prefer waiting; use checkpoint only after the stronger stalled signals above are also present
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
7. After a usable checkpoint, set the next wait window from ETA without breaking the total round budget:
   - `eta_confidence=high`: wait `estimated_remaining_minutes`, capped by `max_grace_window`
   - `eta_confidence=medium`: wait `min(estimated_remaining_minutes, default_grace_window)`
   - `eta_confidence=low`: wait only `short_grace_window`, then checkpoint again or move to retry/block
   - ETA is advisory only; it must not justify exceeding the round's remaining runtime budget
   - if the checkpoint reports the worker is in an active pre-write or early-write phase and `can_continue=true`, treat that as positive health evidence; prefer another wait window over retry escalation
   - after any usable checkpoint with `can_continue=true`, explicitly continue the same worker round and resume waiting
     for that worker's final closeout; if the worker was closed by the interrupt handling path, reopen it first, then
     send a resume message that preserves the same goal, scope, and budget before the next wait window
   - do not close, replace, or mark the round malformed merely because the most recent message was a checkpoint
   - before classifying a round as missing closeout, perform at least one post-checkpoint `wait_agent` window sized from ETA or the default grace rule above
   - if a worker later emits a valid final closeout after a prior checkpoint, accept that final closeout as the round result rather than freezing the earlier checkpoint as terminal state
   - incomplete checkpoint content alone is not retry justification; first use the post-checkpoint grace rule unless the worker explicitly cannot continue
8. Let the main agent decide whether to continue based on:
   - closeout JSON
   - the presence or absence of `Next-session startup prompt:`
   - repeated prompts
   - scope expansion
   - risk level
   - remaining budget
9. If a delegated worker round stalls, omits closeout, or returns contradictory closeout:
   - degrade worker reliability when ETA repeatedly misses, there is no substantive progress, or no closeout arrives
   - do not classify a round as stalled while the latest evidence still shows recent visible output or a credible near-term next action
   - if the worker gives no response, a malformed final closeout after the post-checkpoint grace handling above, `can_continue=false`, or exhausts checkpoint budget,
     enter `retry_once_then_blocked`
   - retry the same bounded round once with a fresh worker subagent
   - the retry worker must inherit the partial diff, relevant logs, validation evidence, and the previous worker
     failure reason
   - if the second worker still fails to emit a usable closeout, stop the loop as `blocked`
   - do not recover the implementation locally and do not silently continue outside the loop contract
   - keep the stop reason explicit in the final closeout
   - if the first worker already produced substantive owned-scope changes, preserve that fact in the retry context and do not describe the round as diff-free unless Git still confirms there are no relevant changes
10. Stop when:
   - no further next-session startup prompt is emitted
   - the closeout JSON says `continue: false`
   - a budget limit is exhausted
   - validation fails under a stop-on-failure policy
   - a worker closeout fails twice under the retry-once policy
   - the delegated round expands scope or reports high risk

## Output Contract

Every delegated round run through this loop must end with:

1. a concise human-readable closeout
2. `Next-session startup prompt: <prompt>` when a future round is required
3. a fenced JSON block containing:
   - `closeout_status`
   - `continue`
   - `next_prompt`
   - `stop_reason`
   - `validation`
   - `commit`
   - `consumed_budget`
   - `remaining_budget`
   - `scope_expanded`
   - `risk_level`

The main agent treats the JSON block as the primary control surface. The keyword line is a human-readable mirror, not a
replacement control plane.

Checkpoint responses are not a second closeout format. They are bounded health reports used only to decide the next
wait window or whether to enter `retry_once_then_blocked`.

## Boundaries

* do not use this skill as a substitute for `graft-boot`
* do not bypass `graft-multi-agent-task`; this skill only orchestrates repeated delegated rounds of it
* do not let the loop broaden ownership beyond the declared `allowed_scopes`
* do not treat the loop as permission to skip closeout, validation, or scoped commit rules
* do not let checkpoint interrupts turn the loop into real-time remote control of the worker
* do not let a stalled or malformed delegated round silently downgrade into untracked main-agent execution
* do not assume a delegated round can inherit unstated governance; the round prompt must restate the inherited context
* do not reintroduce `run_loop.py`, `test_run_loop.py`, or `codex exec --ephemeral` style external fresh-session
  runners as part of this skill
