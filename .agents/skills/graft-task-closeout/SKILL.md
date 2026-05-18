---
name: graft-task-closeout
description: Repository-specific end-of-task closeout workflow for Graft. Use when a task slice is ending and the agent needs to decide whether to emit only a next-step startup prompt or to commit the current validated slice first and then hand off safely. This is the default slice-end path for work started through `graft-boot`.
---

# Graft Task Closeout

Use this skill when the current `Graft` slice is ending and the agent needs a concise, repeatable closeout workflow.
When a slice started through `graft-boot`, treat this skill as the normal required wrap-up path rather than an
optional extra.

Treat root `AGENTS.md` as the closeout-governance source of truth. This skill coordinates existing repository skills; it
does not replace startup, validation, or commit rules.

## When To Use

Typical triggers:

* `close this slice`
* `wrap up and hand off`
* `generate the next-step startup prompt`
* `commit current validated slice if needed and hand off`

Prefer this skill over `graft-commit` when the main question is task closeout rather than commit creation alone.

## Preconditions

1. Ensure the current turn already has the startup receipt required by `AGENTS.md` `4.1 Startup Governance`.
2. Confirm the task is actually at a closeout point:
   - the current slice is complete enough to hand off, or
   - the current slice is blocked and needs an honest next-step prompt
3. If validation status is unclear, use `graft-validation-runner` before deciding whether a commit is allowed.

## Workflow

1. Read the closeout rules in `AGENTS.md`:
   - `4.1 Startup Governance` for required handoff prompt fields
   - `13. Git Workflow Rules` for pre-handoff commit requirements
2. Classify the closeout state:
   - `validated and owned`: the current slice reached the required validation level and ownership is clear
   - `handoff_only`: the slice needs a next-step prompt but is not ready for a safe commit
   - `blocked`: ownership, validation, or scope is too ambiguous to claim closure
3. Decide whether a commit is required:
   - if the task ends with a real next-task handoff and the current slice reached the required validation level, prefer
     committing the confirmed owned scope before handoff
   - if validation is still insufficient, do not force a commit; record the exact gap
   - if ownership is mixed or ambiguous, do not force a commit; keep the scope uncommitted
4. Always evaluate commit eligibility using `graft-commit` rules, even when the answer may be “do not commit yet”.
5. If a commit is justified or explicitly requested, delegate commit execution to `graft-commit`:
   - keep the scope limited to the confirmed owned slice
   - reuse the same ownership and validation rules instead of inventing a second commit path
6. Emit one explicit next-task startup prompt whenever the output hands work to a future turn. The prompt must include:
   - `governance source`
   - `task class`
   - `recovery source`
   - `owned scope`
   - if recovery context matters, the parent topic and subtopic to read after startup preflight
7. Report the closeout result concisely:
   - whether a commit was created, and if so the scoped title and short SHA
   - whether the output is a handoff prompt only
   - what validation was used or what exact validation gap remains
8. When the caller asks for machine-readable closeout, end the result with one fenced ` ```json ` block that matches the
   closeout state and the current delegated-round budget.

## Output Contract

The closeout result should stay concise and should contain:

1. `closeout status`: `completed_no_handoff`, `committed_and_handed_off`, `handoff_only`, or `blocked`
2. `validation`: exact command run or the exact limitation
3. `next-step startup prompt`: only when a future turn is expected
4. when machine-readable closeout is requested, one fenced JSON block with:
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

Use plain language, but keep the startup prompt explicit enough that the next turn can rerun startup preflight without
guessing the inherited context.

### JSON Closeout Contract

When a caller such as `graft-multi-agent-loop` requests machine-readable closeout, use these rules:

* keep the human-readable closeout first
* if a future turn is required, include exactly one line beginning with `Next-session startup prompt:`
* append exactly one fenced ` ```json ` block as the last structured artifact
* `continue=true` requires a non-empty `next_prompt`
* `continue=false` requires `next_prompt=null`
* `validation.status` should be one of `passed`, `failed`, or `not_run`
* `risk_level` should be one of `low`, `medium`, or `high`
* `consumed_budget` must describe only the current slice or delegated round
* `remaining_budget` must reflect the budget left after the current slice or delegated round

Recommended JSON shape:

```json
{
  "closeout_status": "completed_no_handoff | committed_and_handed_off | handoff_only | blocked",
  "continue": true,
  "next_prompt": "string or null",
  "stop_reason": "string or null",
  "validation": {
    "status": "passed | failed | not_run",
    "commands": ["..."],
    "note": "string or null"
  },
  "commit": {
    "created": true,
    "sha": "string or null",
    "title": "string or null"
  },
  "consumed_budget": {
    "rounds": 1,
    "files_changed": 3,
    "commits": 0,
    "runtime_minutes": 10
  },
  "remaining_budget": {
    "rounds": 4,
    "files_changed": 27,
    "commits": 2,
    "runtime_minutes": 80
  },
  "scope_expanded": false,
  "risk_level": "low"
}
```

## Boundaries

* do not treat this skill as permission to commit unrelated changes
* do not duplicate `graft-boot`; this skill assumes startup already happened in the current turn
* do not duplicate `graft-validation-runner`; use it when validation scope is uncertain
* do not duplicate `graft-commit`; use its rules for every commit-eligibility decision and use it directly when the
  decision is to create a commit
* do not claim the next turn can skip startup preflight

## Example Startup Prompt Template

`You are handling a <task class> task. Governance source: root AGENTS.md. Task class: <task class>. Recovery source: <none|parent topic|parent topic + subtopic>. Owned scope: <owned scope>. First rerun startup preflight from AGENTS.md 4.1, then continue with <next step>.`
