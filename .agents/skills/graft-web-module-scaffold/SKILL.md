---
name: graft-web-module-scaffold
description: "Shape a new Graft Vue module around its server capability, routes, API, permissions, and TDesign conventions."
---

# Graft Web Module Scaffold

Apply [root governance](../../../AGENTS.md) and reuse this task’s startup and validation evidence.

Use this skill when adding a new `web` feature module.

## Workflow

1. Identify the server module or capability the web module belongs to.
2. Define the minimum module path under `web/src/modules/<name>`.
3. Establish the required connection points before writing code:
   - API layer
   - route definition
   - page entry
   - permission mapping
   - menu metadata
   - optional homepage widget/component contribution
4. Prefer the repository's standard admin shapes:
   - list page
   - create or edit dialog
   - detail drawer
   - search area
   - batch action area
5. Use TDesign as the primary component system.
6. Keep page-local state inside the page unless a shared store is clearly justified.
7. If the module changes backend menu or permission semantics, treat it as cross-boundary work and validate both sides.
8. Do not ask whether a new menu needs a separate homepage quick action or quick link registration:
   - homepage quick actions are derived from the currently visible sidebar / bootstrap menus
   - only evaluate whether this module needs a homepage dashboard widget or component contribution
10. Capture reusable corrections through normal closeout when this task actually exposes one.

## Guardrails

- do not invent parallel frontend-only authorization rules
- do not mix multiple UI libraries
- do not move page-local form state into a store without a clear shared-state reason
