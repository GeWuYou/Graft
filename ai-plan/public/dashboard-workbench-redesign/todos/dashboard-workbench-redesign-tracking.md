# Dashboard Workbench Redesign Tracking

## Topic

Dashboard Workbench Redesign

## Scope

Explore the existing dashboard authority chain, validate a fixed-slot presentation policy, adopt it on the formal homepage, and stage later typed contract evolution without guessing severity from widget IDs or display metadata.

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
- `graft-openapi-contract-review` and `graft-api-dx-review`: correct alert fallback optionality at the wire authority without a compatibility DTO.
- `graft-localization-governance`: keep platform capability display key-first and owner-local in both locale catalogs.
- `graft-cache-governance`: classify this live status read as `no cache`; the defect is contract/authority drift rather than read amplification.

## Current Recovery Point

- Authority discovery and status-semantics decisions are complete.
- Work Intake and design documents are bootstrapped.
- The deterministic preview, development route, locales, and focused tests are implemented.
- Browser evidence and two visual refinement passes are complete; preview v2 now reuses authenticated bootstrap navigation in the real shell and exposes a responsive searchable “全部入口” workspace.
- Preview copy now uses concise production-console language; design rationale and validation disclaimers were removed from user-facing descriptions, with “示例数据” retained only beside the data time.
- Production `/` now renders the shared workbench over the existing summary response; the preview remains available for final comparison and deletion approval.
- Formal-homepage acceptance reproduced invalid-payload warnings for key-first audit/access items and duplicate Monitor/CapabilityCoordinator PostgreSQL/Redis rows.
- The corrective batch has synchronized OpenAPI/Web alert optionality, removed duplicate health authority, localized CapabilityCoordinator facts, and passed full cross-boundary validation.
- PR review cleanup now keeps contextual Attention/Resources navigation out of quick-entry ranking, sorts activity by
  parsed instants, names attention-only counts explicitly, removes test type escapes, and isolates the fixed preview
  scenario plus copy behind one deletion boundary.
- The latest PR review inventory is fully reconciled: the portable GitHub MCP launcher, exclusive workbench action
  variants, and exact terminal resize-send regression are repaired; previously addressed threads, the earlier Monaco
  timeout evidence, and the external docstring-coverage target are classified from current repository evidence.
- A follow-up PR review found no new runtime or contract defect; it corrected one compound modifier in this recovery
  record, then gave the CI-sensitive Monaco KeepAlive lifecycle test a local 20-second budget after the same five-second
  timeout reproduced remotely while focused and joint local repetitions remained green.
- Automated refresh of the already-open in-app browser was blocked by its local-address security policy, so formal-homepage visual acceptance remains human-owned.

## Task Checklist

- [x] Complete repository and runtime exploration.
- [x] Lock five-state presentation semantics and evidence semantics.
- [x] Implement preview model, page, locale copy, and development-only route.
- [x] Add focused model, route, and page tests.
- [x] Run focused Web validation and ai-plan structural validation.
- [x] Pass the full repository Web validation entrypoint.
- [x] Capture desktop and full-page browser artifacts; iterate at least once.
- [x] Complete the second readability, Operational Status, Attention, Quick Actions, and right-rail refinement round.
- [x] Record human acceptance of the preview before production adoption.
- [x] Implement production Web adoption over the existing typed payloads.
- [x] Repair access-log 4xx/5xx severity at the server fact authority without changing OpenAPI shape.
- [x] Make alert fallback `title` optional at OpenAPI authority and synchronize generated/Web consumers.
- [x] Remove Monitor's duplicate database/Redis homepage projection and localize CapabilityCoordinator rows.
- [x] Pass focused and full cross-boundary validation.
- [x] Resolve the bounded Dashboard/topic PR review findings without changing TDesign component usage or wire contracts.
- [ ] Manually refresh the open formal homepage, repeat browser inspection, and record human acceptance.
- [ ] Record human acceptance of the formal homepage and delete the preview.
- [ ] Implement the additive typed attention contract in a later batch.

## Validation Evidence

