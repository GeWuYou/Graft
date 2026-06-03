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
