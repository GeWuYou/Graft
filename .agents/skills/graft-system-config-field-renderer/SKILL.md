---
name: graft-system-config-field-renderer
description: Use when adding or modifying system config definitions, system config schemas, or system config edit/render UI in Graft.
---

# Graft System Config Field Renderer

Use this skill for changes that touch `server` system config definitions, `config_schema` JSON, OpenAPI/system-config
contracts, or `web` UI that edits or renders system config values.

Follow root `AGENTS.md` first. For cross-boundary work, read both `server/AGENTS.md` and `web/AGENTS.md` before
implementation.

## 1. Resolve Authority First

- Treat the module-owned system config definition and its `config_schema` as the first UI rendering authority.
- Check `config_schema` before using `item.type`; use `item.type` only as a fallback when schema is missing or unusable.
- Do not add local UI compatibility mappings until direct schema or definition repair has been ruled out for the slice.
- Missing schema is allowed for definitions that do not need rich rendering metadata; visible schema fallback copy is not
  allowed without matching `x-i18n` keys.

## 2. Field Decision Table

Pick the editor from the effective schema first:

| Schema signal | Editor |
| --- | --- |
| `enum` present | `Select` |
| `type: "boolean"` | `Switch` |
| `type: "integer"` or `type: "number"` | `InputNumber` |
| `type: "object"` or `type: "array"` | JSON textarea |
| `type: "string"` | `Input` |

If the schema is absent, malformed, or does not provide a usable type, apply the same table to `item.type`.

## 3. I18n Requirements

- Resolve display copy from `config_schema["x-i18n"]` before reading schema fallback fields.
- Require `x-i18n.titleKey` for schema `title`, `x-i18n.descriptionKey` for schema `description`, and
  `x-i18n.placeholderKey` for schema `placeholder`.
- For enum options, prefer `x-i18n.enumLabels.<value>.labelKey` and `.descriptionKey` over inline fallback copy.
- Add locale catalog entries in the owning `web/src/modules/<name>/locales/**` boundary when keys are introduced.
- Run `web/scripts/check-i18n-governance.ts` through the documented validation entrypoint after changing schema copy or
  locale keys.

## 4. TDesign Preflight

When changing `web` render or edit UI, query TDesign MCP with `framework=vue-next` before coding for every touched
component from the decision table: `Select`, `Switch`, `InputNumber`, `Textarea`, or `Input`.

Use only the needed MCP calls:

- `get_component_docs` for props, events, slots, and supported usage.
- `get_component_dom` when style overrides, DOM assumptions, or selector work are involved.
- `get_component_changelog` only for upgrade or version-drift risk.

Record the queried components and adoption decision in closeout. If the slice only changes server definitions,
schemas, docs, or validation scripts, record TDesign MCP as not applicable.

## 5. Focused Validation

Run the smallest validation that directly covers the slice:

- schema or server definition changes: focused Go tests for the owning module plus `graft validate backend --stage lint`
  when practical.
- web renderer changes: `cd web && bun run check` or the narrow directly affected test command when the slice is small.
- i18n governance changes: `cd web && bun run test:run scripts/check-i18n-governance.test.ts` and
  `cd web && bun run lint:i18n` when practical.

Report any skipped validation command and the exact reason.
