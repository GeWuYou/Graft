# System Config Model Unification

## Current Status

- Status: `active`.
- Task class: `docs/automation with cross-boundary impact`.
- Goal: promote the System Config information architecture, object/scalar boundary, field renderer baseline, and phased
  optimization route into repository design truth.
- Primary design authority:
  - `ai-plan/design/系统配置模型与渲染设计.md`

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `docs/automation with cross-boundary impact`
- recovery source: `parent topic`
- authority summary: repository-level System Config design is the canonical long-term guidance; module-owned
  `ConfigDefinition` and `config_schema` remain runtime/schema authority; OpenAPI source remains wire-contract
  authority; `web/src/modules/system-config` and `web/src/shared/schema-form` are downstream UI consumers.

## Active Scope

- Keep the design topic focused on System Config model and renderer governance.
- Do not change server, web, OpenAPI, generated artifacts, or locale catalogs in this topic unless a later slice
  explicitly moves from design to implementation.
- Track follow-up implementation as phased slices instead of mixing implementation changes into this design-only topic.

## Entry Points

- Tracking: `ai-plan/public/system-config-model-unification/todos/system-config-model-unification-tracking.md`
- Trace: `ai-plan/public/system-config-model-unification/traces/system-config-model-unification-trace.md`
