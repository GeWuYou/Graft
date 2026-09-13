---
name: graft-server-module-scaffold
description: "Shape a new Graft server module with explicit lifecycle, public contracts, routes, permissions, and validation."
---

# Graft Server Module Scaffold

Apply [root governance](../../../AGENTS.md) and reuse this task’s startup and validation evidence.

Use this skill when adding a new compile-time module under `server/modules/<name>`.

## Workflow

1. Confirm the capability belongs in a business module, not in `server/core`.
2. Choose a short, stable, lowercase module name.
3. Define module metadata:
   - `Name`
   - `Version`
   - `DependsOn`
4. Split lifecycle responsibilities clearly:
   - `Register` for routes, menus, permissions, migrations, jobs, and public services
   - `Boot` for runtime behavior
   - `Shutdown` for cleanup
5. Define any stable cross-module contract in `server/internal/moduleapi/**` or another documented stable boundary.
6. Expose capability-oriented interfaces, not repositories or raw database models.
7. Before writing code, define the module checklist:
   - route surface
   - menu entries
   - permission codes
   - migration registration
   - optional cron jobs
   - optional public services
   - optional homepage dashboard widget/component contribution
8. Do not add or require module-declared homepage quick link / quick action registration for new menu entries:
   - homepage quick actions are derived from visible sidebar / bootstrap menus
   - keep widget contribution review separate when the homepage needs module-owned summary UI
9. Add tests for dependency ordering, duplicate registration, and service resolution whenever those concerns are touched.
10. Capture reusable corrections through normal closeout when this task actually exposes one.

## Guardrails

- do not push business logic into platform core
- do not rely on hidden DI magic
- do not expose module internals as cross-module APIs
- do not create a historical plugin-named compatibility path
