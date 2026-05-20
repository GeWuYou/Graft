# Graft Design System Templates

This folder stores non-runtime reference templates for Graft admin pages.

## Purpose

- Standardize page structure for Graft backend modules.
- Keep AI-generated pages aligned with the same visual grammar.
- Reuse TDesign Vue Next patterns without turning the folder into a runtime surface.

## Templates

- `shell.md` - app shell, navigation, and content frame.
- `auth.md` - login and account entry patterns.
- `overview-dashboard.md` - overview cards, metrics, charts, and status blocks.
- `list-form-detail.md` - CRUD-style list, form, drawer, and detail patterns.

## Rules

- Reference only, not importable runtime code.
- Prefer TDesign Vue Next primitives.
- Keep colors token-driven and state-driven.
- Do not treat starter/demo pages as production truth.
