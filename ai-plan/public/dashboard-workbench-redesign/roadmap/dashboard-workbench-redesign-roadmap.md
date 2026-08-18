# Dashboard Workbench Redesign Roadmap

## Phase 1: Preview And Acceptance

- [x] Implement the development-only fixed-scenario workbench in the real application shell.
- [x] Validate five-state semantics, responsive information hierarchy, theme tokens, and i18n.
- [x] Capture desktop and full-page inspection artifacts, then complete at least one visual refinement.
- [x] Keep the formal homepage and all production contracts unchanged.

## Phase 2: Production Web Adoption

- [x] Introduce the accepted fixed-slot presentation projection over the existing dashboard summary.
- [x] Keep widget load status, display state, priority, and business presentation status separate.
- [x] Collapse normal health facts and keep optional-source failure visible without overstating impact.

## Phase 2.1: Existing-Contract Authority And Content Expansion

- [x] Enrich Monitor and Audit facts and register Announcement timeline, Backup health, and Runtime Target stat-group
  contributions through the existing Dashboard Registry and existing widget payload types.
- [x] Execute permission-filtered widget loaders with at most 4 concurrent workers, the request deadline, isolated
  source failures, and existing deterministic `priority -> order -> id` output regardless of completion order; keep
  runtime facts uncached.
- [x] Project the enriched existing Dashboard summary, widgets, bounded container summary/realtime state, and
  menu-derived quick actions without adding HTTP, OpenAPI, permissions, cache, or dependencies.
- [x] Keep the first screen bounded to Operational Status, Attention, System Health, and Module Coverage; show 5
  Attention rows and 3 Health rows by default, with the remainder available through an expandable detail surface.
- [x] Project `stat-group` as operational metric groups and `link-list` as contextual links capped at 6 before expand;
  keep contextual-link activation completely isolated from quick-action ranking.
- [x] Derive Module Coverage only from `system_summary`; derive source coverage only from the status of widgets that
  remain after server authorization filtering.
- [x] Keep the homepage container projection bounded to overview, Top 3 CPU, Top 3 memory, and at most 5 anomalies;
  route actions to canonical container list/details and delete `/infrastructure/docker/containers/resources` without an
  alias, redirect, compatibility menu, or replacement aggregate page.
- [x] Hide Recent Activity when no real timeline contribution exists; do not fabricate activity from other summaries.
- [x] Synchronize deterministic preview scenarios for normal, abnormal, permission-limited, no-sample, and long-list
  states; verify desktop 8/4, narrow-screen single-column order, themes, i18n, and keyboard expansion.
- [x] Complete formal-homepage machine browser inspection in the current dark/brand theme at 1920 × 1080,
  768 × 1024, and 390 × 844; repair Collapse keyboard focus/activation and expanded-list alignment, then re-run the
  validation justified for each repair (full Web after keyboard; focused Web after alignment).
- [ ] After focused/full Web validation and formal-homepage human acceptance, delete the development route,
  `pages/preview/**`, `presentation/preview-workbench.ts`, and the `dashboard.previewWorkbench.*` locale namespace
  together.

## Phase 3: Typed Attention Contract

- Add typed attention items, evidence state, server generation time, and source coverage to the OpenAPI authority.
- Refactor server loaders to return payload, control, metrics, and attention facts separately.
- Authorize actions at the server boundary and execute loaders under a bounded concurrent budget.
- Add container-owned attention contributions rather than deriving container severity in the page.

## Phase 4: Bounded Personalization

- Keep a fixed homepage information budget as module count grows.
- Add stable issue/resource identity and de-duplication before role or user policy.
- Introduce `server/modules/dashboard` only if persisted policy or personalization becomes a real product requirement.
