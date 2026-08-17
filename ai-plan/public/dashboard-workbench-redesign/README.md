# Dashboard Workbench Redesign

## Current Status Summary

- Topic objective: redesign the global dashboard into a stable operational workbench without letting module count control page geometry.
- Current status: `active · production-web-adoption-implemented`
- Task class: `cross-boundary`
- Intake summary: a long-running feature with repository-wide design truth, active recovery material, and a topic-local roadmap.
- Canonical authority:
  - `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
  - `server/internal/dashboard/**` and `openapi/**` for production contribution facts
  - `web/src/modules/dashboard/**` for the dashboard presentation policy and preview
- Completed so far: authority discovery, status-semantics audit, TDesign MCP preflight, Work Intake bootstrap, the deterministic preview, visual refinement, and production Web adoption over the existing summary contract.
- Not started yet: the additive typed attention contract evolution.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: server modules own contribution facts; the dashboard module owns presentation policy; OpenAPI owns future wire contracts.

## Owned Scope

- `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
- `ai-plan/public/dashboard-workbench-redesign/**`
- `web/src/modules/dashboard/**`
- `server/internal/httpx/accesslog_dashboard.go` and its focused tests
- the development-only dashboard preview route registration

Out of scope:

- changing OpenAPI contracts or the Dashboard Registry shape
- introducing a second UI library, chart baseline, menu entry, or permission

## Locked Decisions

1. The page type is `overview-dashboard`, and the production registry continues to contribute structured facts rather than arbitrary Vue components.
2. Presentation status is independent from widget load status, display state, and ordering priority.
3. The preview route is development-only and must be deleted when the accepted design replaces the production homepage.

## Phase Plan

- Phase 1: deterministic development-only preview, tests, browser evidence, and human visual acceptance.
- Phase 2: production Web presentation projection over the existing summary response.
- Phase 3: additive typed attention, coverage, freshness, and bounded concurrent server aggregation.

## Current Recovery Point

- Work Intake bootstrap and design authority are established.
- Preview v2 improves secondary-text readability, separates overall status from concrete Attention items, restores the authenticated shell menu, groups Health and Resources in the right rail, and renders a configurable set of independent Quick Action items.
- “全部入口” now projects the current authorized top-level navigation into a searchable compact list; it uses the shared responsive workspace surface as a desktop Drawer and a narrow-screen fullscreen workspace.
- The production `/` homepage now uses the accepted workbench component and a pure typed-payload projector; the preview remains available for final comparison.
- Access-log authority now maps 4xx and slow requests to warning and reserves error for 5xx.
- Next step: collect final human acceptance of `/`, then remove the development preview in the same bounded slice.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is stored in the tracking file.

## Pending Batch Direction

- Collect final production-homepage annotations and visual acceptance.
- Remove the preview after acceptance, then plan the additive server/OpenAPI contract slice separately.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
cd web && bun run check
```

## Execution Entry

- Preferred entry: `ai-plan/public/dashboard-workbench-redesign/startup-prompt.md`
- Current prototype batch is main-agent-owned; `graft-multi-agent-batch` is limited to read-only review.
