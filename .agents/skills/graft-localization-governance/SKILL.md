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
3. Read `ai-plan/design/governance/platform/本地化与i18n治理规范.md`.
4. Read `ai-plan/design/governance/platform/契约治理与魔法值治理规范.md` when adding or changing stable keys.
5. Read `ai-plan/public/localization-governance/README.md` when continuing the localization migration topic.
6. Read `ai-plan/public/server-locale-ownership-migration/README.md` when the task is specifically about server locale physical ownership, owner-local embedded resources, or the final drift audit for that migration.

## Authority Rules

- Keep `server/internal/i18n.Service` as the only server i18n facade.
- Treat backend embedded locale YAML as canonical user-visible server copy, with physical ownership split across:
  - `server/internal/i18n/locales/*.yaml` for core/display
  - `server/internal/moduleruntime/locales/*.yaml` for internal runtime-owned copy
  - `server/modules/<name>/locales/*.yaml` for module-owned copy
- Keep locale resource embed, load, validate, freeze, and registry construction centralized in `server/internal/i18n`.
- Treat `server/internal/i18n/locales/modules/` as guard-only legacy-free state; do not reintroduce module-owned or runtime-owned locale YAML there.
- Do not add new production Go user-visible hardcoded localization copy; only technical identifiers may remain as Go strings by default.
- Do not let business modules, `configregistry`, `httpx`, or `moduleapi` import `go-i18n`, loader internals, or provider
  internals.
- Owner packages may `go:embed locales/*.yaml` and expose read-only resource descriptors, but must not parse YAML, validate keys,
  build registries, or freeze locale state themselves.
- Keep `web/src/locales/**` as the web locale state and aggregation boundary.
- Keep module web messages in `web/src/modules/<name>/locales/**`; do not copy module keys into root catalog.
- Prefer stable keys over server-provided final text for menus, errors, permissions, system config metadata, and schema
  labels.
- Treat fallback text as a temporary exception only; register file, field, reason, removal condition, and validation scope when direct authority repair cannot be completed in the same slice.
- Do not keep `permission.Item{Name, Description}` user-visible fallback text in registration sources; if display text is still required by current APIs, resolve it from locale keys through `i18n.Service`.
- Do not keep core dashboard/runtime visible copy as raw Go strings when a stable title/description/label key already exists.
- Do not keep production TS/Vue bilingual locale objects such as `[LOCALE.ZH_CN]: '工作台'` / `[LOCALE.EN_US]: 'Workspace'`; locale catalogs are the only canonical truth for visible UI copy.
- 2026-06-18 当前无登记中的生产 Go 用户可见本地化硬编码例外，也无登记中的生产 TS/Vue 双语 UI 硬编码例外；除显式登记项外，不应再接受新的用户可见硬编码本地化 copy。

## Server Workflow

1. Classify messages by namespace and owner before editing.
2. Preserve existing facade types: `Namespace`, `LocaleTag`, `MessageKey`, `MessageResource`, `Registration`, and
   `LookupRequest`.
3. For resource-file work, convert flat YAML entries into `i18n.Registration` and register through
   `Service.RegisterMessages`; do not bypass duplicate-key, unsupported-locale, or freeze rules.
4. Keep backend locale infrastructure centralized in `server/internal/i18n`, but let resource ownership follow the owner package
   directory.
5. Register owner-local embedded resources through runtime pre-registration before module `Register`; do not change facade or provider exposure.
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
6. Treat `[LOCALE.ZH_CN]` / `[LOCALE.EN_US]` computed property bilingual objects the same as plain `'zh-CN'` / `'en-US'` hardcoded copy; they must be moved into locale catalogs.

## Phase Defaults

- Phase 1: add raw embedded resource registration to `server/internal/i18n`; do not move files yet.
- Phase 2: migrate one low-risk module such as `announcement` or `container`.
- Phase 3: migrate remaining module-owned locale resources.
- Phase 4: migrate internal runtime-owned locale resources such as `module-runtime`.
- Phase 5: update governance docs, recovery materials, and CI/script drift guards.

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
