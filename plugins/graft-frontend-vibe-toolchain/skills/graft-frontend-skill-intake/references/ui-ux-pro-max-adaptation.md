# UI/UX Pro Max Adaptation Reference

This is a project-local, independent summary of selected ideas from
`nextlevelbuilder/ui-ux-pro-max-skill`. It is not a vendored copy or a runtime dependency.

## Provenance

- Source: `https://github.com/nextlevelbuilder/ui-ux-pro-max-skill`
- Compared revision: `a38d04c3d5c298c851dbe5e6ee1965ee3de42cb5`
- License observed in the comparison checkout: MIT
- Local comparison checkout: `temp/ui-ux-pro-max-skill` (development-only, read-only)

The source's useful structure is intent routing plus on-demand reference material and a pre-delivery quality pass.
Graft keeps that structure by routing to existing narrow skills instead of creating a general-purpose UI/UX skill.

## Accepted Graft Translations

- **Intent-first selection:** distinguish new-page direction, focused UX/accessibility fixes, asset or motion choices,
  and runtime evidence before selecting a workflow.
- **Outcome-first checks:** verify observable outcomes such as named controls and focus order, state meaning beyond color,
  loading/error/empty recovery, stable text and layout, responsive overflow, and readable chart alternatives.
- **Context-aware design brief:** for new admin pages, capture operator context, product workflow, information density,
  page type, states, and existing theme/token constraints before composing components.
- **Constrained motion and assets:** motion communicates state, does not shift layout bounds, and has a reduced-motion path;
  icons and other assets come from existing Graft/TDesign-approved sources.
- **Evidence before claims:** a recommendation is guidance until the routed Graft implementation and browser QA establish
  that it fits the actual page and container sizes.

## Explicit Exclusions

- No upstream `search.py`, BM25 catalog, CSV/JSON datasets, generated catalogs, or persistence workflow.
- No generated `design-system/**/MASTER.md` or page override hierarchy; Graft's existing page-type and token authorities
  remain the only design baseline.
- No React, Next.js, Tailwind, shadcn, native/mobile, Uno Platform, or other stack-specific implementation guidance.
- No GSAP snippets, third-party fonts, icon packages, animation libraries, or package-manager changes.
- No landing-page, brand, slides, banner, or standalone-app workflows in the admin product.

When source advice conflicts with repository governance, discard the advice and follow Graft's existing skills and
`web/AGENTS.md`.
