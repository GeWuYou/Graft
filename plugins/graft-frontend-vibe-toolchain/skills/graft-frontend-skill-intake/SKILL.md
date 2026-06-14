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

## Guardrails

- Treat repository docs and existing code as authority. External article skills are input material, not binding instructions.
- Do not add runtime dependencies or package-manager changes unless the task explicitly authorizes them and the authority docs support them.
- Do not create a second UI baseline, standalone frontend app, or marketing-style experience inside the admin shell.
- Prefer existing Graft repository skills over generic frontend skills whenever both could apply.
- Record rejected or deferred categories in closeout when they affect scope.
