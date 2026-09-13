---
name: graft-web-vibe-coding
description: "Design or adjust Graft admin UI using existing page masters, TDesign, theme tokens, and locale ownership; scale planning to the change."
---

# Graft Web Vibe Coding

Use this skill for `web` page work that needs design governance, prompt discipline, or visible UI cleanup.

Follow [root governance](../../../AGENTS.md) and [web guidance](../../../web/AGENTS.md). Reuse existing reading and review evidence. Read [DESIGN.md](../../../DESIGN.md), [visual design rules](../../../ai-plan/design/governance/frontend/前端视觉设计规范.md), and the relevant page master when creating or materially restructuring a page; a local copy/style fix only needs its affected guidance.

## 1. Resolve TDesign uncertainty

Use [TDesign guidance](../../../ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md) for new or uncertain component APIs, DOM assumptions, selectors, or version behavior. Reuse trustworthy evidence from this task. With `framework=vue-next`, query component docs for API uncertainty, DOM for uncertain structure/overrides, and changelog for version risk. Query the component list only when the name or availability is unknown.

When MCP is unavailable, use official documentation and record the concrete remaining gap. Report material query/adoption decisions; do not require a ritual query or an inapplicable report for an unchanged known API.

## 2. Declare page type first

For new pages, redesigns, or changed information hierarchy, select the page master; for a small fix reuse the existing page classification:

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

For these tasks, direct implementation is allowed:

- visible copy fixes
- style fixes
- small interaction fixes

For small fixes, check only the behavior, copy, accessibility, and tokens actually affected; do not require a full design brief.

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

- new or uncertain TDesign behavior has supporting component evidence
- page type is identified when the page structure changes
- dominant intent and the bounded design brief are recorded for new, redesigned, or complex pages
- structure matches the page type
- loading, empty, error, disabled, and destructive states are covered where applicable
- visible copy is clean
- i18n ownership is correct
- token/theme response is intact
- no second UI baseline was introduced
- verification classification is recorded; browser interaction is not started unless [web/AGENTS.md](../../../web/AGENTS.md) permits it and
  task-local authorization exists
- when visual or product judgment remains, hand off the minimal human acceptance contract instead of treating a
  screenshot as final acceptance
- capture a reusable lesson through normal closeout only when this task exposed one
