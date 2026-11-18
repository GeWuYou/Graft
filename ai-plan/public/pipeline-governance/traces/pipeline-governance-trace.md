# Pipeline Governance Trace

## 2026-11-19 OpenAPI breaking-change gate corrective batch

- Review findings fixed: (1) removed operations can now be approved through the documented `deleted` sentinel in `openapi/approved-breaking.json`; the sentinel entry still binds base projection hash + method + path + change-kind and approves nothing while the operation still exists, so it cannot cover a modification. (2) Contract projections now include the path-item-level fields that apply to the reviewed operation (shared parameters, servers, description), so an unreviewed shared-parameter change invalidates approvals on that path; sibling operations still never enter each other's projections. (3) `scripts/openapi-breaking.mjs` accepts the optional base ref as a positional argument (as the Justfile help advertises) in addition to `--base`, rejecting a duplicated base.
- oasdiff emits no findings for weakening, changing, or removing shared path-level parameters on the tested Graft paths, so coverage for them comes from the projection hashes invalidating approvals, not from new oasdiff findings; recorded here so the boundary is explicit.
- Proof: deleting the `DELETE` operation of `/api/ops/applications/templates/{templateId}` failed the gate and passed with a sentinel ledger entry; a reviewed response delta with bound approvals passed, then the same ledger failed after an additional unreviewed shared-parameter change on the same path (approvals stale, unrelated paths still approved).

## 2026-11-19 OpenAPI breaking-change gate batch

- Added `scripts/openapi-breaking.mjs`: re-bundles both the base revision (materialized from the base commit's `openapi/` source graph because `openapi/dist/` is not tracked) and the current source through the same redocly path, runs `oasdiff breaking --flatten-allof --format json` via the Go tool directive in `server/go.mod`, and fails on ERR-level findings unless `openapi/approved-breaking.json` approves them. `--flatten-allof` is required because Graft composes responses through allOf envelopes; without it every envelope-internal change stays a level-2 warning with an `all-of-not-flattened` disclaimer.
- Approval ledger entries bind method, path, change kind, and SHA-256 projections of the reviewed operation plus its transitive local references on both sides; entries whose hashes no longer match are reported as stale and cannot approve. Ledger schema drift fails closed. `--mode report` keeps the gate non-blocking for local iteration.
- Added the blocking `OpenAPI Breaking Change Gate` job next to `Cross-Boundary Contract Freshness` and included it in `Required checks passed`.
- The canonical npm `oasdiff` name is a non-functional security placeholder and third-party npm wrappers redistribute unofficial binaries, so the tool dependency is vendored through the existing Go `tool` block instead of root `package.json`; this expands the manifest edit to `server/go.mod` and `server/go.sum`.
- Local validation passed: `node --test scripts/test-openapi-breaking.mjs`, `just openapi-check`, gate failure on a synthetic breaking change and pass on the unmodified tree, `actionlint`, `python3 scripts/validate_ai_plan_structure.py`, and `git diff --check`.

## 2026-08-01 Intake and first reliability batch

- Classified as long-running `refactor` work after GitHub Actions history showed 18 failures in the last 100 PR-validation runs.
- Kept `Build Artifacts` and `Web Check` out of implementation scope because their current optimization work is already complete or active.
- Added PR-only stale-run cancellation, non-web time limits, and canonical TruffleHog exclusions for two known fake update diagnostics fixtures.
- Added bounded PostgreSQL image-pull retry before disposable migration bootstrap and synchronized SSE test publication with the memory-hub subscription.
- Recorded the GitHub settings checklist as review-only; no GitHub repository API write is authorized in this topic.
- Local validation passed: `actionlint`, focused migration tests, repeated realtime tests, backend lint and build/test stages, and both AI Plan governance guards.

## Locked Decisions

- Preserve blocking CI gates and keep report-only checks optional in the repository ruleset.
- Treat a real GitHub Actions run as the required evidence for the observation batch.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": [
    "first-reliability-implementation"
  ],
  "pending_batches": [
    "github-actions-observation-and-settings-review"
  ],
  "current_batch": "github-actions-observation-and-settings-review",
  "next_batch": "github-actions-observation-and-settings-review",
  "closeout_status": "awaiting-external-observation"
}
```
