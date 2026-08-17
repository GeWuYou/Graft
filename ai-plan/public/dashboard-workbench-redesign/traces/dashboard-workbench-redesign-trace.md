# Dashboard Workbench Redesign Trace

## 2026-08-17 Intake And Prototype Start

- Established the Work Contract, active topic, repository-wide design authority, and topic-local roadmap.
- Confirmed that current modules register structured widget metadata and payloads, not arbitrary Vue components.
- Confirmed three current homepage data surfaces: dashboard summary, container resource summary/realtime, and menu-derived quick actions.
- Confirmed TDesign MCP v0.2.4 access for the Vue Next components used by the preview and checked the installed 1.20.5 declarations.

## Locked Decisions

- The fixed scenario uses one warning, one unknown, two healthy dependencies, and neutral successful activity; it fabricates no error.
- Access-log source unavailability is a local observability warning, not a confirmed platform error.
- An outbound-network capability that has not run is unknown, not warning or error.
- The preview stays inside the real shell but is development-only and has an explicit deletion trigger.

## 2026-08-17 Preview Implementation

- Added a dashboard-owned presentation model with explicit `region`, `PresentationStatus`, and `EvidenceState` fields.
- Added deterministic design-validation facts without adapting through the legacy widget load/display metadata.
- Added the development-only real-shell route, four primary surfaces, quiet health rows, neutral resource no-sample copy, and a shared-icon quick-action Drawer.
- Added focused model, page, and release-route isolation tests; 3 files and 11 tests pass.
- Ran independent read-only change review. No blocker or P1 was found; all four P2 findings were repaired and revalidated.
- Full `bun run check` is currently blocked by a pre-existing export error in shared query-list code outside this topic's owned scope.

## 2026-08-17 Browser Inspection And Refinement

- Verified the development route in the real authenticated Graft shell at 1920 × 1080.
- Verified the “全部快捷操作” Drawer exposes all six shared-icon actions.
- Checked 1440, 1200, 992, and 768 viewport widths; none produced horizontal overflow, and narrow layouts retained the intended semantic order.
- Completed one screenshot-driven visual refinement: removed repeated status icons, kept health quiet, and stopped the missing-resource note from becoming an oversized empty surface.
- Captured and retained the required desktop and full-page PNG artifacts under `.ai/artifacts/browser/dashboard-workbench-redesign/`.

## 2026-08-17 Preview V2 Readability And Shell Refinement

- Preserved the approved Attention-first information architecture, quiet healthy semantics, warning/unknown separation, 8/4 composition, compact Resources, and bottom Quick Actions.
- Raised page, section, row, timestamp, and resource supporting copy to the readable body-large token while keeping verified healthy detail at body-medium.
- Reframed Operational Status as the overall answer “Needs Attention · 2 Items” and removed the repeated count from the concrete Attention section.
- Strengthened Attention at row level with one leading status identity, a three-pixel semantic rail, subtle row tint, stronger title, and explicit outline action; unknown remains neutral.
- Grouped Health and Resources into the supplemental right rail and converted Quick Actions into one compact action group with hover and focus feedback.
- Let authenticated development previews use the canonical session bootstrap so the formal sidebar and header remain present; anonymous development preview remains available for local inspection.
- Verified 1920 × 1080 and the 1440 / 1200 / 992 / 768 responsive widths with no horizontal overflow, then regenerated both required PNG artifacts.
- Passed the full `bun run check` entrypoint: 303 test files and 2091 tests plus the release build and governance stages.

## 2026-08-17 Quick Entry And Responsive Drawer Refinement

- Replaced the segmented Quick Actions frame with independent shared-icon action items. The home set is selected by `showOnHome` data and its responsive grid accepts any configured count without slicing to four.
- Reframed the Drawer as “全部入口”: a matching frequent section, search, and compact navigation lists grouped by the authorized shell's existing top-level sections.
- Kept arrows only for executable actions such as creating a build; ordinary page navigation remains arrowless and borderless until hover or keyboard focus.
- Reused the shared `ResponsiveDialog` workspace policy instead of a dashboard-local fixed Drawer width. Fixed its Teleport styling authority so narrow screens receive a true fullscreen surface attached to `body` without the mobile navigation crossing the overlay.
- Re-ran TDesign MCP protocol queries for Input docs and DOM, verified the locked 1.20.5 declarations, and used the documented `v-model`, `clearable`, `type=search`, and `prefixIcon` surface.
- Rechecked the live authenticated page at 1920 × 1080 and the entry workspace at 390 × 844, regenerated both PNG artifacts, and left the in-app browser on the open Drawer for annotation.
- Passed focused validation for 5 files and 34 tests, then passed the full `bun run check`: 304 test files and 2095 tests plus release build and all governance stages.

## 2026-08-17 Preview Copy Refinement

