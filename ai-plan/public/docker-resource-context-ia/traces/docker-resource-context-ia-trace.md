# Docker Resource Context IA Trace

## 2026-07-21 Intake and Batch 1

- Classified the task as long-running and `cross-boundary` because Volume needs a server-owned Context projection and both list filters are OpenAPI behavior.
- Locked the shared reading model: `Overview -> Context -> Relations -> Configuration -> Metadata -> Danger Zone`.
- Locked the rule that Context is business information, while metadata is diagnostic-only and cannot carry navigation or ownership inference.

## 2026-07-21 Contract and Web Implementation

- OpenAPI and the Container module now return a single aggregated Network or Volume detail projection containing Context, Relations, Configuration, and safe diagnostic metadata.
- Network list projection gained sanitized container references; the old inspect-shaped detail endpoint field and its unreferenced schema were removed.
- Web now renders Context before Relations, uses a single detail request per Drawer, moves advanced filters behind the default keyword/status controls, and keeps metadata collapsed or absent when empty.
- Validation passed: `git diff --check`, `python3 scripts/validate_ai_plan_structure.py`, `cd server && go run ./cmd/graft validate backend`, and `cd web && bun run check`.
- Recovery state advanced: contract, server projection, and Web implementation are complete; the topic now awaits
  cross-boundary automated validation, verification classification, and closeout. Browser inspection is conditional on
  explicit authorization; otherwise remaining product judgment is handed to human acceptance.

## Locked Decisions

- The web module never derives Compose ownership from Labels.
- Context is a reusable card with only normalized, typed fields; it supports Docker today and future runtime providers without exposing inspect payloads.
- Relations are resource-specific rather than Container-only.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "context-contract-and-design-guideline",
    "container-server-projection",
    "network-volume-web-ia"
  ],
  "pending_batches": [
    "cross-boundary-validation-and-closeout"
  ],
  "current_batch": "cross-boundary-validation-and-closeout",
  "next_batch": "verification-classification-and-closeout",
  "closeout_status": "in-progress"
}
```
