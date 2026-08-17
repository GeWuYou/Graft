---
name: graft-frontend-page-builder
description: Build or reshape Graft admin frontend pages using Vue 3, TypeScript, TDesign Vue Next, Pinia, Vue Router, Axios, UnoCSS, Graft page-type governance, and TDesign MCP. Use instead of generic Frontend App Builder, Frontend Design, Impeccable UI, Theme Factory, landing-page, or React/Tailwind/shadcn builder skills for Graft web work.
---

# Graft Frontend Page Builder

Use this skill for real Graft admin UI work, not standalone websites. The output must fit `web/src/modules/<name>` or shell-owned frontend boundaries and preserve existing route, menu, permission, i18n, and theme conventions.

## Workflow

1. Complete repository startup preflight and read `web/AGENTS.md`, root `DESIGN.md`, `ai-plan/design/architecture/前端架构设计.md`, `ai-plan/design/governance/frontend/前端视觉设计规范.md`, `ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md`, and `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` before implementation.
2. Classify the page type through existing Graft frontend governance, especially `$graft-web-vibe-coding` when page, shell, visual, copy, or prompt shaping is involved.
3. For new pages, page redesigns, or complex page changes, consume the bounded design brief produced by `$graft-web-vibe-coding` before coding. The brief must state:
   - page type and the primary operator/workflow
   - information density and hierarchy (what must be scannable first)
   - page header, primary action area, main content surface, and feedback surface
   - expected loading, empty, error, disabled, and destructive-action states
   - theme/token dependencies, responsive constraints, and i18n boundary
4. Convert the brief into the smallest existing Graft composition. Prefer TDesign Vue Next primitives, existing module/shared components, and repository tokens; do not invent a parallel component vocabulary or generic template.
5. Use TDesign Vue Next components through `$graft-web-vibe-coding` and the TDesign MCP docs when component API, DOM, or changelog detail is needed.
6. Keep implementation inside the existing Graft module structure. Do not create a second app, framework baseline, global style system, or local design system.
7. Build dense admin surfaces: clear tables, forms, filters, detail panels, drawers, dialogs, status indicators, and action bars. Avoid marketing hero pages, decorative card-heavy layouts, and oversized display copy.
8. Treat feedback and state communication as part of the page structure: status must not rely on color alone, controls need visible loading/disabled behavior, and text must remain stable within its container at narrow widths.
9. Keep visible copy i18n-safe and aligned with existing locale patterns.
10. Validate with the frontend entrypoint required by repository governance, normally `bun run check`, plus focused checks when appropriate.

## Replacement Map

- Frontend App Builder -> Graft module/page implementation under `web/src/modules/**`.
- Frontend Design / Impeccable -> Graft page-type workflow, TDesign components, repository tokens, responsive constraints.
- Theme Factory -> existing Graft theme tokens and TDesign theming only.
- Tailwind/shadcn/React starter -> reject unless repository docs are explicitly revised first.

## Constraints

- Do not introduce React, shadcn, Tailwind as a runtime baseline, or web/package dependency changes.
- Do not add visible in-app instructions about how to use the UI unless product copy already requires them.
- Do not use generic generated templates that bypass Graft module, API, contract, permission, or route ownership.
- Escalate to cross-boundary governance when the frontend symptom requires server descriptors, OpenAPI, typed contract, menu, route, or permission authority repair.

## Design Brief Consumption

The design brief is an implementation boundary, not a new design-system artifact. It is normally kept in the task
proposal or review record and does not become a global `MASTER.md`, generated catalog, or runtime configuration file.

When the brief conflicts with repository authority, preserve the authority order: page-type governance, TDesign Vue
Next, Graft tokens/theme, module ownership, then task-local preference. Resolve the conflict in the brief before coding.
