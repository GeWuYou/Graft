---
name: graft-plugin-scaffold
description: Scaffold or shape a new Graft server plugin before implementation. Use when adding a backend capability that should live under server/plugins, and Codex needs to define the plugin lifecycle, dependencies, public service boundaries, permissions, menus, migrations, routes, jobs, and validation scope in the repository's standard pattern.
---

# Graft Plugin Scaffold

Use this skill when adding a new `server` plugin.

## Workflow

1. Confirm the capability belongs in a plugin, not in `server/core`.
2. Choose a short, stable, lowercase plugin name.
3. Define plugin metadata:
   - `Name`
   - `Version`
   - `DependsOn`
4. Split lifecycle responsibilities clearly:
   - `Register` for routes, menus, permissions, migrations, jobs, and public services
   - `Boot` for runtime behavior
   - `Shutdown` for cleanup
5. Define any cross-plugin contract in `internal/pluginapi` or an equivalent stable boundary.
6. Expose capability-oriented interfaces, not repositories or raw database models.
7. Before writing code, define the plugin checklist:
   - route surface
   - menu entries
   - permission codes
   - migration registration
   - optional cron jobs
   - optional public services
8. Add tests for dependency ordering, duplicate registration, and service resolution whenever those concerns are touched.
9. At closeout, do not skip reusable-lesson evaluation:
   - prefer routing the slice through `graft-task-closeout`
   - if this skill is used as a self-contained implementation and closeout path, delegate the Experience Capture Check
     to `graft-lessons-learned`

## Guardrails

- do not push business logic into platform core
- do not rely on hidden DI magic
- do not expose plugin internals as cross-plugin APIs
