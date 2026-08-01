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

## Controller Ownership And State Machine

The outer main agent is the only topic lifecycle owner, loop state owner, batch transition owner, and terminal decision
owner. Workers own implementation for one delegated round only. They may return evidence and an optional advisory
`suggested_follow_up`; they must not decide whether the topic continues, choose the next batch, update
`pending_batches`, or mark the topic `archive-ready`.

The controller must use this state machine:

```text
INIT
  -> DISPATCH
  -> WAIT
  -> VERIFY
  -> SETTLE
  -> ADVANCE
  -> DISPATCH_NEXT
  -> DISPATCH

SETTLE -> ARCHIVE_CHECK -> ARCHIVE_READY | BLOCKED

VERIFY -> RECOVERY_REQUIRED
  -> RECOVERY_CONTEXT_RESTORED
  -> RECOVERY_COMPLETE
  -> RESUME_CURRENT_BATCH
  -> DISPATCH
  -> WAIT
  -> VERIFY
```

Rules:

* a worker return enters `VERIFY` only; it never enters a terminal state directly
* an accepted batch enters `SETTLE`; the controller alone updates batch state there
* non-empty `pending_batches` requires `ADVANCE` followed by `DISPATCH_NEXT`
* `ARCHIVE_CHECK` is allowed only after the controller has settled an empty pending set
* `ARCHIVE_READY` and `BLOCKED` are the only terminal states; `END` is reachable only from one of them
* `RECOVERY_REQUIRED`, `RECOVERY_CONTEXT_RESTORED`, `RECOVERY_COMPLETE`, and `RESUME_CURRENT_BATCH` are non-terminal
  controller states; they must never transition to `END`, `ARCHIVE_READY`, or `BLOCKED` without an explicit terminal
  decision
* `RECOVERY_COMPLETE` requires restored controller state, confirmed repair authority, a safe bounded repair or retry,
  and `execute_repair` when root `AGENTS.md` requires repair authorization
* after `RECOVERY_COMPLETE`, the controller must enter `RESUME_CURRENT_BATCH` and dispatch a repaired worker for the
  preserved logical batch; it must not wait for a separate user request to restart execution

Termination invariant:

* worker completion never completes a topic loop
* batch completion never completes a topic loop
* commit success never completes a topic loop
* validation success never completes a topic loop
* recovery receipt creation, recovery-context restoration, authority confirmation, and recovery completion never
  complete a topic loop
* only a controller terminal state can complete a topic loop
* the main agent must not emit final completion while controller state is non-terminal, or before `pending_batches` is
  empty and the archive-readiness check has completed

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
     - validation commands remain behavioral constraints for the delegated worker. The worker may diagnose an ordinary
       lint, type, style, or test failure, but must return the root `AGENTS.md` `Repair Confirmation Interaction
       Contract` proposal before repair; only the native `execute_repair` choice permits the declared repair,
       validation rerun, or scoped commit
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
   - in `topic-completion-loop`, this state is mandatory, controller-owned, and must be updated after every accepted
     closeout during `SETTLE`
5. Keep orchestration in the main agent and delegate each bounded implementation round to exactly one `worker`
   subagent by default:
   - build one round prompt that restates the inherited startup context, loop mode, owned scope, remaining budget,
     batch-state expectations, allowed scopes, validation expectations, health-check rules, and required closeout
     format
   - require the worker round to run the slice through `$graft-multi-agent-task`
   - require each implementation Phase or batch to run `$graft-commit` after successful validation and before the next
     Phase or batch starts, unless validation, ownership, mixed-worktree, or scoped-staging rules block the commit
   - use an `explorer` subagent instead of a `worker` only when the round is genuinely read-only
   - allow `graft-multi-agent-batch` only inside the delegated round when that round itself benefits from parallel
     subagent work; inside loop rounds, default sidecars to read-only `explorer` subagents unless a bounded write
     slice is clearly justified
   - before every round and sidecar dispatch, verify that the worker model is the same level as or lower than the
     immediate delegating agent's model; record `parent_model`, `worker_model`, `model_relation`, `reasoning_effort`,
     and comparison evidence in the round state
   - if the worker model is higher or its level cannot be verified, pause and request explicit user approval or
     direction; do not dispatch or silently choose a model based on availability
