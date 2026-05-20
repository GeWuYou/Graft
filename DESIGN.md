# Graft Design System

Graft is a control console, not a marketing site.

## Visual Stance

- Calm, precise, structured, and slightly premium.
- Prefer clear hierarchy over decorative density.
- Keep one intentional accent: the theme workbench/button-bar feel may stay expressive.

## Color

- Use TDesign tokens first: `--td-*`, brand theme, and semantic status colors.
- Keep page, container, border, and text layers token-driven.
- Use raw hex colors only as examples or final fallback.
- Avoid purple-biased defaults, neon gradients, and hard-coded single-mode palettes.

## Typography

- Follow TDesign font tokens and existing backend-console scale.
- Use strong titles, compact supporting text, and tabular numbers where data must read quickly.
- Keep copy direct and operational.

## Layout

- Base shell: header, side nav, content, and explicit page actions.
- Standard pages: hero summary, filter/search area, table or cards, then detail surfaces such as drawers/dialogs.
- Use cards for grouping, not for decoration.

## Components

- Prefer TDesign Vue Next primitives: `Layout`, `Menu`, `Card`, `Table`, `Form`, `Drawer`, `Dialog`, `Tag`, `Alert`, `Tabs`, `Result`.
- For monitor pages, keep charts inside token-aware cards with responsive legend/tooltip/axis colors.
- For auth pages, keep the layout focused and frictionless.

## Motion

- Use short, useful motion only.
- Hover, reveal, and drawer/dialog transitions should feel quick and controlled.
- Avoid ornamental animation loops.

## Do / Don’t

- Do: token-driven colors, compact information density, explicit state labels, reusable page skeletons.
- Do: keep `web/ai-libs/tdesign-vue-next-starter` as reference only.
- Don’t: introduce a second UI baseline, mock/demo routing, or marketing-style layouts.
- Don’t: guess TDesign DOM structure without checking docs or MCP.

## Agent Prompt Guide

Use this phrasing when generating a page:

> Build a Graft admin page using TDesign Vue Next. Keep it token-driven, structured, and console-first. Use standard backend patterns, avoid decorative marketing layouts, and keep the current theme-workbench accent as the only intentionally expressive element.

## References

- Detailed spec: `ai-plan/design/前端视觉设计规范.md`
- Reference templates: `ai-plan/design/graft-design-system/`
