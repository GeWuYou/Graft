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
    - ai-plan/public/build-history
closeout:
  archive: true
  lessons_review: true
```

## Current Recovery Point

- Both the discovery and filtering/pagination batches are accepted. The outer controller must now perform the topic-level archive-readiness check before selecting any new work.

## Task Checklist

- [x] Establish the first justified Build history batch.
- [x] Add Build-owned history filtering and pagination.
- [ ] Complete the controller-owned archive-readiness check.

## Latest Batch Evidence

- Build list queries now support exact `application_id`, `image_repository`, and `image_tag` filters plus inclusive RFC 3339 `created_after` and `created_before` bounds.
- The repository computes total and items from the same Build-owned filter set and keeps `created_at DESC, id DESC` ordering stable across pagination windows.
- The Jobs page submits the generated OpenAPI query shape and retains filter, page, and page-size state locally.
- Validation passed: `cd server && go test ./modules/build/...`, `cd server && go run ./cmd/graft validate backend`, `cd web && bun run check`, `just openapi-check`, and `git diff --check`.

## Acceptance Conditions

- Build remains the authority for job and artifact history.
- No duplicate Task execution/log/realtime authority is introduced.
- Any persistence, OpenAPI, and web changes pass the applicable completion validation.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["phase-2-history-discovery", "phase-2-history-filtering-and-pagination"],
  "pending_batches": [],
  "current_batch": null,
  "next_batch": null,
  "closeout_status": "active",
  "stop_reason": null,
  "recovery": {"status": "none", "resume_target": null, "repair_authority": null, "repair_eligible": false}
}
```
