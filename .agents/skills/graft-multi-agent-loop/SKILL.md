---
name: graft-multi-agent-loop
description: Repository-specific loop orchestrator for Graft multi-agent tasks. Use when one bounded task should run through repeated fresh sessions of `graft-multi-agent-task` until closeout stops emitting a next-session startup prompt or an execution budget stops the loop.
---

# Graft Multi-Agent Loop

Use this skill when a `Graft` task should run as a sequence of fresh-session slices instead of one long conversation.

Treat root `AGENTS.md` as the only governance source. This skill is only an outer automation wrapper around
`graft-multi-agent-task`; it does not define a second startup path, a second validation contract, or a second commit
workflow.

## When To Use

Use this skill when all of the following are true:

* the task should be executed through `graft-multi-agent-task`
* the task may require multiple future-session handoffs before it is actually complete
* you want a local loop runner to keep launching fresh Codex sessions until closeout says to stop or a budget is exhausted

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
3. Use `scripts/run_loop.py` to launch repeated fresh sessions with `codex exec --ephemeral`.
4. For each round, tell the child session to:
   - run the task through `$graft-multi-agent-task`
   - keep closeout human-readable
   - emit the required machine-readable JSON closeout result
5. Let the loop runner decide whether to continue based on:
   - closeout JSON
   - the presence or absence of `Next-session startup prompt:`
   - repeated prompts
   - scope expansion
   - risk level
   - remaining budget
6. Stop when:
   - no further next-session startup prompt is emitted
   - the closeout JSON says `continue: false`
   - a budget limit is exhausted
   - validation fails under a stop-on-failure policy
   - the child session expands scope or reports high risk

## Output Contract

Every child session run through this loop must end with:

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

The loop runner treats the JSON block as the primary control surface and falls back to the keyword line only when JSON
is missing or malformed.

## Boundaries

* do not use this skill as a substitute for `graft-boot`
* do not bypass `graft-multi-agent-task`; this skill only orchestrates repeated fresh runs of it
* do not let the loop broaden ownership beyond the declared `allowed_scopes`
* do not treat the loop as permission to skip closeout, validation, or scoped commit rules
* do not assume a fresh session can inherit unstated governance; the round prompt must restate the inherited context
