# System Config Model Unification Tracking

## Current Goal

Convert the System Config exploration and planning discussion into durable repository design truth.

## Current Scope

- [x] Create repository-level System Config model and renderer design document.
- [x] Register this active topic in `ai-plan/public/README.md`.
- [x] Add topic recovery entrypoint, tracking file, and trace file.
- [x] Phase 1 implementation slice: UI consistency without backend model changes.
- [x] Phase 2 implementation slice: front-end renderer extraction.
- [x] Phase 3 backend registry/OpenAPI enhancement.
- [x] Phase 4 typical config migration.
- [x] Phase 5 validation and archive-readiness.

## Authority Discovery

- Design truth owner: `ai-plan/design/governance/platform/系统配置模型与渲染设计.md`.
- Runtime definition authority: module-owned `ConfigDefinition` and `config_schema`.
- Override persistence authority: `server/modules/system-config`.
- Wire contract authority: OpenAPI source under `openapi/**`.
- UI consumer authority: `web/src/modules/system-config` and shared renderer code under `web/src/shared/schema-form`.

## Current Risks

- Do not let derived `fields` views become a second schema truth.
- Do not migrate old flat keys without explicit compatibility expiry and cleanup conditions.
- Do not plan nested object or array visual builders before backend schema validation supports them.
- Do not let frontend key-specific mappings replace module-owned schema repair.

## Latest Validation

- Phase 5 archive-readiness slice.
- Ran validation:
  - `cd server && go test ./internal/scheduler ./internal/configregistry ./internal/dashboard ./modules/system-config ./modules/notification`
  - `cd server && go run ./cmd/graft validate backend --stage lint`
  - `cd web && bun run check`
- Browser validation was skipped by explicit user constraint.
- Verified no old flat-key runtime/web consumers remain. Old keys remain only in design migration-before notes and tests
  that assert removed keys are ignored or not registered.

## Terminal Archive-Ready Closeout

- completed_batches:
  - Phase1 UI consistency
  - Phase2 renderer extraction
  - Phase3 backend registry/OpenAPI enhancement
  - Phase4 typical config migration
  - Phase5 validation/archive readiness
- pending_batches: []
- current_batch: archive-ready
- next_batch: none
- terminal_status: archive-ready
