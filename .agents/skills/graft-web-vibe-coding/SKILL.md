---
name: graft-web-vibe-coding
description: Repository-specific frontend workflow for Graft web pages, shell surfaces, visible copy, and AI prompt shaping. Use when a task should first declare a page type, choose one of the first-stage built-in page masters or register an extension type, then implement token-driven, theme-responsive, i18n-safe UI in TDesign Vue Next.
---

# Graft Web Vibe Coding

Use this skill for `web` page work that needs design governance, prompt discipline, or visible UI cleanup.

Follow root `AGENTS.md`, `web/AGENTS.md`, `DESIGN.md`, `ai-plan/design/governance/frontend/前端视觉设计规范.md`, and the relevant
template under `ai-plan/design/graft-design-system/`.

## 1. Run TDesign MCP preflight before coding

Before page-type planning or implementation, decide whether the current slice changes `TDesign Vue Next` component
usage.

If yes:

- query TDesign MCP before coding, with `framework=vue-next`
- only query the components touched by this slice; do not do a full-library sweep
- use the minimum relevant MCP calls:
  - `get_component_list`
    - confirm the component name and whether `vue-next` provides the expected component
  - `get_component_docs`
    - confirm props, events, slots, supported usage, and recommended practice
  - `get_component_dom`
    - required for style overrides, DOM structure assumptions, slot layout assumptions, or selector work
  - `get_component_changelog`
    - required when the task involves upgrade risk, version drift, or behavior differences across versions
- record closeout evidence with:
  - `ui_component_change: yes`
  - `mcp_queried: yes`
  - `framework: vue-next`
  - `components: ...`
  - `queries: ...`
  - `adoption: adopted | partially_adopted | not_adopted`
  - `reason: ...`

If no:

- explicitly record `TDesign MCP preflight: not applicable`
- record closeout evidence with `ui_component_change: no`, `mcp_queried: no`, and `framework: not-applicable`

If MCP is unavailable:

- fall back to official TDesign documentation only
- record `mcp_queried: fallback_to_official_docs`, the fallback reason, and the affected components in closeout

Do not postpone this preflight to validation or post-implementation review.

## 2. Declare page type first

Before coding, classify the task as one of:

- `shell`
- `auth`
- `overview-dashboard`
- `list-form-detail`

These are the first-stage built-in page masters, not the full set of page types.

If the task does not fit them naturally, register an extension type first and define:

- information hierarchy
- component composition
- state set
- theme response rules
- i18n requirements
- acceptance rules

## 3. Split by task size

For these tasks, return a structure proposal before coding:

- new pages
- page rewrites
- complex layout work
- any change that alters information hierarchy or interaction model

The structure proposal must include:

- page type
- `page header`
- `primary action area`
- `main content surface`
- `feedback surface`
- theme/token dependencies
- i18n ownership

For these tasks, direct implementation is allowed:

- visible copy fixes
- style fixes
- small interaction fixes

Even then, still run the same self-checks.

## 4. Build the page the Graft way

- Use `TDesign Vue Next` as the only runtime UI system.
- Treat `web/ai-libs/tdesign-vue-next-starter` as reference only.
- Use token-driven surfaces, borders, text, and status colors.
- Keep layout console-first and operational; do not introduce marketing-style hero treatment.
- Prefer explicit backend composition over ornamental layouts.

## 5. Copy and i18n rules

- Visible UI copy must be product-facing.
- Do not ship AI debug text, migration notes, demo labels, or implementation-phase explanations in user-facing copy.
- Keep menu titles, page titles, empty states, and helper text inside the correct locale boundary.
- `title_key` remains the stable truth for menu and route titles.

## 6. Theme compatibility rules

- Light mode, dark mode, and custom brand theme must all remain readable.
- Charts, tags, borders, cards, dialogs, and feedback states must react to token changes.
- Use raw hex values only as last-resort fallback.

## 7. Final self-check

Before handing off:

- TDesign MCP preflight is recorded as `used`, `not applicable`, or `fallback to official docs`
- when preflight was `used`, the closeout names the queried components, query types, adoption status, and reason
- page type is declared
- structure matches the page type
- visible copy is clean
- i18n ownership is correct
- token/theme response is intact
- no second UI baseline was introduced
- reusable-lesson evaluation is completed through `graft-task-closeout`, or explicitly delegated to
  `graft-lessons-learned` when this skill owns a self-contained closeout
