# Versioning Policy

本文件固定 `v0.1.0` release governance 的 SemVer and compatibility authority。它定义 release versioning 开发约束，
不声明额外 release workflow 或 runtime support 已实现。

## SemVer Baseline

- repository releases use semantic versioning as the operator-facing compatibility language
- the canonical official release identity is the repository Git tag `vMAJOR.MINOR.PATCH`
- `server` binary, `web` dist artifact, and release notes for an official release must come from the same release tag

## Release Type Boundaries

### Patch

- for bug fixes and safe additive adjustments
- must not silently rename, remove, or reinterpret stable config keys
- must not require a destructive migration path
- must not rely on unsupported mixed-tag deployment assumptions

### Minor

- may add features and bounded compatibility-managed changes
- any governed `rename`, `semantic-change`, `removal`, or destructive operator action must be documented in release
  notes and upgrade notes
- must preserve one coherent same-tag release package across `server`, `web`, and release notes

### Major

- default planning boundary for intentionally incompatible release governance changes
- still requires explicit migration, rollback, and operator guidance

## Breaking Change Criteria

A release change should be treated as breaking when it requires one or more of the following outside the supported
baseline:

- mixed-tag `server` and `web` deployment
- destructive schema action without explicit governed path
- incompatible stable config reinterpretation
- operator intervention beyond ordinary patch expectations without documented release guidance

## Version Coordination

- official `server` artifact, `web` artifact, and release notes must share the same repository tag and commit lineage
- mixing `server` and `web` artifacts from different official release tags is unsupported in `v0.1.0`
- migration version identifiers are internal ordering numbers only; they must not be used as product versions or
  compatibility labels
- frontend toolchain dependency versions and build dependency versions are build inputs, not product release identity

## Support Cadence Boundary

- `v0.1.0` promises one active repository release line at a time
- `v0.1.0` does not promise LTS, multi-minor parallel support, or independent `server` / `web` official release
  cadence