6. During an active round, keep the outer main agent limited to orchestration work:
   - inspect repository state or returned artifacts as needed for acceptance
   - wait for the worker result
   - parse worker evidence, verify it, and track remaining budget
   - make the controller transition from `WAIT` to `VERIFY`; decide whether to accept, retry, or block only after
     verification
   - do not edit repo-tracked implementation files for the active round
   - treat worker health as a nested state machine: `running -> checkpoint_requested -> checkpoint_received ->
     waiting_for_final_closeout -> completed | retry_pending | blocked`; this nested state never replaces controller
     state
   - when validation failure, scope conflict, or another recoverable blocker ends a worker round, enter
     `RECOVERY_REQUIRED` rather than settling the batch or ending the topic
   - preserve `current_batch` and `pending_batches`; do not add the failed batch to `completed_batches`, choose a
     `next_batch`, or enter `ADVANCE` while recovery is active
   - persist a recovery receipt with the failed-round evidence, required context, repair authority, repair-eligibility
     result, and the required `RESUME_CURRENT_BATCH` target
   - after a new session completes startup preflight, restore that receipt as `RECOVERY_CONTEXT_RESTORED`; once the
     recovery conditions are satisfied, transition through `RECOVERY_COMPLETE` and automatically resume the current
     batch
7. In `topic-completion-loop`, batch success must continue by default:
   - after an accepted worker closeout, the outer main agent must:
     - verify owned scope stayed bounded
     - verify validation and commit results for the current batch
     - refuse to dispatch the next implementation batch when a successful validated batch has uncommitted owned
       changes, unless the worker reported a concrete validation or ownership blocker under `$graft-commit`
     - enter `SETTLE` and update `completed_batches`
     - update `pending_batches`
     - update topic recovery materials such as trace and todos when the loop owns them
     - enter `ADVANCE` and automatically choose `next_batch`
     - when `pending_batches` is not empty, enter `DISPATCH_NEXT` and dispatch the next worker unless a terminal stop
       condition applies
     - when `pending_batches` becomes empty, do not stop immediately; enter `ARCHIVE_CHECK` first
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
     - `no final closeout yet`
     - `no new visible output evidence`
     - `closeout not started`
   - when the current tool surface does not expose a direct activity query, do not rewrite "cannot observe tool
     activity" into "no tool activity"
   - if the worker still shows recent visible output or other signs that an edit wave is about to start, keep waiting
     instead of interrupting
   - one wait timeout, one soft-timeout hit, or the combination of `no visible diff yet` plus `no final closeout yet`
     is not enough to close, replace, or locally take over the worker
   - stalled judgment requires all of the following:
     - the round has exceeded soft timeout
     - there has been prolonged lack of new visible output evidence
     - the worker has not reached closeout
     - a checkpoint request still fails to return a usable health response
9. Use bounded checkpoint requests instead of ad-hoc remote control:
   - every round starts with `checkpoint_budget=1` unless the round budget explicitly raises it to `2` or `3`
   - checkpoint requests use `interrupt=true`
   - checkpoint is a health check only; it is not a closeout, not a stop signal, and not permission for the outer main
     agent to finish the worker's implementation locally
   - in `topic-completion-loop`, checkpoint is exceptional only for:
     - `blocked`
     - architecture decision required
     - unsafe worktree
     - validation failed and the required repair proposal is awaiting a native user choice, `continue_current_scope` or
       `cancel_workflow` rejected repair, or repair is unsafe/out of scope
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
   - when no stronger explicit task-specific wait rule is present, use this minimum ladder:
     - first active wait window: `15` minutes
     - first timeout plus healthy checkpoint with `can_continue=true`: second wait window of at least `30` minutes
     - later healthy checkpoints: wait by credible ETA when available, otherwise keep doubling the prior window within
       applicable hard limits
   - classify the post-checkpoint state before any closure or retry decision:
     - `silent_timeout`: no usable final closeout arrived after the required post-checkpoint wait, there is no recent
       meaningful progress evidence, and the worker no longer shows a credible continuation signal
     - `active_but_unfinished`: the latest checkpoint or post-checkpoint evidence shows recent meaningful progress and
       `can_continue=true`, but the final closeout is still pending
     - `blocked`: the worker explicitly reports a blocker, unsafe continuation, out-of-scope repair, or
       `can_continue=false`
   - recent meaningful progress includes explicit diagnosis, owned-scope file edits, validation output, a relevant
     `git diff` change, or a concrete next step paired with visible tool activity
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
   - only `silent_timeout` may trigger immediate post-checkpoint closure after that required wait window; if the latest
     evidence is `active_but_unfinished`, continue the same worker with a bounded continuation window or refreshed
     grace instead of escalating directly to retry or blocked
   - do not close immediately after recent owned-scope file edits, validation work, or a new relevant `git diff`;
     refresh grace once within the current hard limits and reassess only after the worker goes silent again
   - if a worker later emits a valid final closeout after a prior checkpoint, accept that final closeout as the round
     result rather than freezing the earlier checkpoint as terminal state
   - incomplete checkpoint content alone is not retry justification; first use the post-checkpoint grace rule unless
     the worker explicitly cannot continue
