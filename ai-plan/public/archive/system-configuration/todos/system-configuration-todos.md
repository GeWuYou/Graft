# System Configuration Todos

## Loop Batch State

- completed_batches:
  - Batch 1: backend authority baseline and OpenAPI source
  - Batch 2: web module and shared schema form reuse
  - Batch 3: initial config definitions and final cross-boundary validation
- pending_batches: []
- current_batch: archive-ready
- next_batch: none
- terminal_status: archive-ready

## Batch 1 - Backend Authority Baseline

Status: accepted by the loop owner; focused validation recorded in the trace.

- [x] Add `server/internal/configregistry` with `ConfigDefinition`, registry validation, sensitivity/masking metadata, and tests.
- [x] Add `server/modules/system-config` with permissions, menu, messages, route contract, module registration, service/store boundary, and migration for override-only `system_config_values`.
- [x] Add OpenAPI source for list/detail/update/reset system config contracts, including `sensitive`, `masked`, and non-plaintext sensitive value response behavior.
- [x] Wire module registration only as far as needed for compile/build validation.
- [x] Keep API routes on `/api/system-configs` while the web menu remains `/server/system-config`.
- [x] Place System Configuration after Scheduled Tasks in Service Management menu order.

## Batch 2 - Web Module

- [x] Add `web/src/modules/system-config`.
- [x] Add contract paths/permissions/bootstrap route for `/server/system-config`.
- [x] Reuse or lift Scheduled Task JSON Schema parsing/form logic into a shared, business-neutral boundary.
- [x] Implement a MVP settings page with group navigation, effective/default/override visibility, masked sensitive values, save, and reset.
- [x] Regenerate `web/src/contracts/openapi/generated/schema.ts` from OpenAPI source so the web API boundary consumes `/api/system-configs`.
- [x] Add focused route/API tests for System Configuration.

## Batch 3 - Initial Definitions And Closeout

- [x] Register first low-risk definitions from logging/access-log/audit retention defaults.
- [x] Ensure definitions do not copy canonical defaults into the database.
- [x] Run focused and completion validations for server and web.
- [x] Update recovery trace and archive-readiness status.

## Terminal Archive-Ready Closeout

- [x] Outer loop accepted Batch 3 closeout and scoped commit `326835e`.
- [x] Confirmed no pending same-session implementation batches remain.
- [x] Confirmed topic acceptance conditions are met.
- [x] Marked the parent topic `archive-ready`.
