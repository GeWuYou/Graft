# Config Policy

本文件固定 `v0.1.0` release governance 的 stable configuration lifecycle authority。它定义长期可维护的兼容性治理口径，
不伪装成 alias bridge、startup warning 或 rewrite helper 已经存在。

## Stable Config Change Classes

- `additive`
- `default-change`
- `rename`
- `semantic-change`
- `removal`

## Default Value Principle

- new stable config should prefer safe, least-surprising, documentable defaults
- when no reasonable default exists, the config must be documented as required input rather than assumed through hidden
  environment behavior
- any `default-change` must record the behavior impact and whether operator override is expected

## Patch Boundary

- may introduce `additive`
- may introduce low-risk `default-change` only when behavior impact is explicitly documented
- must not silently introduce `rename`, `semantic-change`, or `removal`

## Minor Boundary

- may introduce `rename`, `semantic-change`, or `removal` only with release notes and upgrade notes
- must record:
  - canonical owner
  - deprecated_in
  - removal_target
  - replacement
  - operator action required

## Major Boundary

- may carry intentionally incompatible config lifecycle changes as a planned release boundary
- still requires explicit replacement guidance and operator action framing

## Deprecation Record Baseline

For every governed `rename`, `semantic-change`, or `removal`, keep at least:

- config key or config group
- canonical owner
- change class
- deprecated_in
- removal_target
- replacement
- operator action required
- release-notes required
- upgrade-notes required

## Rename And Removal Compatibility

- `v0.1.0` does not assume startup deprecation warnings, config alias bridges, or config rewrite helpers
- when a rename is governed, the new key is canonical and the migration path is manual and documentation-driven
- removal must not be described as safe by default; documents must state when the old key stops being accepted

## Release Packaging Config Boundary

- release-grade binaries must keep deployment-specific runtime config external; packaging a binary does not authorize
  hard-coding environment-specific operator values into the artifact
- `graft validate release` minimum guarantees do not imply full config completeness validation, operator secret
  validation, or environment-preflight success
- publish-time environment variables, workflow inputs, asset filenames, or GitHub Release metadata are release
  automation inputs, not stable runtime config authority
- external release assets such as `LICENSE`, `SBOM`, license compliance report, and `web` dist may evolve in packaging
  shape without being reclassified as runtime config keys

## Documentation-First Release Config Evolution

- if future release packaging introduces sample config bundles, environment manifests, or installer-generated config,
  the stable operator-facing config authority must still be documented here or in a sibling release authority doc
  before automation is treated as canonical behavior
- do not treat publish workflow defaults, branch-local scripts, or ad-hoc release notes as the sole record of config
  compatibility policy

## Operator-Visible Docs Status

- operator-facing config reference docs are deferred in the current phase
- until those docs are intentionally authored, this file is the release-governance authority for config lifecycle
