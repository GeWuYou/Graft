# Navigation IA And Resource Route Refactor Trace

## 2026-07-12 Navigation Design, Topic, And Skill

- Work Intake classified the navigation refactor as long-running, with a design and active topic but no roadmap or ADR.
- Established `ai-plan/design/architecture/导航与资源路由信息架构规范.md` as the repository authority for seven navigation domains and stable resource-oriented UI routes.
- Added `graft-navigation-route-governance`; unresolved placement or resource boundary now blocks menu and UI route edits until the user decides.
- Preserved the rule that UI route migration has no aliases or redirects and excludes HTTP API endpoint changes.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill"],
  "pending_batches": [
    "backend-navigation-contract",
    "frontend-navigation-route-migration",
    "cross-boundary-validation-closeout"
  ],
  "current_batch": "navigation-design-topic-and-skill",
  "next_batch": "backend-navigation-contract",
  "closeout_status": "batch-1-complete"
}

## 2026-07-12 Backend Navigation Contract

- Added explicit `code`, `parent_code`, and `kind` graph semantics to the backend menu registry; bootstrap validates malformed graphs before projection and prunes empty domain groups after permission/config filtering.
- Registered the seven canonical domain groups in core and re-parented current menu entries without deriving hierarchy from UI paths.
- Migrated menu UI paths to the design authority, including `/projects`, `/containers`, `/system/**`, `/users`, `/roles`, `/permissions`, `/scheduled-tasks`, `/system-config`, and `/announcements`; HTTP API URIs remain unchanged.
- Extended the OpenAPI bootstrap menu schema and generated server/web contract consumers with explicit navigation metadata.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["navigation-design-topic-and-skill", "backend-navigation-contract"],
  "pending_batches": ["frontend-navigation-route-migration", "cross-boundary-validation-closeout"],
  "current_batch": "backend-navigation-contract",
  "next_batch": "frontend-navigation-route-migration",
  "closeout_status": "batch-2-complete"
}
```
```
