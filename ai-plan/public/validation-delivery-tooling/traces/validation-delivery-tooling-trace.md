# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only (no design, roadmap, or ADR). `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner to reconcile.
- Evidence reuse: plans bind head SHA and SHA-256 digests with per-file reasons and coverage; plan mode fails on empty selection, missing tests, or coverage gaps; `--compare-report` verifies previous reports fail-closed; `--report-out` persists executed reports; `just check-local-verify` verifies then executes.
- Check suite: six groups from Justfile/workflow reality; cross-group `requires=` with auto-include and cycle/unknown rejection; independent failures; process-level deadline with process-group kill; JSON + terminal summary; self-ignored logs under `.artifacts/checks/<group>/`.
- CI impact: conservative path-to-job mapping; unknown/workflow/script/toolchain/image surfaces force FULL; subsets always include hygiene and summary jobs. Advisory only.
- Workflow: grouped jobs; `web-check` timeout; `Required checks passed` needs every other job and verifies every job of the current run/attempt/SHA via the Actions API (push-event Secret Scan skip only); job names unchanged; push runs never cancelled.
- Commits (base 270c2fb3c): 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c. The lead pushes and integrates.

## 2026-11-20 Corrections 1-3 (cross-check hardening, GET pin, PR head SHA binding)

- Corrections 1-2 (gh api hardening, GET pin) and correction 3 (bind the job SHA cross-check to `github.event.pull_request.head.sha || github.sha`) are archived in `archive/trace-history-corrections.md` with commits, CI run evidence and validation.
- Dispositions: fixed. Final commits: 56f7df274, ae83ae328.

## 2026-11-20 Browser acceptance harness batch (PR 4, worker track2-browser)

- Added `scripts/browser_acceptance.py` + `scripts/test_browser_acceptance.py`: suite registry (workbench/task/runtime-target; docker/compose daemon-required, explicit `--allow-daemon` only), chromium baseline plus firefox/webkit `@cross-browser` matrix enforced against spec-file tag markers, `--tier local|ci` (ci forbids keep/reuse diagnostics and defaults its own report dir), zero retries by design, `--keep`/`--reuse-environment`/`--teardown-only` environment diagnostics, per-attempt sanitized JSON reports (repo/home paths compressed, DSN passwords, bearer and query secrets redacted), `--dry-run`/`--list-suites`/`--list-engines` work in any worktree.
- Added `scripts/browser/serve_dist.py` (production-bundle server mirroring the graft-web proxy surface with websocket tunneling), `tests/browser/` (compose overlay with per-attempt tmpfs PostgreSQL/Redis, playwright.config.ts with retries=0 and grepInvert, package.json, five tagged suite specs), `docs/development/browser-acceptance.md` and a separated Justfile section.
- Seeds: test-only `TestSeedBrowserFixture` in `server/modules/{project,task,runtime-target}`, guarded by `GRAFT_BROWSER_SEED_DATABASE_URL` (skip when unset). Project seed reuses the module's own `ImportApplication` write path; task seed writes terminal `application.compose.*` history for the shared fixture application id; runtime-target seed avoids the local Docker probe endpoint. No non-test server code touched.
- Validation: 33 harness unittest cases OK; `go vet`/`go build` and DB-free fixture tests on the three touched packages OK; `just -n` recipe parse OK. Browser execution is out of scope in leased worktrees: the lead runs the documented primary-checkout sequence in `docs/development/browser-acceptance.md` after merge and records the chromium local-tier evidence.

## Next Steps

- Observe a real pull-request run with the fixed cross-check after the lead pushes.
- Record chromium local-tier browser acceptance evidence from a primary-checkout run (see docs/development/browser-acceptance.md).
- Later batch: evaluate gating workflow jobs with `scripts/ci_impact.py` output; that changes CI behavior and needs its own scope and acceptance.
