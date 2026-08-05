# Build History Tracking

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Build owns history projections and artifact evidence; Task, Container, and Project retain their established authority boundaries.
requires:
  design: false
  topic: true
  roadmap: false
  adr: false
execution:
  engine: graft-multi-agent-loop
  dispatch_skill: graft-multi-agent-task
bootstrap:
  targets:
    - ai-plan/public/archive/build-history
closeout:
  archive: true
  lessons_review: true
```

## Archive Record

- The discovery and filtering/pagination batches completed, the controller accepted archive readiness, and the topic moved to `ai-plan/public/archive/build-history/`. No active recovery or further batch is pending.

## Task Checklist

- [x] Establish the first justified Build history batch.
- [x] Add Build-owned history filtering and pagination.
- [x] Complete the controller-owned archive-readiness check.

## Latest Batch Evidence

- Build list queries now support exact `application_id`, `image_repository`, and `image_tag` filters plus inclusive RFC 3339 `created_after` and `created_before` bounds.
- The repository computes total and items from the same Build-owned filter set and keeps `created_at DESC, id DESC` ordering stable across pagination windows.
- The Jobs page submits the generated OpenAPI query shape and retains filter, page, and page-size state locally.
- Validation passed: `cd server && go test ./modules/build/...`, `cd server && go run ./cmd/graft validate backend`, `cd web && bun run check`, `just openapi-check`, and `git diff --check`.

## Acceptance Conditions

- Build remains the authority for job and artifact history.
- No duplicate Task execution/log/realtime authority is introduced.
- Any persistence, OpenAPI, and web changes pass the applicable completion validation.

## Archive Readiness

- `ARCHIVE_READY`: all acceptance conditions pass. Build retains history authority, no Task/Container/Project authority is duplicated, and the committed cross-boundary implementation passed its required validation.
- Experience capture: none. The completed work did not produce a new reusable lesson beyond the existing authority-first and topic-completion-loop governance.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-2-history-discovery", "phase-2-history-filtering-and-pagination"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "archived",
  "stop_reason": "All Build History acceptance conditions passed and the topic was archived.",
  "recovery": {"status": "none", "resume_target": null, "repair_authority": null, "repair_eligible": false}
}
```