- Focused Vitest for the second round: 4 files, 20 tests passed.
- Full `bun run check` after production adoption passed: 305 test files and 2099 tests, plus typecheck, TS7 check, release build, format, lint, i18n, style, density, hygiene, responsive, pagination, and OpenAPI frontend governance stages.
- Focused backend package validation passed: `go test ./internal/httpx -count=1`, including the 4xx/5xx/slow severity assertions. The repository backend completion entrypoint could not finish because the environment does not contain the pinned `golangci-lint v2.12.2` binary.
- `python3 scripts/validate_ai_plan_structure.py` and `git diff --check` passed after the recovery documents were updated.
- Independent read-only review found no blocker or P1. Its four P2 findings were resolved by adding an explicit presentation region, enforcing evidence invariants, using the shared `access-log` icon key, and removing the mobile orphan metric row.
- Browser inspection used the existing signed-in external browser after the reproducible WSL browser agent was blocked by missing Chromium system libraries. The real shell, status copy, quick-action Drawer, and fixed scenario were visible and interactive.
- Responsive checks at 1440, 1200, 992, and 768 viewport widths reported no horizontal overflow and preserved Attention → Health → Activity → Resources order.
- Preview v2 uses body-large typography for operational secondary information, keeps healthy detail quiet, moves status identity to the leading edge of each Attention row, and removes the repeated section count.
- Authenticated preview navigation now follows the canonical bootstrap path so its sidebar, header, and content shell match the formal homepage; anonymous development preview access remains available.
- Health and compact Resources now form a right-side supplemental rail. Home Quick Actions are independent icon action items selected by presentation data rather than a fixed four-item slice or one segmented outer frame.
- “全部入口” derives its groups and entries from the authorized shell menu, keeps the configured home actions in a matching frequent section, and filters all entries without inventing a parallel IA.
- The Drawer uses the shared `ResponsiveDialog` workspace policy: the project compact desktop Drawer token at wide widths and a true fullscreen surface below the compact breakpoint. Browser checks confirmed the fullscreen header, scrolling content, and overlay stacking at 390 × 844.
- Copy-only refinement updated the dashboard-owned Chinese and English locale catalogs without changing structure, layout, component hierarchy, styles, or interaction. The refreshed preview copy passed the focused preview test and the full Web validation entrypoint.
- Production adoption extracts one shared `DashboardWorkbench` surface used by both formal and preview routes. The production container retains summary, focused widget retry, quick-action configuration/ranking, container resource state, and realtime ownership.
- The pure production projector reads only validated widget type payloads: alert `level`, health `status`, and timeline `status`. Loader failure becomes `warning + source-failed + retry`; widget state, priority, ID, and title never override severity.
- Quick Actions use all authorized menu leaves, existing usage ranking, and configured `maxItems`; browser checks confirmed six-item projection tests and the full responsive “全部入口” navigation are independent.
- Access-log authority now emits 4xx warning, 5xx error, and slow warning; the warning-only case keeps widget state and priority at warning.
- Browser inspection of formal `/` covered 1920 × 1080, 768 × 1024, and 390 × 844. No horizontal overflow occurred; the 390 workspace is fullscreen, and compact healthy rows reduced real-data column-height imbalance without reducing typography.
- The corrective batch passed all four OpenAPI freshness checks, the full backend completion entrypoint, and the full Web completion entrypoint with 305 test files and 2101 tests plus the release build and governance stages.
- Independent read-only review found no remaining blocking, high, or medium issue after OpenAPI fallback semantics and core display-locale ownership were clarified.
- The in-app browser tab remains open and handed off for annotation. Browser security policy blocked automated refresh of the local address, so the corrected runtime view is intentionally not claimed as automated acceptance evidence.
- PR review cleanup passed 6 focused Dashboard Vitest files with 33 tests, `bun run typecheck`, and
  `bun run lint:i18n`; it also kept the documented backend validation entry unchanged because
  `cd server && go run ./cmd/graft validate backend` is the explicit repository entry when no prebuilt binary is
  assumed.
- The integrated PR review cleanup passed the full `bun run check` entrypoint with 306 test files and 2107 tests,
  release build, type checks, formatting, lint, i18n, style, pagination, and hygiene governance.
- The latest PR review repair passed 45 focused AI-governance helper tests, both AI governance structure validators,
  Dashboard focused Vitest (3 files, 26 tests), WebTerminal/Monaco-focused Vitest (2 files, 20 tests), repeated Monaco
  isolation and pressure runs, and TypeScript typecheck. The integrated `bun run check` then passed with 306 test
  files and 2109 tests plus the release build and every frontend governance stage.
- The follow-up Monaco diagnosis passed 20 isolated lifecycle repetitions and 10 joint Monaco/WebTerminal repetitions;
  the single-test body completed in about 1.4-1.8 seconds locally while Monaco import and CSS evaluation dominated the
  process wall time, supporting a test-local timeout repair instead of a global runner change.
- Retained artifacts:
  - `.ai/artifacts/browser/dashboard-workbench-redesign/dashboard-redesign.png`
  - `.ai/artifacts/browser/dashboard-workbench-redesign/dashboard-redesign-full.png`

## Acceptance Conditions

- The preview is reachable only in development at `/mock/dashboard-preview`.
- Unknown, warning, info, healthy, and error remain semantically distinct.
- The fixed scenario contains no fabricated error state.
- The formal homepage uses the accepted workbench; the Dashboard Registry shape remains unchanged, while OpenAPI correctly treats key-first alert fallback `title` as optional.
- Automated validation passes; browser artifacts remain inspection evidence rather than acceptance proof.

## Batch State

```json
{
  "completed_batches": [
    "authority-discovery-and-design",
    "development-preview-and-browser-evidence",
    "production-web-adoption",
    "homepage-acceptance-corrective-repair"
  ],
  "pending_batches": [
    "formal-homepage-browser-acceptance",
    "preview-removal-after-acceptance",
    "typed-attention-contract"
  ],
  "current_batch": "formal-homepage-browser-acceptance",
  "next_batch": "formal-homepage-browser-acceptance",
  "closeout_status": "awaiting-human-visual-acceptance"
}
```
