# Validation Delivery Tooling Trace

## 2026-11-20 Intake and PR 3 implementation batch

- Long-running `optimize` work, `docs/automation`; active topic only (no design, roadmap, or ADR). `kind: optimize` extends the ADR-003 v1 vocabulary; left for the ADR owner to reconcile.
- Evidence reuse, check suite, CI impact and workflow discipline implemented as summarized in the tracking checklist; full batch detail archived in `archive/trace-history-corrections.md` (commits 2edcdc3f6, 7902543de, 2e0cc9734, c621ce40c on base 270c2fb3c).

## 2026-11-20 Corrections 1-3 (cross-check hardening, GET pin, PR head SHA binding)

- Archived in `archive/trace-history-corrections.md` with commits and CI evidence; dispositions fixed. Commits: 56f7df274, ae83ae328.

## 2026-11-20 Browser acceptance harness batch (PR 4, worker track2-browser)

- Added `scripts/browser_acceptance.py` + unit tests (suite/engine/tier registry, zero retries, keep/reuse diagnostics, sanitized per-attempt reports, dry-run/list modes), `scripts/browser/serve_dist.py`, `tests/browser/` (tmpfs compose overlay, Playwright config, five tagged suite specs), `docs/development/browser-acceptance.md`, separated Justfile section, and test-only `TestSeedBrowserFixture` seeds in `server/modules/{project,task,runtime-target}` (no non-test server code touched). Full capability list lives in the docs and tracking checklist.
- Commits: c1501a2b5, 77612a329, 9c335138c, 1c3b193f5, 997d43575, 16fe0722b.

## 2026-11-20 Browser batch correction 1 (PR 378 CI: Secret Scan + 15 Greptile findings)

- All 15 Greptile findings verified real and fixed; none noise/stale. Proof anchors: `authenticateRequest` extracts bearer tokens only (authz.go:197); password-change routes sit behind authenticated+required/restricted guards (route_handlers.go:233,255); the success envelope nests totals under `data` (httpx/response.go:99); proxy header-pair misuse in `http.client.request`; `socket.create_connection` positional misuse; `response.read(n)` buffers until length/EOF (SSE); daemon suites checked an env var the harness never set.
- Secret Scan blocking finding reproduced with the CI trufflehog image: NOT the compose password but a literal invented DSN in the sanitization test fixture (c1501a2b5, scripts/test_browser_acceptance.py:245, raw_sha256 f70d79c0964d779078ce1d8890c855a879f79d59d13cb912b26a074455facf58). Fixed per synthetic-fixture policy: the current test composes the credential at runtime (eda72736d). The historical introduction commit still fails the base→head range scan; the lead owns the disposition — extend `scripts/scan_secrets.py`'s exact-finding exemption with that entry (same class as the existing synthetic finding) or force-push a rewritten branch. The current tree is clean.
- eda72736d: streaming dist proxy (read1, header mapping, websocket tuple), incremental environment state with compose down port restore and failure reporting, unique compose project per run, `--allow-daemon` forwarded to Playwright, comma-separated `--suite`, sanitized report paths, self-ignored report dir; 45 unit tests including ephemeral in-process proxy servers.
- 39ab2e5f5: auth setup sends the login access token as the required-change bearer, falls back between initial and acceptance credentials, seeds the localStorage session snapshot; specs read envelope `data.total`; single-element workbench assertion; fixture ports bound to 127.0.0.1.
- Validation: 45 unittest cases OK; `go vet`/`go build` and DB-free fixture tests OK; `just -n` OK; `git diff --check` OK; trufflehog on the corrected head reports only the historical finding above.

## Next Steps

- Observe a real pull-request run with the fixed cross-check after the lead pushes.
- Record chromium local-tier browser acceptance evidence from a primary-checkout run (docs/development/browser-acceptance.md).
- Lead disposes the historical Secret Scan finding (exemption entry or branch rewrite) and re-runs CI.
- Later batch: evaluate gating workflow jobs with `scripts/ci_impact.py` output (own scope and acceptance).
