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
6. Select icons by the action or state they communicate, not ornament. Keep a consistent established icon family; pair unfamiliar or icon-only controls with a tooltip and accessible name, and hide decorative icons from assistive technology when visible text already conveys the meaning.
7. Treat empty-state visuals as secondary to a clear status and next action. Do not use generated imagery where a concise TDesign empty state, icon, or CSS treatment is sufficient.

## Motion Decision

- Prefer TDesign-native loading, transition, drawer, dialog, and feedback behavior.
- Use subtle motion only to clarify state changes, preserve spatial continuity, or make an actual wait understandable; loading feedback must never conceal blocked input or an unknown failure.
- Respect reduced-motion preferences when adding custom animation. The UI must remain understandable and fully operable without the motion.
- Keep custom transitions interruptible and independent from correctness: a rapid state change must replace the prior animation and settle to the current state without waiting for animation completion.
- Prefer transform and opacity for custom movement so transitions do not reflow dense tables, forms, or panels. Continuous motion is reserved for active progress indicators.
- Avoid motion that distracts from repeated admin workflows, hides latency, makes table/form scanning harder, or acts as decorative marketing choreography.

## Boundaries

- Do not add animation libraries, image pipelines, fonts, or package dependencies.
- Do not replace Graft theme tokens or TDesign styling with one-off visual systems.
- Do not add reusable assets without checking the active shared-asset governance and allowed scope.
