# Upgrade Policy

本文件固定 `v0.1.0` release governance 的 upgrade safety authority。它定义 supported path、unsupported path、
operator responsibility boundary 和 compatibility principle，但不承诺自动化 preflight、自动迁移或自动回滚。

## Supported Upgrade Path

- upgrade from one official repository release tag to another official repository release tag
- keep `server` artifact, `web` artifact, and release notes aligned to the same target tag
- apply the manifest-declared explicit migration before normal runtime startup; the command is idempotent even when the target contains no schema change
- verify database backup and restore readiness before live migration
- preserve a pre-change config snapshot before governed change application

## Unsupported Or Discouraged Path

- mixed-tag `server` / `web` deployment
- using mutable `latest` or `beta` image tags as an upgrade target
- treating a Beta installation as a stable-production compatibility promise; Beta testing may use the matching immutable
  Beta tag and the beta channel selection rules when repeatability matters
- relying on `graft serve` for migration side effects
- skipping release notes, upgrade notes, or config review for releases with governed changes
- treating migration ordering numbers as product compatibility promises

## Upgrade Compatibility Principles

- forward-only migration governance is the only supported live schema evolution baseline in `v0.1.0`
- release tag is the only official product compatibility label
- config compatibility is governed by explicit change class, not by implicit legacy fallback
- rollback remains documentation-first and operator-controlled; a backup restore receipt never implies schema rollback

## Operator Responsibility Boundary

Operators are responsible for:

1. confirming source release and target release tag
2. reading release notes and upgrade notes before touching runtime state
3. verifying backup and restore readiness
4. preserving a pre-change config snapshot
5. running the manifest-declared explicit migration before normal startup
6. performing minimum post-upgrade verification and recording rollback decision points

## Related Authorities

- migration class and release boundary rules live in `migration-policy.md`
- config lifecycle and compatibility rules live in `config-policy.md`
- version and same-tag coordination rules live in `versioning-policy.md`
- machine-readable release identity and self-update execution boundaries live in `platform-self-update.md`
