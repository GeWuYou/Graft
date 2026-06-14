---
name: graft-frontend-figma-intake
description: Intake Figma links, screenshots, design exports, or design notes as reference material for Graft frontend work. Use when a user provides Figma-derived input, but keep repository docs, existing code, Graft page-type governance, and TDesign Vue Next authority above Figma. Do not add Figma SDKs or make Figma the source of truth.
---

# Graft Frontend Figma Intake

Treat Figma as reference evidence, not authority. Translate useful intent into Graft admin patterns and reject incompatible framework, styling, routing, or dependency assumptions.

## Workflow

1. Capture what the Figma input is meant to convey: information architecture, density, component state, spacing, copy, interaction, or asset direction.
2. Map it to existing Graft page type, route/menu/module ownership, TDesign components, theme tokens, and i18n rules.
3. Preserve product semantics over pixel matching when the design conflicts with repository governance.
4. Use `$graft-frontend-page-builder` for implementation and `$graft-frontend-browser-qa` for visual/browser verification.
5. If the Figma reference implies new shared assets or governance changes, stop at a scoped recommendation unless the active task permits those files.

## Accepted Inputs

- User-provided screenshots, exported images, text notes, measurements, or component descriptions.
- Browser-accessible design references when credentials and policy allow normal browsing.
- Existing repository design docs and current UI as stronger local references.

## Boundaries

- Do not install Figma SDKs, plugins, CLIs, token sync tools, or package dependencies.
- Do not treat design tokens, component names, or layout primitives from Figma as canonical when they conflict with Graft or TDesign.
- Do not copy external proprietary design assets into the repo unless the user confirms rights and the allowed scope includes assets.
