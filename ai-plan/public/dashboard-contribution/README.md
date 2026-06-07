# Dashboard Contribution

## Current Status

- Status: `in-progress`.
- Branch: `feat/dashboard-contribution`.
- Loop mode: `topic-completion-loop` through `$graft-multi-agent-loop`.
- Startup receipt:
  - governance source: root `AGENTS.md`
  - task class: `cross-boundary`
  - recovery source: `parent topic`
  - authority summary: `server` runtime/module registries declare Dashboard widget contributions; `openapi/**` owns the wire contract; `web` consumes generated OpenAPI types and renders generic dashboard widgets without importing module business components.

## Final Architecture Decision

Dashboard MVP uses a contribution mechanism, not a hard-coded business dashboard.

- `server/internal/dashboard` owns only the MVP platform contribution surface:
  - registry
  - widget definition
  - widget loader contract
  - aggregate HTTP route
- It must not own persistence, user preferences, layout presets, favorites, recent visits, announcements, or business dashboard data.
- Future user workspace capabilities should move to `server/modules/dashboard`, while the core registry remains an internal runtime surface.
- Widget contribution happens during module `Register(ctx)`, aligned with menu, permission, cron, and config registries.

## Contract Decision

Widget wire shape uses `type + payload`.

```json
{
  "id": "core.module-runtime-health",
  "module_key": "core",
  "type": "health",
  "payload": {}
}
```

MVP avoids OpenAPI `oneOf` and typed-slot payloads because the current generated chains are:

- `web`: `openapi-typescript 7.13.0`
- `server`: `oapi-codegen v2.7.0`

The OpenAPI source should define concrete payload schemas for documentation and local mapping tests, but `DashboardWidget.payload` should remain a plain object in the top-level widget contract.

## Widget Namespace Rules

- Widget `id` is globally unique.
- Standard shape: `<namespace>.<widget_key>`.
- `namespace` should match the server module descriptor `moduleID` when the widget is module-owned.
- Core widgets use `core.<widget_key>`.
- Duplicate registration is a startup failure; never overwrite.

Initial widget IDs:

- `core.module-runtime-health`
- `rbac.access-summary`

## MVP Fixed System Summary

Keep the fixed area intentionally small:

- current user: username and display name from request auth context
- environment: `config.App.Env`
- locale: default and fallback locale
- modules: total/enabled/degraded summary from module runtime snapshot
- visible widgets: count after permission filtering

Do not add uptime, version, DB/Redis health, system load, recent visits, favorites, or BI metrics in MVP.

## Acceptance Conditions

- Home dashboard has a fixed system summary plus module-contributed widgets.
- Dashboard page does not import audit, scheduler, rbac, user, monitor, or system-config business components.
- Frontend renderer branches only by stable `DashboardWidgetType`, not by `module_key` or concrete widget id.
- Widget data is loaded through backend aggregation, not frontend N-interface composition.
- Permission filtering happens server-side; loaders still validate sensitive data boundaries.
- Phase 1 includes no more than two required widgets:
  - `core.module-runtime-health`
  - `rbac.access-summary`
- No dashboard persistence tables are introduced in MVP.

