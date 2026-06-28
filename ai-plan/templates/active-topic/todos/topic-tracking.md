# <Topic Title> Tracking

Copy to `ai-plan/public/<topic>/todos/<topic>-tracking.md` and replace every placeholder before use. Delete unused
sections instead of carrying template text forward.

## Topic

<Topic Title>

## Scope

<describe the bounded topic scope>

## Repository Truth

- `AGENTS.md`
- `<authority path 1>`
- `<authority path 2>`

## Current Recovery Point

- <current state summary>
- <current risk, blocker, or escalation status>
- <current next step>

## Task Checklist

- [ ] `<batch or milestone 1>`
- [ ] `<batch or milestone 2>`

## Acceptance Conditions

- <acceptance condition 1>
- <acceptance condition 2>

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [],
  "pending_batches": [
    "<current batch>"
  ],
  "current_batch": "<current batch>",
  "next_batch": "<next batch or repeat current batch>",
  "closeout_status": "not-started"
}
```
