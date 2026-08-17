# Dashboard Workbench Redesign

## Current Status Summary

- Topic objective: redesign the global dashboard into a stable operational workbench without letting module count control page geometry.
- Current status: `active · formal-homepage-human-acceptance-pending`
- Task class: `cross-boundary`
- Intake summary: a long-running feature with repository-wide design truth, active recovery material, and a topic-local roadmap.
- Canonical authority:
  - `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
  - `server/internal/dashboard/**` and `openapi/**` for production contribution facts
  - `web/src/modules/dashboard/**` for the dashboard presentation policy and preview
- Completed so far: authority discovery, status-semantics audit, TDesign MCP preflight, Work Intake bootstrap, the deterministic preview, visual refinement, production Web adoption, and the corrective repair for key-first alerts and duplicate platform health facts.
- In progress: human inspection and annotation of the refreshed formal homepage.
- Not started yet: the additive typed attention contract evolution.

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary: server modules own contribution facts; OpenAPI owns shared wire contracts; the dashboard module owns presentation policy.

## Owned Scope

- `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
- `ai-plan/public/dashboard-workbench-redesign/**`
- `web/src/modules/dashboard/**`
- `server/internal/httpx/accesslog_dashboard.go` and its focused tests
- `openapi/components/schemas/dashboard-alert-list-payload.yaml` and generated contract projections
- `server/internal/app/runtime_dashboard.go`, `server/modules/monitor/dashboard_widget.go`, their locale resources, and focused tests
- the development-only dashboard preview route registration

Out of scope:

- changing the Dashboard Registry shape or capability/monitor public APIs
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
- Formal-homepage acceptance found two corrective issues: key-first audit/access alerts were rejected because the OpenAPI and Web guard incorrectly required fallback `title`, and Monitor duplicated PostgreSQL/Redis facts already owned by CapabilityCoordinator.
- The corrective batch now treats `title_key` as authoritative and fallback `title` as optional, keeps Monitor focused on active anomalies, and leaves PostgreSQL/Redis/outbound-network health with CapabilityCoordinator.
- Contract generation, full server/Web validation, and independent review passed. The authorized in-app browser remains open for annotation, but its local-address policy blocked automated refresh; final visual acceptance therefore remains explicitly human-owned.

## Work Intake

- This topic was created through `Work Intake`.
- The full `Work Contract` is stored in the tracking file.

## Pending Batch Direction

- Manually refresh the open formal homepage, collect final annotations, and record visual acceptance.
- Remove the preview after acceptance, then plan the additive typed-attention contract slice separately.

## Validation Targets

```bash
git diff --check
python3 scripts/validate_ai_plan_structure.py
just openapi-check
cd server && go run ./cmd/graft validate backend
cd web && bun run check
```

## Execution Entry

- Preferred entry: `ai-plan/public/dashboard-workbench-redesign/startup-prompt.md`
- The corrective batch may use `graft-multi-agent-batch` with disjoint OpenAPI/Web and server-health write scopes; the main agent owns integration, full validation, browser acceptance, and closeout.
