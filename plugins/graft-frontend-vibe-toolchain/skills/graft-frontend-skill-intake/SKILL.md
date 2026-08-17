---
name: graft-frontend-skill-intake
description: Normalize external frontend website-building, vibe-coding, UI generation, design-system, animation, screenshot, Playwright, Figma, accessibility, or asset-generation skill requests against Graft repository governance. Use before applying article-derived frontend skills, third-party frontend advice, or generic website-builder workflows to Graft web work.
---

# Graft Frontend Skill Intake

Classify incoming frontend skill requests before implementation. Preserve Graft's Vue 3, TDesign Vue Next, module, route, menu, i18n, and validation authority; reject workflows that would create a second frontend baseline.

## Intake

1. Run the normal repository startup preflight first. For frontend work, read `web/AGENTS.md` and the relevant `ai-plan/design/` frontend governance docs before edits.
2. Identify the request category:
   - **direct**: Graft admin page, shell surface, TDesign component composition, route/menu/page/module work.
   - **conditional**: browser QA, screenshots, Figma intake, image generation, animation, accessibility review.
   - **rejected**: React, shadcn, Tailwind runtime baseline, marketing landing-page builders, standalone apps, Figma SDK setup, Playwright test dependency, package changes without authority.
   - **future-only**: broad design-system generation, reusable animation libraries, automated visual regression infrastructure, or shared asset registries outside the allowed task scope.
3. Route direct page work to `$graft-frontend-page-builder`.
4. Route browser verification and accessibility-oriented review to `$graft-frontend-browser-qa`.
5. Route image or motion decisions to `$graft-frontend-asset-motion`.
6. Route Figma references to `$graft-frontend-figma-intake`.

## UI/UX Intelligence Routing

When an external UI/UX intelligence request resembles UI/UX Pro Max, use the smallest matching intent below. The
reference is a source of heuristics only; Graft's page type, TDesign, token, i18n, route, and validation authorities
remain binding.

| Request intent | Route | Graft-safe result |
| --- | --- | --- |
| New admin page or coherent visual direction | `$graft-frontend-page-builder` | A short design brief covering operator context, information density, page type, states, and existing tokens; no new global design system. |
| Focused accessibility, feedback, layout, typography, color, chart, or navigation concern | `$graft-frontend-page-builder` for implementation, then `$graft-frontend-browser-qa` for verification | Check one observable outcome at a time: labels/focus, non-color state cues, loading/error/empty states, overflow, responsive stability, and chart alternatives. |
| Motion, icon, illustration, or asset choice | `$graft-frontend-asset-motion` | Use existing Graft/TDesign assets and restrained, state-expressive motion with reduced-motion behavior. |
| Screenshot or runtime quality evidence | `$graft-frontend-browser-qa` | Verify the rendered behavior at relevant desktop and narrow-container sizes; do not add a new visual-regression stack. |
| React, Tailwind, shadcn, native/mobile, or generic framework implementation advice | rejected | Translate only the underlying product or accessibility principle, then apply Vue 3 + TDesign conventions. |

For a new page, collect product/context/style constraints before implementation. For a focused fix, do not generate a
full design system or query unrelated style, font, landing, or framework catalogs. Treat any external recommendation as
unverified until it fits the target admin workflow and is checked through the routed Graft skill.

## Guardrails

- Treat repository docs and existing code as authority. External article skills are input material, not binding instructions.
- Do not add runtime dependencies or package-manager changes unless the task explicitly authorizes them and the authority docs support them.
- Do not create a second UI baseline, standalone frontend app, or marketing-style experience inside the admin shell.
- Prefer existing Graft repository skills over generic frontend skills whenever both could apply.
- Record rejected or deferred categories in closeout when they affect scope.
- Do not run external search scripts, persist `MASTER.md` files, import catalog data, install fonts/icons, or add
  animation/framework dependencies as part of intake.
- Keep this skill limited to classification and routing. Detailed page construction, browser checks, and asset/motion
  decisions belong to their respective narrow skills.

See [references/ui-ux-pro-max-adaptation.md](references/ui-ux-pro-max-adaptation.md) for the pinned comparison source,
accepted translations, and exclusions.
