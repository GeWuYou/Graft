# <Topic Title> Trace

Copy to `ai-plan/public/<topic>/traces/<topic>-trace.md` and replace every placeholder before use.

## <YYYY-MM-DD> <batch or milestone>

- <decision, implementation step, or validation milestone>
- <decision, implementation step, or validation milestone>

## Locked Decisions

- <decision>
- <decision>

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
