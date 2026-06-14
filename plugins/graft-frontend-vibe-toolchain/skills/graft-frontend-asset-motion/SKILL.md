---
name: graft-frontend-asset-motion
description: Decide when Graft frontend work may use generated bitmap assets, existing visual assets, icons, or motion/animation guidance. Use for conditional Imagegen, UI Animation, asset, hero, illustration, empty-state, icon, transition, or motion requests in Graft web tasks while preserving admin-product restraint and repository asset governance.
---

# Graft Frontend Asset Motion

Use assets and motion only when they serve the admin workflow. Graft is a composable admin platform; most surfaces should be quiet, dense, and operational rather than marketing-oriented.

## Asset Decision

1. Prefer existing shared assets, TDesign icons, and established repository patterns.
2. Use lucide or existing icon libraries only when already enabled by the repo. Do not add a new icon package from this skill alone.
3. Use image generation only when the task genuinely needs a bitmap visual that cannot be better represented by existing assets, TDesign components, or CSS.
4. For generated bitmap work, use the system `imagegen` skill and keep outputs as assets only when the active task scope allows asset changes.
5. Reject decorative marketing backgrounds, gradient blobs, generic stock-like imagery, and large hero illustrations for normal admin pages.

## Motion Decision

- Prefer TDesign-native loading, transition, drawer, dialog, and feedback behavior.
- Use subtle motion only to clarify state changes, preserve spatial continuity, or reduce perceived latency.
- Respect reduced-motion expectations when adding custom animation.
- Avoid motion that distracts from repeated admin workflows, hides latency, or makes table/form scanning harder.

## Boundaries

- Do not add animation libraries, image pipelines, fonts, or package dependencies.
- Do not replace Graft theme tokens or TDesign styling with one-off visual systems.
- Do not add reusable assets without checking the active shared-asset governance and allowed scope.