11. Let the main agent decide whether to continue based on controller-owned state and verified worker evidence:
   - `round_status`, implementation, commit, validation, risk, and blocker evidence
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
   - `retry_once_then_blocked` is allowed only after one of these explicit post-checkpoint outcomes:
     - `silent_timeout`
     - a malformed final closeout after the post-checkpoint grace handling above and without recent meaningful progress
     - `blocked`
     - checkpoint budget exhausted without a usable health response
   - `active_but_unfinished` is not a retry trigger; keep the same worker alive with one bounded continuation or
     refreshed grace window, then reassess against hard limits and the latest evidence
   - retry the same bounded round once with a fresh worker subagent
   - re-run the model-level delegation check for the retry worker; retrying must not escalate above the current
     worker's model without explicit user approval
   - the retry worker must inherit the partial diff, relevant logs, validation evidence, and the previous worker
     failure reason
   - if the second worker still fails to emit a usable closeout, return retry-exhaustion as batch-wave evidence with
     the failed round's recovery inputs; this does not stop or block the topic by itself
   - the outer controller alone decides whether retry exhaustion becomes `RECOVERY_REQUIRED`, `BLOCKED`, or another
     legal controller transition after verifying the evidence and preserving the current batch
   - do not recover the implementation locally and do not silently continue outside the loop contract
   - keep the stop reason explicit in the final closeout
   - if the first worker already produced substantive owned-scope changes, preserve that fact in the retry context and
     do not describe the round as diff-free unless Git still confirms there are no relevant changes
13. Stop when:
   - the topic reaches `archive-ready`
   - the loop becomes `blocked`
   - a user-defined hard limit is exhausted
   - the worker reports a non-recoverable validation failure, the user cancels the required repair, or the repair is
     unsafe or out of scope and no safe bounded recovery can be defined
   - the outer controller explicitly resolves retry-exhaustion evidence into a terminal state after the retry-once
     policy is exhausted; retry exhaustion alone is not a loop stop
   - the delegated round expands scope or reports high risk
   - the worktree becomes unsafe for scoped worker continuation
   - the user explicitly stops the loop
   - a reference metric overrun combines with a substantive safety, validation, scope, closeout, retry, risk, or
     user-stop reason
14. Use `Next-session startup prompt:` for either a terminal handoff or a recoverable handoff:
   - `blocked`
   - `archive-ready`
   - `explicit stop`
   - `RECOVERY_REQUIRED` when the receipt identifies the preserved `current_batch`, the recovery context, and the
     `RESUME_CURRENT_BATCH` target
   - a recoverable handoff is not a stop signal and must not be represented as `blocked`; after preflight and recovery
     completion, the controller resumes without user re-dispatch
   - do not use it as the normal continuation mechanism for ordinary `topic-completion-loop` batch success

## Output Contract

### Worker Round Closeout

Every delegated worker round must end with a concise human-readable closeout and a fenced JSON block containing only
round evidence:

- `round_status`
- `implementation_result`
- `changed_scope`
- `commit`
- `validation_evidence`
- `risks`
- `blockers`
- `required_context` when recovery or retry needs prior evidence
- optional `recovery_requirements`, containing evidence only and never a controller transition
- `parent_model`, `worker_model`, `model_relation`, `model_rank_verified`, and `higher_model_approval`
- optional `suggested_follow_up`

`suggested_follow_up` may contain candidate work or recovery context, but is advisory only. A worker must not emit or
infer `continue`, `pending_batches`, `next_batch`, `archive_ready`, `topic_complete`, `stop_loop`, `suspend_topic`,
or `wait_for_user`. It must not emit a `Next-session startup prompt:`; the outer controller alone may emit terminal
or recoverable handoff output.

### Controller Decision Record

The outer main agent treats worker evidence as input, not as a control decision. After `VERIFY`, it alone records the
controller transition and owns `controller_state`, `current_batch`, `completed_batches`, `pending_batches`,
`next_batch`, budget, recovery receipt, and any terminal reason. It may dispatch the next worker only from
`DISPATCH_NEXT`, may dispatch the repaired current batch only from `RESUME_CURRENT_BATCH`, and may emit final
completion only from `ARCHIVE_READY` or `BLOCKED`.

