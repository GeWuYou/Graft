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

## Operator-Visible Docs Status

- operator-facing config reference docs are deferred in the current phase
- until those docs are intentionally authored, this file is the release-governance authority for config lifecycle
