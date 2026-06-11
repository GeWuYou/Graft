---
name: graft-shared-asset-reuse
description: Use before adding, moving, renaming, removing, or replacing reusable Graft frontend/backend/cross-boundary assets so agents discover existing shared assets and maintain the curated shared asset registries without turning them into source inventories.
---

# Graft Shared Asset Reuse

Use this skill when a task may add, move, rename, remove, replace, or duplicate reusable assets in `web`, `server`,
OpenAPI, validation scripts, or repository skills.

Treat root `AGENTS.md` and `ai-plan/design/共享资产复用治理规范.md` as the governance sources. This skill does not replace
startup, validation, commit, closeout, OpenAPI, web, server, or lessons-learned governance.

## 1. Purpose

This skill defines the workflow for:

- discovering existing curated shared assets before adding new code
- deciding whether a changed asset belongs in the registry
- keeping registry entries small and maintainable
- reporting registry additions, removals, replacements, or rejected reuse candidates in closeout

The shared asset registry is a curated governance index, not a fully auto-generated source tree inventory.

## 2. When To Use

Use this skill when a task touches or proposes reusable assets under:

- `web/src/shared/**`
- `web/src/components/**`
- `web/src/composables/**`
- `web/src/utils/**`
- `web/src/modules/*/shared/**`
- `web/src/modules/*/presenter/**`
- `server/internal/**`
- `server/internal/moduleapi/**`
- `server/modules/*/moduleapi/**`
- `openapi/**`
- `scripts/validate_*.py`
- `.agents/skills/**`

Also use it before adding a new helper, component, composable, presenter, mapper, service, repository helper, route
builder, logger helper, config registry helper, scheduler helper, migration validator, OpenAPI mapper, or common UI
pattern.

## 3. When Not To Use

Do not use this skill for:

- pure copy or style edits with no reusable asset impact
- module-private business logic that clearly stays within one module
- generated artifact refreshes where the authority source did not change
- unrelated runtime bugfixes that do not add or alter shared-capability surfaces

If not applicable, closeout still reports `shared_asset_preflight.status: not_applicable` with a short reason.

## 4. Required Reads

After normal startup preflight, read:

- `ai-plan/design/共享资产复用治理规范.md`
- the relevant registry files:
  - `.ai/registries/web-shared-assets.yaml`
  - `.ai/registries/server-shared-assets.yaml`
  - `.ai/registries/cross-boundary-assets.yaml`
- `web/AGENTS.md` for frontend work
- `server/AGENTS.md` for backend work
- `ai-plan/design/契约治理与魔法值治理规范.md` for stable contracts or cross-boundary values

Read only the registries relevant to the current class when scope is narrow; read all three for cross-boundary or
governance changes.

## 5. Reuse Preflight

Before adding new reusable code:

1. Search the relevant registry by `type`, `purpose`, and `use_when`.
2. Use `rg` against real source paths for similar helper/component/method names and repeated implementations.
3. Classify each candidate as shared-owned, module-owned, contract-owned, generated-derived, or not reusable.
4. Reuse existing assets when the owner and `use_when` match.
5. Reject reuse only with a concrete reason, such as module-private semantics, authority mismatch, or generated-only source.
6. If adding a new shared asset, verify it meets at least one registry entry criterion before adding a registry entry.
7. If a changed registered path disappeared, determine whether it moved, was renamed, replaced, or intentionally removed.

Do not promote module-private business logic into `shared/**` only to satisfy reuse pressure.

## 6. Registry Entry Criteria

Add a registry entry only when the asset is one of:

- reused by at least two modules
- owned by a stable shared boundary
- a common duplicate-risk capability for future agents
- a cross-boundary asset such as OpenAPI contracts, i18n key-first presentation, moduleapi, scheduler registry,
  configregistry, logger, httpx, migration validation, or route/menu/permission contract

Do not register every helper, component, or module-private file.

## 7. Removal And Rename Rule

If a registered asset path disappears, do not silently delete the entry.

Determine whether the asset was:

- moved
- renamed
- replaced by another asset
- intentionally removed

When removing or replacing a registry entry, include `removed_reason` or `replaced_by` in closeout.

## 8. Validation

For registry or skill changes, run:

```bash
python3 scripts/validate_shared_asset_registries.py
python3 scripts/validate_ai_governance.py
```

If the validator script or tests changed, also run:

```bash
python3 -m unittest discover -s scripts -p 'test_*.py'
```

Run `cd web && bun run check` or `cd server && go run ./cmd/graft validate backend --stage lint` only when the task
actually changes corresponding runtime, contract, or validation semantics.

## 9. Closeout

Every applicable task must report:

```text
shared_asset_preflight:
- status: used | not_applicable
- registries_checked:
- assets_reused:
- assets_considered_but_rejected:
- new_registry_entries:
- registry_entries_removed_or_replaced:
- validation_commands:
```

For `registry_entries_removed_or_replaced`, include:

```text
- name: <asset-name>
  removed_reason: <why removed, if removed>
  replaced_by: <replacement name/path, if replaced>
```

## 10. Safety Rules

- Do not create `.ai/skills/graft-shared-asset-reuse/SKILL.md`.
- Do not turn the registry into a full source inventory.
- Do not auto-generate registry entries from every file in source trees.
- Do not register generated-only artifacts as authority.
- Do not bypass OpenAPI, module contract, `moduleapi`, `httpx`, `logger`, `configregistry`, or `cronx` authority.
- Do not delete stale registry entries without classifying move, rename, replacement, or intentional removal.
