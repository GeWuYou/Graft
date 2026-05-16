# Contract Governance Scanner

This directory contains the repository-local contract governance scanner used by
hooks and CI to detect high-risk magic values and contract drift candidates.

## Entrypoint

```bash
scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode changed
scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci
scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode report
```

## Modes

- `changed`
  - scans staged files when available, otherwise scans files changed against
    `HEAD`
  - intended for `pre-commit`; CI remains the authoritative remote blocking
    entrypoint
- `ci`
  - scans repository-tracked files, applies baseline / allowlist, and fails on
    blocking findings
  - intended for the blocking CI job
- `report`
  - scans repository-tracked files and prints a non-blocking governance report,
    including duplicate-string candidates and drift candidates

## Metadata Files

- `baseline.json`
  - historical findings temporarily tolerated while the repository is being
    cleaned up
- `allowlist.json`
  - time-bounded exceptions with explicit ownership and cleanup metadata

Both files require:

- `id`
- `path`
- `rule`
- `severity`
- `owner`
- `reason`
- `created_at`
- `cleanup_phase`
- `expire_at`

## Severity

- `P0`
  - blocking in hooks and CI
- `P1`
  - warning in local hooks, blocking for new findings in CI
- `P2`
  - warning only
- `P3`
  - report only

## Scope Defaults

The scanner intentionally focuses on contract-sensitive surfaces first:

- headers
- storage keys
- API error codes
- permission codes
- event names
- route names / special route paths
- API paths
- message keys

Generated artifacts, third-party code, and build output are skipped by default.

## Local Hook Usage

- `.husky/pre-commit`
  - runs `lint-staged` first, then executes the `changed` scan through
    `scripts/run_python.sh`
- `.husky/pre-push`
  - does not rerun the local changed-file scan; push-time blocking is owned by
    CI
