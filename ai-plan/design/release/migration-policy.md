# Migration Policy

本文件固定 `v0.1.0` release governance 的 schema evolution authority。它定义 migration policy，不承诺自动化 rollback、
startup-time schema repair 或环境编排支持。

## Core Baseline

- live schema evolution is governed as forward-only migration application
- `graft migrate up` and `graft dev` are the explicit migration entrypoints
- `graft serve` remains pure runtime startup and must not become an implicit migration path
- any upgrade that may apply live migrations must verify database backup and restore capability first
- any governed live migration path must preserve the pre-change config snapshot needed for manual recovery
- rollback support remains documentation-first and operator-controlled

## Migration Change Classes

### Additive Migration

- adds schema elements without invalidating the currently supported data model
- examples include adding nullable columns, additive tables, or additive indexes that do not narrow compatibility
- acceptable in `patch` or `minor` when release notes stay honest and no destructive operator action is required

### Compatible Migration

- changes schema shape but keeps the governed operator path compatible when paired with explicit release notes, upgrade
  notes, and operator actions
- examples include controlled backfills, phased index changes, or shape changes that preserve the release-level upgrade
  path
- acceptable in `minor`
- `patch` should avoid compatible migrations that create rollout ambiguity, manual coordination burden, or unclear
  rollback decisions

### Destructive Migration

- removes schema elements or irreversibly rewrites state in a way that narrows rollback choices or operator tolerance
- never a normal `patch` release change
- only acceptable when release notes, upgrade notes, rollback decision points, and maintenance-window expectations are
  explicit
- any intentionally incompatible destructive evolution should default to `major` planning unless a narrower authority
  decision is documented first

## Version Boundary

### Patch

- may include `additive` migration only when the operator path remains straightforward and explicitly documented
- must not rely on destructive migration behavior
- must not hide schema implications behind runtime startup

### Minor

- may include `additive` or governed `compatible` migration
- may include bounded `destructive` migration only when release notes, upgrade notes, rollback decision points, and
  operator prerequisites are explicit

### Major

- is the default planning boundary for intentionally incompatible schema evolution
- still requires explicit operator guidance, maintenance expectations, and rollback risk framing

## Governance Notes

- migration file version identifiers are internal ordering values, not product versions and not compatibility labels
- migration policy must be interpreted together with `versioning-policy.md` and `upgrade-policy.md`
