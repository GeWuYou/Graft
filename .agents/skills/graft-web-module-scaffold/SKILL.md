---
name: graft-web-module-scaffold
description: Scaffold or shape a new Graft web module before implementation. Use when adding a Vue 3 admin feature under web/src/modules and Codex needs to align menu, route, page, api, and permission boundaries with the backend plugin contract and the repository's TDesign-based frontend conventions.
---

# Graft Web Module Scaffold

Use this skill when adding a new `web` feature module.

## Workflow

1. Identify the backend plugin or capability the module belongs to.
2. Define the minimum module path under `web/src/modules/<name>`.
3. Establish the required connection points before writing code:
   - API layer
   - route definition
   - page entry
   - permission mapping
   - menu metadata
4. Prefer the repository's standard admin shapes:
   - list page
   - create or edit dialog
   - detail drawer
   - search area
   - batch action area
5. Use TDesign as the primary component system.
6. Keep page-local state inside the page unless a shared store is clearly justified.
7. If the module changes backend menu or permission semantics, treat it as cross-boundary work and validate both sides.
8. At closeout, do not skip reusable-lesson evaluation:
   - prefer routing the slice through `graft-task-closeout`
   - if this skill is used as a self-contained implementation and closeout path, delegate the Experience Capture Check
     to `graft-lessons-learned`

## Guardrails

- do not invent parallel frontend-only authorization rules
- do not mix multiple UI libraries
- do not move page-local form state into a store without a clear shared-state reason
