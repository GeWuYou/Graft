# Plugin Residual Governance Freeze

## Status

- Topic: `plugin-residual-governance-freeze`
- Status: `active`
- Task class: `cross-boundary`
- Recovery source: `parent topic`
  - `server-module-semantics-and-live-migration-reset`

## Startup Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `parent topic`
- authority summary:
  - current runtime and frontend authority remain `server/modules/**`, `server/internal/moduleapi/**`, `web/src/modules/**`, and active governance docs
  - archive topics and old topic names remain historical evidence only
  - intentionally retained `plugin` literals are allowed only when explicitly classified as historical wording, rename-history evidence, intentional domain/wire semantics, or third-party/framework vocabulary

## Goal

Freeze the accepted non-archive `plugin / Plugin / server/plugins` residual set and make it auditable.

This topic must:

1. remove any remaining current-authority wording drift that still uses `plugin` where `module` is canonical
2. classify all remaining non-archive hits into explicit residual categories
3. add a repository-local checker and allowlist so new drift fails closed
4. keep archive materials and archived topic names untouched

This topic must not:

- reopen archive topics
- rename intentional wire/domain literals unless a separate authority topic approves it
- introduce compatibility aliases or fallback naming

## Residual Classes

- `history_or_topic_name`
  - active recovery docs that must reference historical topic names, rename maps, preserved worktree names, or prior accepted slices
- `historical_governance`
  - explicit “historical plugin naming” explanations and anti-goal wording such as `runtime plugin platform`
- `intentional_domain_or_wire`
  - retained literal values or design examples whose current approved contract or workflow wording still uses `plugin`
- `third_party`
  - dependency names, framework API keys, Vue Test Utils `plugins` mount options, and similar non-repository-owned vocabulary

## Scope

- included:
  - `AGENTS.md`
  - `server/AGENTS.md`
  - `web/AGENTS.md`
  - active `ai-plan/design/**`
  - active `ai-plan/roadmap/**`
  - active `ai-plan/public/**`
  - `scripts/plugin_residual/**`
- excluded:
  - `ai-plan/public/archive/**`
  - dependency lockfiles and package manifests except as allowlist exclusions
  - runtime/public contract renames outside this residual-governance purpose

## Validation Chain

- `python3 scripts/plugin_residual/test_check_plugin_residuals.py`
- `python3 scripts/plugin_residual/check_plugin_residuals.py`
- `git diff --check`
- add server/web validation only if live `server/**` or `web/src/**` runtime code changes land

## Current Residual Matrix

- active governance and design docs:
  - keep explicit historical naming and anti-goal wording
- active recovery docs:
  - keep closed topic names, rename-history evidence, and accepted residual summaries
- monitor and logging governance docs:
  - keep explicitly approved `plugin` wire/domain examples until a separate authority topic says otherwise
- frontend tests and dependency references:
  - keep third-party `plugins` API vocabulary excluded from repository-owned drift

## Next-Session Prompt

`Re-run startup preflight from root AGENTS.md. Governance source: root AGENTS.md. Task class: cross-boundary. Recovery source: parent topic server-module-semantics-and-live-migration-reset. Owned scope: AGENTS.md, server/AGENTS.md, web/AGENTS.md, active ai-plan/design/**, active ai-plan/roadmap/**, active ai-plan/public/**, scripts/plugin_residual/**, and only the minimum additional live authority files required by uncategorized residual cleanup. Treat archive topics as read-only evidence.`
