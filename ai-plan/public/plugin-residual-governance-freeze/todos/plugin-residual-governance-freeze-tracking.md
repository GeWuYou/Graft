# Plugin Residual Governance Freeze Tracking

## Topic

- Topic: `plugin-residual-governance-freeze`
- Status: `active`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
- Parent topic: `server-module-semantics-and-live-migration-reset`

## Owned Scope

- `AGENTS.md`
- `server/AGENTS.md`
- `web/AGENTS.md`
- active `ai-plan/design/**`
- active `ai-plan/roadmap/**`
- active `ai-plan/public/**`
- `scripts/plugin_residual/**`

## Exclusions

- `ai-plan/public/archive/**`
- third-party dependency payload files such as lockfiles and package manifests
- live wire/domain rename work outside residual classification

## Frozen Decisions

- “彻底” in this topic means current-authority cleanup plus residual governance freeze
- archive topics and archived topic names remain read-only historical evidence
- intentional domain/wire `plugin` literals are allowed only when explicitly allowlisted
- new uncategorized non-archive hits are blocking drift

## Residual Classes

- `history_or_topic_name`
- `historical_governance`
- `intentional_domain_or_wire`
- `third_party`

## Batch Record

- Batch 1:
  - opened the new active topic
  - corrected current-authority wording drift in active governance/design/roadmap docs
  - added `scripts/plugin_residual/check_plugin_residuals.py`
  - added `scripts/plugin_residual/allowlist.json`
  - added checker regression tests

## Validation Record

- `python3 scripts/plugin_residual/test_check_plugin_residuals.py`
- `python3 scripts/plugin_residual/check_plugin_residuals.py`
- `git diff --check`

## Notes

- this topic does not claim that every literal `plugin` in active non-archive files is wrong
- it claims that every remaining non-archive hit must now be intentionally categorized
