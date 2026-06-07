# Dashboard Contribution Trace

## 2026-06-07 - Topic Setup

- Branch renamed from `feat/system-configuration` to `feat/dashboard-contribution`.
- Startup receipt established:
  - governance source: root `AGENTS.md`
  - task class: `cross-boundary`
  - recovery source: `parent topic`
  - authority summary: `server` runtime/module registries declare Dashboard widget contributions; `openapi/**` owns the wire contract; `web` consumes generated OpenAPI types and renders generic dashboard widgets.
- Final architecture decision:
  - MVP implementation starts in `server/internal/dashboard`.
  - The internal package is limited to registry, definitions, loader contract, and aggregate route.
  - Future dashboard persistence, layout, presets, favorites, recent visits, and preferences should move to a future `server/modules/dashboard`.
- Final widget contract decision:
  - Use `type + payload`.
  - Avoid `oneOf` and typed-slot payloads for MVP because current `openapi-typescript` and `oapi-codegen` generation would add avoidable complexity.
- Initial loop budget:
  - loop mode: `topic-completion-loop`
  - max rounds: 5
  - max commits: 5
  - max runtime: bounded by active session
  - validation failure policy: stop on directly affected validation failure

