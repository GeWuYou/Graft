# System Configuration Trace

## 2026-06-07 topic start

- Re-ran startup preflight for a `cross-boundary` task:
  - read root `AGENTS.md`
  - read `.ai/environment/tools.ai.yaml`
  - read `server/AGENTS.md`
  - read `web/AGENTS.md`
  - read `$graft-multi-agent-loop` and `$graft-multi-agent-task`
- Confirmed topic authority:
  - `server/internal/configregistry` should own the platform declaration registry
  - `server/modules/system-config` should own persistence, API, menu, permissions, and service behavior
  - OpenAPI source should own wire contracts
  - web should consume backend menu/permission/OpenAPI contracts
- Renamed branch from `feat/scheduled-task-mvp` to `feat/system-configuration`.
- User constraints incorporated:
  - plan sensitive/masked fields in OpenAPI from the start
  - do not place the registry in `server/internal/config`
  - store administrator overrides only in `system_config_values`
  - keep `ConfigDefinition` as module-registered authority, not persisted truth

## 2026-06-07 Batch 1 retry worker

- Implemented backend authority baseline:
  - added `server/internal/configregistry` for module-registered `ConfigDefinition` authority, validation, value typing, and masking metadata
  - exposed the registry through `module.Context` and the runtime service container
  - added `server/modules/system-config` with typed route/permission/message contracts, menu placement at `/server/system-config`, route handlers, service/store boundary, and override-only SQL migration
  - wired `system-config` into compile-time module registration with module-owned migration path
- Implemented OpenAPI source baseline:
  - added list/detail/update/reset paths
  - added schemas for system config items, list responses, update request, and envelopes
  - modeled `sensitive`, `masked`, nullable value fields, and `masked_placeholder` so sensitive effective/default/current values are not returned as plaintext
- Generation note:
  - added the backend systemconfig generated-type package needed by the new routes
  - current OpenAPI freshness automation does not yet include a `backend-system-config` target; extending that checker is left for a later validation/automation slice

## 2026-06-07 Batch 1 owner acceptance

- Corrected the API/menu boundary before commit:
  - API source and generated bundle use `/api/system-configs`
  - the Service Management menu path remains `/server/system-config`
  - menu order is `105`, after Scheduled Tasks (`104`)
- Re-ran focused validation:
  - `cd server && go test ./internal/configregistry ./modules/system-config ./internal/moduleregistry ./internal/app`
  - `cd server && go build ./cmd/graft`
  - `cd server && go test ./internal/contract/openapi/...`
  - `git diff --check`

## 2026-06-07 Batch 2 worker

- Implemented the web System Configuration module:
  - added `web/src/modules/system-config` with module registration, bootstrap route, route/API/permission contracts, locales, and typed API wrappers
  - registered `/server/system-config` as a Service Management dynamic route consumer with stable route name `SystemConfigList`
  - added a settings-style MVP page that lists module/grouped config definitions, displays effective/default/override values, shows masked sensitive values, supports override save, and supports reset to module default
- Reused JSON Schema form capability through a shared boundary:
  - added `web/src/shared/schema-form` with business-neutral JSON parsing, JSON Schema parsing, value preview, and `JsonSchemaValueFields`
  - changed Scheduled Task schema/json utils into thin re-exports over the shared implementation so existing task-instance config behavior keeps using the same API
- Refreshed frontend OpenAPI generated types:
  - regenerated `web/src/contracts/openapi/generated/schema.ts` so the web contract boundary includes `/api/system-configs`
  - kept OpenAPI source authority unchanged in this batch
- Focused validation run by the worker:
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run typecheck`
  - `cd web && bunx vitest run src/modules/index.test.ts src/modules/system-config/bootstrap-routes.test.ts src/modules/system-config/api/system-config.test.ts`
  - `cd web && bun run lint:i18n`
  - `cd web && bun run lint`
  - `cd web && bun run stylelint`
  - `cd web && bun run format:check`
  - `git diff --check`
- Completion validation run by the worker:
  - `cd web && bun run check`
  - covered format, typecheck, OpenAPI frontend governance, i18n governance, lint, stylelint, hygiene, full vitest, and release build
  - full vitest passed with 64 test files and 361 tests
  - `git diff --check`

## 2026-06-07 Batch 3 worker

- Registered the first low-risk `ConfigDefinition` values:
  - `httpx.access-log-retention-cleanup` in `server/internal/httpx`
  - `logger.app-log-retention-cleanup` in `server/internal/logger`
  - `audit.audit-log-retention-cleanup` in `server/modules/audit`
- Kept authority and runtime behavior separate:
  - registered definitions expose existing cleanup job default JSON and schema as module/core-owned metadata
  - cleanup handlers and Scheduled Task instance `DefaultConfig` behavior remain unchanged
  - `system_config_values` still stores administrator overrides only; defaults are not seeded or copied into rows
- Strengthened override-only proof:
  - `List` and `Get` return module defaults without persisted overrides
  - `Update` writes one administrator override
  - `Reset` deletes the override and returns the module default again
- Validation run:
  - `cd server && go test ./internal/configregistry ./modules/system-config ./internal/httpx ./internal/logger ./modules/audit ./internal/app`
  - `cd server && go run ./cmd/graft validate backend`
  - `cd web && bun run check`
  - `git diff --check`
- Validation result:
  - backend completion entrypoint passed after fixing lint findings in the touched system-configuration/configregistry slice
  - web completion entrypoint passed; Vitest reported 64 test files and 361 tests passed
  - `git diff --check` passed
- Archive-readiness judgment:
  - all three planned batches are complete
  - acceptance conditions are met for registry placement, override-only persistence, sensitive/masked OpenAPI response shape, menu placement, and server/web validation
  - no additional in-scope implementation batch remains; outer loop can perform terminal archive-ready handling

## 2026-06-07 outer loop archive-ready closeout

- Accepted Batch 3 worker closeout:
  - scoped commit: `326835e feat(system-config): register initial retention defaults`
  - changed scope stayed inside allowed Batch 3 server/system-config, retention-definition, and recovery-doc boundaries
  - worker validation passed:
    - `cd server && go test ./internal/configregistry ./modules/system-config ./internal/httpx ./internal/logger ./modules/audit ./internal/app`
    - `cd server && go run ./cmd/graft validate backend`
    - `cd web && bun run check`
    - `git diff --check`
- Ran terminal archive-readiness review:
  - `ConfigDefinition` authority is registered through `server/internal/configregistry`
  - `server/internal/config` remains startup/environment configuration only
  - `server/modules/system-config` stores administrator override JSON only in `system_config_values`
  - first low-risk defaults are registered from access-log, app-log, and audit retention cleanup authority without changing Scheduled Task instance config semantics
  - OpenAPI and web MVP surfaces remain aligned with `/api/system-configs` and `/server/system-config`
  - server and web completion entrypoints passed for the final cross-boundary slice
- Topic verdict: `archive-ready`.
- Remaining future scope is non-blocking:
  - auth/login/password-policy definitions should be added in a later bounded slice after the baseline remains stable
