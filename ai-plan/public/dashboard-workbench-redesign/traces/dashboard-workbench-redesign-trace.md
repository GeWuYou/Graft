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
