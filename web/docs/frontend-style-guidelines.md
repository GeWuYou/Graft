# Frontend Style Guidelines

This document records `web` frontend style rules that apply across business modules and shell pages.

## Typography Token Usage

Business-visible text in `web` must be token-driven. Page, module, and shell UI text must not directly use fixed
`font-size` values such as `12px`, `13px`, `14px`, `16px`, or similar hardcoded pixel scales.

Page text should prefer TDesign font tokens so light mode, dark mode, brand theme, and personalization font-size settings
can change text size consistently. When a page needs a different hierarchy, choose the nearest TDesign token first
instead of adding a local pixel value.

Recommended mapping:

| Usage                                            | Recommended token |
| ------------------------------------------------ | ----------------- |
| Helper text, hints, secondary metadata           | `body-small`      |
| Default body copy, table values, field values    | `body-medium`     |
| Card titles, compact section titles              | `title-small`     |
| Page sections, drawer titles, major panel titles | `title-medium`    |
| Page main titles                                 | `title-large`     |

Allowed hardcoded exception categories:

- Icon sizes.
- Chart axis, tooltip, and legend text.
- Logo typography or logo-mark text.
- Badge, avatar, and numeric badge internals.
- Fixed-format controls where layout dimensions are part of the control contract.
- Code editor or monospace preview surfaces.
- Necessary third-party component overrides.
- Height-coupled visual elements where text size must match a fixed visual asset or component height.

Every exception must record a reason near the declaration or in the owning style guideline. The reason should explain why
a TDesign token cannot represent the requirement for this specific surface.

### Review Checklist

- Business-visible text uses TDesign font tokens rather than hardcoded `font-size` pixel values.
- Any hardcoded font-size exception fits one allowed category and records a reason.
- The selected token matches the information hierarchy instead of being chosen only to match an old pixel size.
- Table cells, field values, helper text, drawer titles, card titles, and page titles follow the recommended token
  mapping unless a documented exception applies.
- Font size responds to personalization font-size settings.
- Personalization font-size control is used as the regression validation sample for typography changes.

## Scroll Containers

Business pages in `web` must not let scroll surfaces drift into browser-default scrollbar styling or page-local one-off behavior.

- `graft-scrollbar` is the only approved visual scrollbar treatment for `web` runtime surfaces.
- The repository may still contain two kinds of scroll responsibilities:
  - page or shell boundary containers that own viewport height and route-level overflow
  - internal scroll surfaces such as JSON viewers, log panes, drawer bodies, markdown tables, and embedded terminal panes
- Both kinds must render through `graft-scrollbar` when a scrollbar is visible. Browser-default white scroll tracks are
  not allowed in either class.
- Do not change component-library-owned vertical scroll containers, especially `t-drawer__body` on tabbed detail drawers,
  solely to force a `graft-scrollbar` class onto a custom child wrapper. Preserve the original scroll owner unless a
  browser-verified bug proves the ownership itself is wrong.
- When one page owns multiple internal scroll panels, the height authority must stay at the page or layout container
  boundary. Child components should fill the provided space and manage only their own internal overflow.
- For drawer-based JSON/log detail views, the default owner of vertical scrolling is the drawer body. `pre`, JSON viewers,
  log panes, and similar code surfaces may own their own local overflow, but they must not implicitly replace the
  drawer's main vertical scroll surface.
- Interactive embedded surfaces with their own scroll context, especially terminals and code/log viewers, must isolate
  scroll chaining so wheel input inside the surface does not continue scrolling the outer page container.
- New local `scrollbar-color`, `scrollbar-width`, `scrollbar-gutter`, or `::-webkit-scrollbar*` rules are forbidden
  unless they are covered by an explicit allowlist entry in the scrollbar governance script and a documented reason in
  the owning file or note.
- The scrollbar governance script is a blacklist gate for browser-native scrollbar styling only. It must fail on
  unallowlisted `scrollbar-*` or `::-webkit-scrollbar*` usage across `web/src`, but it must not infer scroll ownership
  changes from Drawer, JSON, terminal, or other component structure by itself.
- Scroll ownership remains a review/browser-QA concern. Page containers, JSON/code panes, and embedded terminals may
  all own different legitimate scroll contexts as long as they do not reintroduce browser-default scrollbar styling.

### Review Checklist

- Internal scroll viewports use `graft-scrollbar` instead of ad hoc scrollbar rules.
- Page-level containers, not child components, own viewport-height math for long-form tab or panel layouts.
- Page-level boundary scroll containers also render through `graft-scrollbar` and do not expose browser-default white tracks.
- Drawer-based JSON/log detail pages keep vertical scroll ownership on the drawer body unless there is explicit browser evidence for a different owner.
- Embedded terminals, log viewers, and code/JSON panes prevent wheel chaining from unintentionally moving the page.
- Any exception appears in the governance allowlist with a concrete reason and an expiry or cleanup trigger.
