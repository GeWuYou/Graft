---
name: graft-localization-governance
description: Repository-specific workflow for Graft localization and i18n changes. Use when adding or changing server i18n facade behavior, locale resource files, message keys, JSON Schema x-i18n metadata, web locale catalogs, locale aggregation, or key-first localization governance.
---

# Graft Localization Governance

Use this skill before changing Graft localization behavior, locale catalogs, message keys, server i18n registration, or
web locale aggregation.

Treat root `AGENTS.md` as startup truth. This skill does not define a second validation, commit, or recovery workflow.

## Read First

1. Complete root `AGENTS.md` startup preflight.
2. Read the task-class AGENTS files:
   - server-only i18n work: `server/AGENTS.md`
   - web-only locale work: `web/AGENTS.md`
   - shared keys, menu, permissions, routes, OpenAPI, or schema metadata: both files
3. Read `ai-plan/design/本地化与i18n治理规范.md`.
4. Read `ai-plan/design/契约治理与魔法值治理规范.md` when adding or changing stable keys.
5. Read `ai-plan/public/localization-governance/README.md` when continuing the localization migration topic.

## Authority Rules

- Keep `server/internal/i18n.Service` as the only server i18n facade.
- Treat embedded locale YAML under `server/internal/i18n/locales/**` as the canonical backend truth for user-visible localized copy.
- Keep locale resource embed, load, validate, freeze, and registry construction centralized in `server/internal/i18n`.
- Do not add new production Go user-visible hardcoded localization copy; only technical identifiers may remain as Go strings by default.
- Do not let business modules, `configregistry`, `httpx`, or `moduleapi` import `go-i18n`, loader internals, or provider
  internals.
- Do not let business modules embed or load locale files themselves, even for module-owned namespaces.
- Keep `web/src/locales/**` as the web locale state and aggregation boundary.
- Keep module web messages in `web/src/modules/<name>/locales/**`; do not copy module keys into root catalog.
- Prefer stable keys over server-provided final text for menus, errors, permissions, system config metadata, and schema
  labels.
- Treat fallback text as a temporary exception only; register file, field, reason, removal condition, and validation scope when direct authority repair cannot be completed in the same slice.

## Server Workflow

1. Classify messages by namespace and owner before editing.
2. Preserve existing facade types: `Namespace`, `LocaleTag`, `MessageKey`, `MessageResource`, `Registration`, and
   `LookupRequest`.
3. For resource-file work, convert flat YAML entries into `i18n.Registration` and register through
   `Service.RegisterMessages`; do not bypass duplicate-key, unsupported-locale, or freeze rules.
4. Keep backend locale resources centralized under `server/internal/i18n/locales/*.yaml` and
   `server/internal/i18n/locales/modules/*.yaml`; module ownership is semantic, not a license to add per-module loaders.
5. Keep centralized loader coverage for both root and nested module locale files without changing facade or provider exposure.
6. Do not migrate all `defaultCatalogEntries` in an early phase. Treat core HTTP error copy as high blast radius.
7. Keep JSON Schema `x-i18n.titleKey`, `descriptionKey`, and `enumLabels` intact.
8. For menus, widgets, quick links, retention jobs, cron actions, explorer metadata, permission display metadata, and config-definition visible fields, prefer locale-key/resource-backed authority and remove Go fallback copy when the current call chain supports it.
9. Keep `LookupRequest.TemplateData` as the future template bridge; do not expose provider-specific template types.

## Web Workflow

1. Use existing locale aggregation and `bun run lint:i18n` rules.
2. Put shell-owned copy in `web/src/locales/**`.
3. Put module-owned copy in `web/src/modules/<name>/locales/**`.
4. Do not use backend final text as the primary UI truth when a stable key or stable code exists.
5. Keep visible time formatting locale-aware and do not change wire contracts into localized strings.

## Phase Defaults

- Phase 1: add server embedded YAML loader and tests; keep map catalog and facade, and centralize the locale directory strategy in `server/internal/i18n`.
- Phase 2: migrate dashboard quick actions system-config copy as the first sample.
- Phase 3: migrate system-config copy in batches.
- Phase 4: migrate menu, notification, announcement, scheduler, container, and log explorer display copy.
- Phase 5: evaluate `go-i18n` only if plural, template, or translation workflow needs justify it.

## Validation

For docs or skill-only changes:

```bash
git diff --check
python3 /root/.codex/skills/.system/skill-creator/scripts/quick_validate.py .agents/skills/graft-localization-governance
```

For server i18n implementation:

```bash
cd server && go test ./internal/i18n/...
cd server && go run ./cmd/graft validate backend --stage lint
cd server && go build ./cmd/graft
```

For web locale changes:

```bash
cd web && bun run lint:i18n
cd web && bun run check
```

For cross-boundary localization work, validate both sides and report any skipped command with the exact reason.

## Closeout Evidence

```text
Localization governance:
- task_class: server | web | cross-boundary | docs/automation
- owned_scope: <paths>
- authority: server/internal/i18n.Service | web/src/locales aggregation | module locale catalog | shared key contract
- provider_exposure: none | blocked
- resource_format: flat-yaml | not-applicable
- validation: <commands and results>
```
