# Runtime Targets Infrastructure IA Trace

## 2026-07-12 Work Intake And Authority Foundation

- Work Intake classified this as a long-running `feature`: design, roadmap, and active topic are required; ADR is not required because the approved IA is documented directly in the repository design authority.
- Added Runtime Target and Infrastructure design authority. It keeps the existing sidebar mode and limits Section Labels to non-interactive visual grouping.
- Repaired the navigation design conflict: Docker, Kubernetes, and Podman may be ordinary Infrastructure Provider menus only when matching registered Targets report real capabilities.
- Reframed Container list semantics: user-facing “source” is replaced by independent `deployment_type` and `runtime_target`; standalone containers remain outside Application.
- Reaffirmed Application/Project ownership of Compose lifecycle and the shared nature of Registry/Certificates.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation"],
  "pending_batches": ["menu-section-contract-and-sidebar-rendering", "runtime-target-foundation", "container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "work-intake-and-authority-foundation",
  "next_batch": "menu-section-contract-and-sidebar-rendering",
  "closeout_status": "batch-1-complete"
}
```

## 2026-07-12 Menu Section Contract And Sidebar Rendering

- Added `SectionKey` to backend menu entry metadata and rejected it on non-navigable domain groups; it does not create a menu node, route, permission, breadcrumb, tab, quick action, or search result.
- Kept the existing Bootstrap OpenAPI wire shape unchanged in this bounded batch. The web module registration attaches matching visual-only section metadata to the already authorized `/containers` route rather than emitting an undeclared wire field.
- Reclassified the visible Infrastructure entry as `Docker` while preserving the canonical `/containers` route and container API. The page semantic title remains `Containers`.
- The existing sidebar now renders a `Runtime` / `运行时` text label once before visible runtime entries. Header navigation does not render section labels, and collapsed sidebars hide them.
- Validation passed: focused Go and Vitest suites, `cd server && go run ./cmd/graft validate backend`, `cd web && bun run check`, and `git diff --check`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering"],
  "pending_batches": ["runtime-target-foundation", "container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "menu-section-contract-and-sidebar-rendering",
  "next_batch": "runtime-target-foundation",
  "closeout_status": "batch-2-complete"
}
```
