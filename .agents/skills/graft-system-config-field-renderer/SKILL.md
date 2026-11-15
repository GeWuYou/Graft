---
name: graft-system-config-field-renderer
description: "Choose schema-driven editors and locale keys when changing Graft system-config definitions, schemas, or rendering UI."
---

# Graft System Config Field Renderer

Use this skill for changes that touch `server` system config definitions, `config_schema` JSON, OpenAPI/system-config
contracts, or `web` UI that edits or renders system config values.

Follow root [AGENTS.md](../../../AGENTS.md) first. For cross-boundary work, read both [server/AGENTS.md](../../../server/AGENTS.md) and [web/AGENTS.md](../../../web/AGENTS.md) before
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

## 4. TDesign Evidence

When changing a renderer, resolve new or uncertain TDesign API/DOM/version behavior using [the TDesign guidance](../../../ai-plan/design/governance/frontend/TDesign-MCP-辅助开发规范.md), with `framework=vue-next`. Reuse existing task evidence for unchanged known components. Query docs for API details, DOM for uncertain selectors or overrides, and changelog for version risk. Record material adoption decisions or missing evidence.

## 5. Validation

Use `graft-validation-runner` under root validation phases. Commit preparation covers changed server definitions,
web renderer consumers and affected i18n tests. Full CI runs the corresponding domain checks.

Report any skipped validation command and the exact reason.
