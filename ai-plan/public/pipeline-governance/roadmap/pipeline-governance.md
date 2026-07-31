# Pipeline Governance Roadmap

## Batch 1: Reliability Repairs

- Add PR-only cancellation and practical timeouts outside the excluded `Web Check` implementation.
- Canonicalize known fake TruffleHog fixture exclusions.
- Retry transient registry image pulls for migration bootstrap.
- Remove the verified SSE test publication race.

## Batch 2: GitHub Observation And Settings Review

- Inspect the first changed pull-request workflow for cancellation, timeout, secret-scan, migration, and server-test results.
- Review this repository-settings checklist without performing an API write:
  - Require `Secret Scan`.
  - Require `Contract Governance Check`.
  - Require `Cross-Boundary Contract Freshness`.
  - Require `Migration Governance / Migration Governance Gate`.
  - Require `Server Lint Stage`.
  - Require `Server Build/Test Stage`.
  - Retain `Web Check`.
  - Require `Analyze go` and `Analyze javascript-typescript` when code scanning reports them.
  - Do not require `Contract Drift Report`, `Server Lint Audit`, or CTRF report checks.
- Confirm administrator-applied ruleset changes do not weaken pull-request review, deletion, or non-fast-forward protections.

## Completion Evidence

- Local validation covers every changed authority.
- A GitHub Actions run on the changed workflow confirms the intended runtime behavior.
- Repository settings review records the maintainer decision separately from workflow code changes.