The canonical outer-controller closeout schema is the only machine-readable controller decision record. It must contain
controller_state, current_batch, completed_batches, pending_batches, next_batch, stop_reason, terminal_reason, and
recovery. controller_state is authoritative; terminal_reason is required only when the controller resolves to
ARCHIVE_READY or BLOCKED. recovery must contain status, resume_target, current_batch_preserved,
pending_batches_preserved, failed_batch_settled, retry_exhausted, repair_authority, repair_eligible, and
required_context. Retry-exhausted wave output supplies evidence to recovery.required_context; it never supplies
controller_state or a terminal decision. Only the outer controller may emit this record or resolve that evidence to
RECOVERY_REQUIRED, BLOCKED, or another legal transition.

The recovery receipt is context for the controller, not a batch success record. During recovery, `current_batch` and
`pending_batches` remain unchanged, the failed batch remains unsettled, and `ADVANCE` is forbidden. Batch, worker,
and receipt output cannot suspend or terminate the topic; only the controller may resolve the receipt into
`RECOVERY_COMPLETE`, `RESUME_CURRENT_BATCH`, or a terminal state.

`pending_batches=[]` alone is not a stop condition. The controller must first run `ARCHIVE_CHECK`, which may produce
`ARCHIVE_READY`, regenerated `pending_batches`, or `BLOCKED`.

### Canonical Outer-Controller Closeout Schema

The fenced JSON emitted for an outer-controller closeout is the canonical schema for this workflow. Worker and batch
closeouts are evidence inputs and must not add controller state fields to it. Required top-level fields are:

```json
{
  "closeout_status": "completed_no_handoff | committed_and_handed_off | handoff_only | recovery_handoff | blocked | cancelled | unsafe",
  "continue": true,
  "loop_mode": "topic-completion-loop | checkpoint-loop | null",
  "current_batch": "string or null",
  "completed_batches": ["string"],
  "pending_batches": ["string"],
  "next_batch": "string or null",
  "next_batch_prompt": "string or null",
  "next_prompt": "string or null",
  "stop_reason": "string or null",
  "validation": {"status": "passed | failed | not_run", "commands": ["string"], "note": "string or null"},
  "commit": {"created": true, "sha": "string or null", "title": "string or null"},
  "consumed_budget": {"rounds": 1, "files_changed": 0, "commits": 0, "runtime_minutes": 0},
  "remaining_budget": {"rounds": 0, "files_changed": 0, "commits": 0, "runtime_minutes": 0},
  "scope_expanded": false,
  "risk_level": "low | medium | high",
  "recovery": {
    "status": "none | required | context_restored | complete",
    "resume_target": "RESUME_CURRENT_BATCH | null",
    "current_batch_preserved": true,
    "pending_batches_preserved": true,
    "failed_batch_settled": false,
    "retry_exhausted": false,
    "repair_authority": "string or null",
    "repair_eligible": false,
    "required_context": {"failed_round": "object", "evidence": "object"}
  }
}
```

`retry_exhausted` is wave evidence and recovery input only. It never authorizes `BLOCKED`, `ARCHIVE_READY`, or any
other terminal decision. When `recovery.status` is `required`, `required_context` must preserve the failed round,
validation evidence, repair authority, repair eligibility, and the `RESUME_CURRENT_BATCH` target. The outer controller
must settle that evidence before emitting a terminal closeout.

Checkpoint responses are not a second closeout format. They are bounded health reports used only to decide the next
wait window or whether to return retry-exhaustion evidence for an outer-controller recovery or terminal decision; their
`can_continue` field describes worker health, never topic lifecycle.

## Orchestration Examples

Valid flow:

```text
Main Agent: INIT, then dispatch phase-1-generated-registration
Worker: implement changes, validate, commit, return round evidence
Main Agent: VERIFY evidence, SETTLE the accepted batch, update controller state
Main Agent: pending batches remain, so ADVANCE and DISPATCH_NEXT
Main Agent: repeat until ARCHIVE_CHECK reaches ARCHIVE_READY or BLOCKED
```

Invalid flow:

```text
Main Agent: dispatch worker
Worker: success
Main Agent: summarize
END
```

This is invalid because worker success is not a controller terminal state.

## Manual Contract Regression Check

After changing this skill or the related batch/task worker contract, run:

```bash
python3 scripts/validate_loop_controller_contract.py
```

This is a targeted manual regression check. It is not part of `validate_ai_governance.py`, CI, hooks, or default test
discovery.

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