- Replaced design-rationale and validation-oriented helper text with concise operational product copy in the dashboard-owned Chinese and English locale catalogs.
- Kept “示例数据” only beside the data timestamp and removed mock, sorting, visual-weight, and design-validation explanations from the main page descriptions.
- Preserved the approved structure, layout, component hierarchy, visual styles, interaction, five-state semantics, and development-only route boundary.
- Passed the focused preview test and the full `bun run check`: 304 test files and 2095 tests plus release build and all governance stages.

## 2026-08-17 Production Web Adoption

- Extracted the accepted preview surface into `DashboardWorkbench` and made both preview and formal routes consume the same component.
- Added a pure production projector over validated alert, health, and timeline payloads. Loader failures remain local retryable warnings; legacy widget state and priority do not change presentation severity.
- Kept the existing formal homepage data authorities: Dashboard summary, focused widget refresh, authorized menu links with configured ranking/count, and the shared container resource snapshot/realtime manager.
- Distinguished container request failure, missing samples, confirmed samples, and missing permission without manufacturing resource anomalies.
- Repaired the access-log source authority so 4xx and slow requests are warning while 5xx remains error; no OpenAPI or Registry shape changed.
- Reused the shared responsive workspace for all entries. Browser inspection covered 1920 × 1080, 768 × 1024, and 390 × 844 without horizontal overflow; mobile remained fullscreen and searchable.
- Removed the zero-error phrase when no error exists and compacted only healthy row spacing after screenshot review, reducing the real-data column-height difference while preserving readable type.
- Retained the development preview for final formal-homepage acceptance and regenerated the required artifact paths.
- Passed the full Web completion entrypoint: 305 test files and 2099 tests, release build, type checks, formatting, lint, i18n, style, responsive, density, hygiene, pagination, and frontend OpenAPI governance.
- Passed `go test ./internal/httpx -count=1`; the full backend completion entrypoint remains environment-blocked because pinned `golangci-lint v2.12.2` is unavailable.
- Passed the ai-plan structure guard and `git diff --check` after recording the production recovery point.

## 2026-08-17 Formal Homepage Acceptance Diagnosis

- Reproduced two `widget-payload:*` attention rows for audit risk and access-request widgets; their loaders succeeded, but the Web guard rejected server-shaped key-first payloads without fallback `title`.
- Located the drift in the OpenAPI alert-list schema, which incorrectly requires both `title_key` and `title`; classified the repair as a backward-compatible response relaxation with generated consumer synchronization.
- Confirmed duplicate database/Redis health rows come from Monitor and CapabilityCoordinator. CapabilityCoordinator remains the platform health fact authority; Monitor will retain only active anomalies in its homepage contribution.
- Opened the formal homepage in the authorized in-app browser for annotation. Started a disjoint multi-agent batch for OpenAPI/Web and server-health implementation, with main-agent integration and validation ownership.

## 2026-08-17 Corrective Repair And Validation

- Made alert fallback `title` optional at the OpenAPI authority, documented `title_key` precedence, regenerated the bundle and Go/TypeScript projections, and taught the Web guard/projector to accept real key-first audit/access alerts.
- Removed Monitor's duplicate PostgreSQL/Redis homepage rows and derived its summary, state, and priority only from active anomalies.
- Kept PostgreSQL, Redis, and outbound-network status facts under CapabilityCoordinator with explicit key-first labels and state descriptions. Moved their server copy into core display locales and added a physical-ownership regression test.
- Passed the OpenAPI freshness recipe, `cd server && go run ./cmd/graft validate backend`, and `cd web && bun run check` with 305 test files and 2101 tests plus the release build and all governance stages.
- Passed ai-plan structure validation and independent semantic review with no remaining blocking, high, or medium findings. Existing OpenAPI 3.1/oapi-codegen and DTO-boundary warnings remain repository-known and introduced no violation.
- Kept the authorized in-app browser open and handed off for annotation. Its local-address security policy rejected automated refresh, so final visual inspection remains an explicit human acceptance step rather than inferred proof.

## 2026-08-17 PR Review Cleanup

- Kept the explicit backend validation command because server execution truth does not assume a prebuilt `graft`
  binary.
- Separated contextual drill-down from quick-entry activation so Attention and Resources navigation no longer changes
  the browser-local quick-entry rank.
- Replaced ISO text ordering with instant ordering, renamed the attention-only count map, and added explicit unknown
  evidence, offset timestamp, accessibility, navigation-ranking, and typed-fixture regressions.
- Isolated the fixed preview scenario in `presentation/preview-workbench.ts` and its visible copy in
  `dashboard.previewWorkbench.*`; the roadmap now deletes both with the preview route after human acceptance.
- Reused `dashboard.actions.details` as the established server/Web action key and removed the duplicate
  `dashboard.actions.viewDetails` locale entry.
- Focused Dashboard Vitest (6 files, 33 tests), TypeScript typecheck, and strict i18n governance passed; the final
  integrated `bun run check` also passed with 306 test files and 2107 tests plus the release build and all governance
  stages.

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
