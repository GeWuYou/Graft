# Graft Design System

Graft is a control console, not a marketing site.

## Intent

- Keep `web` token-driven, structured, and console-first.
- Make AI-generated pages predictable enough that design quality does not depend on handcrafted prompts every time.
- Keep the current theme workbench as the only intentionally expressive accent.

## Visual Stance

- Calm, precise, structured, and slightly premium.
- Prefer clear hierarchy over decorative density.
- Prefer operational clarity over novelty.

## Color

- Use TDesign tokens first: `--td-*`, brand theme, and semantic status colors.
- Keep page, container, border, and text layers token-driven.
- Use raw hex colors only as examples or final fallback.
- Avoid purple-biased defaults, neon gradients, and hard-coded single-mode palettes.

## Typography

- Follow TDesign font tokens and existing backend-console scale.
- Use strong titles, compact supporting text, and tabular numbers where data must read quickly.
- Keep copy direct and operational.

## Page Type Registry

Built-in page masters for the first stage:

- `shell`: app shell, navigation, top bar, content container.
- `auth`: login, session-expired, and authentication-related pages.
- `overview-dashboard`: overview, monitor, status, and metrics pages.
- `list-form-detail`: CRUD-heavy admin pages such as users, roles, permissions, settings, and resources.

Extension page types may be registered when a page does not naturally fit the first-stage masters:

- `settings`
- `workflow`
- `editor`
- `query-builder-list-detail`
- `log-audit`
- `operation-result`
- `error-result`
- `docs-help`

For `query-builder-list-detail` and `log-audit` pages, quick presets must compile into visible, editable filter fields.
Use `operation-result` for success/fail operation outcome pages; keep `error-result` for exception, maintenance, network, and browser result pages.

Rules:

- Every frontend task must declare a page type before implementation.
- The four built-in masters are not the full page-type universe.
- If a task does not fit them, register an extension type first with:
  - information hierarchy
  - component composition
  - state set
  - theme response rules
  - i18n requirements
  - acceptance rules

## Composition

- Base shell: header, side nav, breadcrumb, content, and explicit actions.
- Standard backend pages should think in `page header`, `primary action area`, `main content surface`, and `feedback surface`.
- Different page types may trim those surfaces; do not force every page into a table/card/detail shape.
- Use cards for grouping, not for decoration.

## Components

- Prefer TDesign Vue Next primitives: `Layout`, `Menu`, `Card`, `Table`, `Form`, `Drawer`, `Dialog`, `Tag`, `Alert`, `Tabs`, `Result`.
- For monitor pages, keep charts inside token-aware cards with responsive legend/tooltip/axis colors.
- For auth pages, keep the layout focused and frictionless.

## Theme Compatibility

- Light mode, dark mode, and custom brand theme changes must all preserve readability.
- Charts, tags, borders, empty states, and feedback panels must react to mode and token changes.
- Raw hex values are allowed only as last-resort fallbacks when token values are unavailable.
- Do not ship page-local palettes that only work in one mode.

## Copy Rules

- User-visible copy must sound product-facing, not implementation-facing.
- Menu labels, page hints, empty states, help text, and action labels must not leak migration notes, demo labels, AI debug text, or contract-governance jargon.
- Internal docs, tests, comments, and `ai-plan/` are allowed to use engineering/governance terms where appropriate; this rule is for visible UI only.

## Motion

- Use short, useful motion only.
- Hover, reveal, and drawer/dialog transitions should feel quick and controlled.
- Avoid ornamental animation loops.

## Do / Don’t

- Do: token-driven colors, compact information density, explicit state labels, reusable page skeletons.
- Do: keep `web/ai-libs/tdesign-vue-next-starter` as reference only.
- Do: declare page type before coding and reuse the registered master or extension rules.
- Don’t: introduce a second UI baseline, mock/demo routing, or marketing-style layouts.
- Don’t: turn a backend page header into a marketing hero block.
- Don’t: guess TDesign DOM structure without checking docs or MCP.

## Agent Prompt Guide

Use this phrasing when generating a new or heavily reworked page:

> Build a Graft admin page using TDesign Vue Next. First declare the page type, then keep the page token-driven, structured, and console-first. Use a backend-style page header, explicit action areas, and theme-responsive surfaces. Avoid decorative marketing layouts, avoid visible demo/debug copy, and keep the current theme-workbench accent as the only intentionally expressive element.

For large tasks, require the AI to return before coding:

- page type
- information hierarchy
- component composition
- state set
- theme response points
- i18n keys or locale ownership

For simple copy/style/interaction fixes, direct implementation is allowed, but the result must still pass page-type, i18n, theme, and visible-copy self-checks.

## Acceptance

- Page structure matches its declared type.
- Visible copy is product-facing and free of demo/debug leakage.
- Colors and states are token-driven.
- Light mode, dark mode, and custom theme changes keep the page readable.
- The result still looks like Graft, not a starter demo or a marketing template.

## References

- Detailed spec: `ai-plan/design/前端视觉设计规范.md`
- Reference templates: `ai-plan/design/graft-design-system/`
