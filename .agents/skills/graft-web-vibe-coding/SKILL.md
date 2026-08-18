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

## 2.1 Route intent into a bounded design brief

Before choosing visual treatment, classify the request by its dominant intent:

- `workflow`: repeated operational action, approval, editing, or bulk work
- `scan`: dashboard, list, status comparison, or monitoring surface
- `explain`: detail, audit, history, or configuration context
- `navigate`: shell, auth, route entry, or cross-module wayfinding
- `feedback`: loading, empty, error, disabled, or destructive-action behavior

Use the intent only to choose the relevant Graft page master and quality checks. It must not introduce a new page type,
framework, token set, or visual baseline. A mixed request may have one primary intent and secondary checks.

For new pages, redesigns, and complex layout changes, emit a concise design brief before implementation:

- `page_type` and primary operator/workflow
- information hierarchy and expected scan order
- `page_header`, `primary_action_area`, `main_content_surface`, and `feedback_surface`
- state set: loading, empty, error, disabled, success, and destructive confirmation where applicable
- responsive constraints for desktop and narrow containers, including text overflow behavior
- theme/token dependencies and i18n ownership
- acceptance checks tied to the selected intent

The brief is task-local planning evidence. Do not persist it as a global design system, generated data catalog, or
runtime artifact.

## 3. Split by task size

For these tasks, use the design brief above as the sole structure proposal before coding. Do not create a separate
structure-proposal artifact with a second field set or recording location.

- new pages
- page rewrites
- complex layout work
- any change that alters information hierarchy or interaction model

The design brief must include:

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
- Let intent shape emphasis, not technology: workflow pages optimize action clarity, scan pages optimize comparison,
  explain pages optimize context, navigate pages optimize orientation, and feedback intent verifies recoverable states.
- For every interactive state, provide a non-color cue and preserve stable layout dimensions so loading, validation, or
  long localized text does not move adjacent controls unexpectedly.

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
- dominant intent and the bounded design brief are recorded for new, redesigned, or complex pages
- structure matches the page type
- loading, empty, error, disabled, and destructive states are covered where applicable
- visible copy is clean
- i18n ownership is correct
- token/theme response is intact
- no second UI baseline was introduced
- verification classification is recorded; browser interaction is not started unless `web/AGENTS.md` permits it and
  task-local authorization exists
- when visual or product judgment remains, hand off the minimal human acceptance contract instead of treating a
  screenshot as final acceptance
- reusable-lesson evaluation is completed through `graft-task-closeout`, or explicitly delegated to
  `graft-lessons-learned` when this skill owns a self-contained closeout
