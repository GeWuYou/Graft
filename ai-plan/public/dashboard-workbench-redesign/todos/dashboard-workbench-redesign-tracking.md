# Dashboard Workbench Redesign Tracking

## Topic

Dashboard Workbench Redesign

## Scope

Explore the existing dashboard authority chain, validate a fixed-slot presentation policy in a development-only preview, and stage later production contract evolution without replacing the formal homepage in the prototype batch.

## Repository Truth

- `AGENTS.md`
- `ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md`
- `web/AGENTS.md`
- `server/internal/dashboard/**`
- `openapi/components/schemas/dashboard-*.yaml`

## Work Contract

```yaml
version: 1
kind: feature
scope: long-running
authority_summary: Server modules own contribution facts, OpenAPI owns shared wire contracts, and the Web dashboard module owns presentation policy.
requires:
  design: true
  topic: true
  roadmap: true
  adr: false
execution:
  engine: direct-specialized-skill
  dispatch_skill: graft-web-vibe-coding
bootstrap:
  targets:
    - ai-plan/public/dashboard-workbench-redesign
    - ai-plan/design/architecture/首页工作台与Dashboard贡献设计.md
    - ai-plan/public/dashboard-workbench-redesign/roadmap/dashboard-workbench-redesign-roadmap.md
closeout:
  archive: true
  lessons_review: true
```

## Semantic Review Selection

- `graft-platform-architecture-review`: fixes presentation ownership above direct widget-card geometry.
- `graft-cross-boundary-review`: preserves server/OpenAPI fact authority while keeping the prototype Web-only.
- `graft-module-architecture-review`: keeps the presentation mapper private to the dashboard module.
- `graft-typescript-dx-audit`: governs the closed presentation/evidence unions and caller inference.
- `graft-test-seam-review`: tests the pure presentation model and development route seam.
- `graft-consistency-review` and `graft-delete-review`: avoid a parallel UI baseline, duplicate sorting authority, and speculative shared helpers.

## Current Recovery Point

- Authority discovery and status-semantics decisions are complete.
- Work Intake and design documents are bootstrapped.
- The deterministic preview, development route, locales, and focused tests are implemented.
- Browser evidence and one visual refinement pass are complete against clean primary-checkout HEAD `c07f5fa3`.
- Current next step: collect human visual acceptance before starting production homepage adoption.

## Task Checklist

- [x] Complete repository and runtime exploration.
- [x] Lock five-state presentation semantics and evidence semantics.
- [x] Implement preview model, page, locale copy, and development-only route.
- [x] Add focused model, route, and page tests.
- [x] Run focused Web validation and ai-plan structural validation.
- [ ] Clear or disposition the existing repository-wide typecheck blocker before claiming full `bun run check` completion.
- [x] Capture desktop and full-page browser artifacts; iterate at least once.
- [ ] Record human acceptance result before production adoption.
- [ ] Implement production Web adoption and later typed attention contract in separate batches.

## Validation Evidence

- Focused Vitest: 3 files, 11 tests passed.
- Scoped ESLint, i18n governance, stylelint/hygiene stages, `git diff --check`, and `validate_ai_plan_structure.py` passed.
- `bun run check` reaches the existing unrelated typecheck failure in `ResourceQueryPanel.vue`: the current HEAD does not export `AdvancedQueryFilterBuilderFrameState` from `AdvancedQueryFilterBuilderFrame.vue`.
- Independent read-only review found no blocker or P1. Its four P2 findings were resolved by adding an explicit presentation region, enforcing evidence invariants, using the shared `access-log` icon key, and removing the mobile orphan metric row.
- Browser inspection used the existing signed-in external browser after the reproducible WSL browser agent was blocked by missing Chromium system libraries. The real shell, status copy, quick-action Drawer, and fixed scenario were visible and interactive.
- Responsive checks at 1440, 1200, 992, and 768 viewport widths reported no horizontal overflow and preserved Attention → Health → Activity → Resources order.
- The visual refinement removed duplicate Attention icons and prevented the neutral Resources surface from stretching into a large empty card.
- Retained artifacts:
  - `.ai/artifacts/browser/dashboard-workbench-redesign/dashboard-redesign.png`
  - `.ai/artifacts/browser/dashboard-workbench-redesign/dashboard-redesign-full.png`

## Acceptance Conditions

- The preview is reachable only in development at `/mock/dashboard-preview`.
- Unknown, warning, info, healthy, and error remain semantically distinct.
- The fixed scenario contains no fabricated error state.
- The formal homepage and production contracts remain unchanged.
- Automated validation passes; browser artifacts remain inspection evidence rather than acceptance proof.

## Batch State

```json
{
  "completed_batches": [
    "authority-discovery-and-design",
    "development-preview-and-browser-evidence"
  ],
  "pending_batches": [
    "production-web-adoption",
    "typed-attention-contract"
  ],
  "current_batch": null,
  "next_batch": "production-web-adoption",
  "closeout_status": "preview-ready-for-acceptance"
}
```
