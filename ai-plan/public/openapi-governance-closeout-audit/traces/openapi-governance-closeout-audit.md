# OpenAPI Governance Closeout Audit

## Startup Receipt

- governance source: `root AGENTS.md`
- task class: `docs/governance closeout`
- recovery source:
  - `ai-plan/public/openapi-governance-closeout-audit/traces/openapi-governance-closeout-audit.md`
  - `current repository state`
  - `latest openapi-route-coverage-closure closeout`
- branch / worktree:
  - `feat/wt-oapi-codegen-types-only-spike`
- worktree status at start:
  - tracked modifications present in `openapi/**`, `server/internal/contract/openapi/**`, `web/src/contracts/openapi/generated/**`, and selected thin alias consumers
  - untracked additions present for new OpenAPI path/schema fragments and `server/internal/contract/openapi/route_coverage_test.go`
  - no archive closeout docs staged at startup
- owned scope:
  - `ai-plan/public/openapi-governance-closeout-audit/**`
  - related topic status in `ai-plan/public/README.md`
  - `ai-plan/public/oapi-codegen-types-only-spike/**` status archive and cross-reference updates
  - necessary governance/checklist documentation only
- forbidden scope:
  - no backend business implementation changes
  - no frontend page implementation changes
  - no generated artifact changes from this closeout step
  - no `openapi-fetch`
  - no generated server/client switch
  - no new interface contract rollout

## Executive Summary

Closeout result is `complete` and `archived`.

The repository has now reached the intended OpenAPI / `oapi-codegen` governance closure point for the current business-route scope. OpenAPI covers all in-scope business backend routes, generated Go and TS contract outputs are current, thin frontend aliases consume generated types where applicable, and the minimal route-coverage plus generated-freshness gates are in place. Future work should stop treating this as a broad governance program and instead treat contract-first delivery as the default rule for each feature slice.

## Archive Decision

- Archive decision: `approved`
- Archived governance topics:
  - `openapi-governance-closeout-audit`
  - `oapi-codegen-types-only-spike`
- Recommended next topic:
  - `feature-delivery-with-contract-first-rule`
- Standalone governance continuation:
  - none

## Route Coverage Closure

- `total_backend_routes_in_scope=29`
- `covered_by_openapi=29`
- `missing_openapi_paths=[]`
- `excluded_routes=["GET /healthz","GET /docs","GET /openapi.json","GET /openapi.yaml"]`
- `/healthz` policy note:
  - `/healthz` remains listed in the excluded operational-route set for business closure accounting, but it is intentionally still preserved in the root spec as a runtime health contract.

## OAPI Codegen Closure Stage

- current: `["A", "B", "C", "D"]`
- intentionally deferred: `["E"]`
- generated server/client status:
  - intentionally deferred, not missing
- settled repository preference:
  - generated types plus request/response consumption are preferred over full generated server/client runtime adoption

## Completed Governance Capabilities

- documentation page integration is present
- OpenAPI covers the current business API surface in scope
- generated Go types are updated
- generated TS schema is updated
- frontend thin alias layers consume generated types
- route coverage minimal gate is established
- `openapi:types:check` blocks stale generated schema output
- `web check`, `backend validate`, and the magic value scanner pass on the closure state

## Remaining Gaps That Do Not Block Archive

- `ApiEnvelope` base schema remains intentionally weakly typed
  - this does not block archive
- error envelope `data` remains intentionally loose
  - this does not block archive
- `route_coverage_test.go` is a minimal closure-set guard, not a full Gin-to-spec inventory diff
  - this does not block archive

## Future Operating Rule

All new or changed HTTP APIs must follow contract-first feature delivery. Every feature slice that adds or changes an interface must update all of the following in the same slice:

- backend route / handler
- OpenAPI path / schema / example
- generated Go / TS types
- frontend thin alias consumption where applicable
- visible documentation page coverage
- route coverage / `openapi:types:check` / `bun run check` / `go run ./cmd/graft validate backend` / magic value validation

## Validation Expectations For This Closure State

- `git diff --check`
- `cd web && bun run openapi:types:check`
- `cd web && bun run check`
- `cd server && go run ./cmd/graft validate backend`
- `scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci`

## Final Closeout

The OpenAPI / `oapi-codegen` governance topic is archived. The repository should not continue this work as a separate governance stream, should not introduce generated server/client as a default runtime path, and should not broaden closure work beyond the already-covered route set. From this point forward, contract-first feature delivery is the default gate for feature work rather than a separate program.

## Machine-Readable Closeout

```json
{
  "status": "complete",
  "continue": false,
  "archived": true,
  "can_continue_feature_work": true,
  "openapi_governance": {
    "route_coverage": "29/29",
    "missing_openapi_paths": [],
    "excluded_routes": ["GET /healthz", "GET /docs", "GET /openapi.json", "GET /openapi.yaml"],
    "docs_page_integrated": true,
    "route_coverage_gate": "minimal closure-set guard"
  },
  "oapi_codegen": {
    "current": ["A", "B", "C", "D"],
    "intentionally_deferred": ["E"],
    "generated_server_client_deferred_reason": "Not required for current plugin architecture and contract-first feature delivery."
  },
  "remaining_gaps": [
    {
      "item": "ApiEnvelope base schema remains intentionally weakly typed",
      "blocks_archive": false
    },
    {
      "item": "Error envelope data remains intentionally loose",
      "blocks_archive": false
    },
    {
      "item": "route_coverage_test.go is minimal, not a full Gin-to-spec diff",
      "blocks_archive": false
    }
  ],
  "future_rule": "All new or changed HTTP APIs must update OpenAPI, regenerate Go/TS types, consume generated aliases where applicable, and pass openapi/types/web/backend/magic-value gates.",
  "validation": {
    "commands": [
      "git diff --check",
      "cd web && bun run openapi:types:check",
      "cd web && bun run check",
      "cd server && go run ./cmd/graft validate backend",
      "scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci"
    ],
    "passed": [],
    "failed": [],
    "not_run": []
  },
  "recommended_next_topic": "feature-delivery-with-contract-first-rule",
  "next_prompt": "none"
}
```

## Route Coverage Audit

Scope basis:

- real route registration starts from `server/internal/app/runtime.go:166`, where plugins mount under `/api`
- core non-plugin routes are declared in `server/internal/app/runtime.go:231-253`
- plugin routes are declared in:
  - `server/plugins/auth/route_handlers.go:18-176`
  - `server/plugins/user/route_user_handlers.go:17-186`
  - `server/plugins/user/route_admin_session_handlers.go:19-105`
  - `server/plugins/rbac/route_registration.go:135-163`
  - `server/plugins/monitor/plugin.go:398-404`

### In-scope backend routes

| Route | backend_route_found | openapi_path_found | method_matched | request_schema_status | response_schema_status | error_response_status | notes |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `POST /api/auth/login` | yes | yes | yes | matched via `LoginRequest` | matched | partial | spec has `401/500`, no explicit `400` example for malformed body although handler returns invalid-argument on bind failure |
| `POST /api/auth/refresh` | yes | yes | yes | n/a | matched | matched | cookie-based refresh modeled in spec |
| `POST /api/auth/logout` | yes | yes | yes | n/a | matched | matched | empty envelope modeled with `data: null` |
| `GET /api/auth/bootstrap` | yes | yes | yes | n/a | matched | matched | frontend consumes generated bootstrap response |
| `GET /api/auth/sessions` | yes | no | no | missing | missing | missing | real route at `server/plugins/auth/route_handlers.go:94-107` |
| `POST /api/auth/sessions/revoke-all` | yes | no | no | missing | missing | missing | real route at `server/plugins/auth/route_handlers.go:77-84` |
| `POST /api/auth/sessions/revoke-others` | yes | no | no | missing | missing | missing | real route at `server/plugins/auth/route_handlers.go:86-92` |
| `POST /api/auth/sessions/:sessionID/revoke` | yes | no | no | missing | missing | missing | real route at `server/plugins/auth/route_handlers.go:109-130` |
| `POST /api/auth/change-password` | yes | no | no | missing | missing | missing | real route at `server/plugins/auth/route_handlers.go:150-172` |
| `POST /api/auth/complete-required-password-change` | yes | no | no | partial frontend-only | missing | missing | real route at `server/plugins/auth/route_handlers.go:175-194`; frontend still handwrites payload |
| `GET /api/users` | yes | yes | yes | n/a | matched | matched | list response modeled and consumed |
| `GET /api/users/:id` | yes | no | no | missing | missing | missing | real route at `server/plugins/user/route_user_handlers.go:35-51` |
| `POST /api/users` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `POST /api/users/:id/update` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `POST /api/users/:id/status` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type and enum alias |
| `POST /api/users/:id/reset-password` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `POST /api/users/:id/delete` | yes | no | no | missing | missing | missing | real route at `server/plugins/user/route_user_handlers.go:177-186`; frontend consumes it |
| `GET /api/users/:id/sessions` | yes | no | no | missing | missing | missing | real route at `server/plugins/user/route_admin_session_handlers.go:19-45` |
| `POST /api/users/:id/sessions/:sessionID/revoke` | yes | no | no | missing | missing | missing | real route at `server/plugins/user/route_admin_session_handlers.go:53-86` |
| `POST /api/users/:id/sessions/revoke-all` | yes | no | no | missing | missing | missing | real route at `server/plugins/user/route_admin_session_handlers.go:89-105` |
| `GET /api/roles` | yes | yes | yes | n/a | matched | matched | list response modeled and consumed |
| `POST /api/roles` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `GET /api/roles/:id/permissions` | yes | no | no | missing | missing | missing | real route at `server/plugins/rbac/route_registration.go:137-138`; frontend consumes it |
| `POST /api/roles/:id/update` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `POST /api/roles/:id/permissions/assign` | yes | yes | yes | matched | matched | matched | Go handler binds generated request type |
| `GET /api/permissions` | yes | yes | yes | n/a | matched | matched | list response modeled and consumed |
| `GET /api/users/:id/roles` | yes | no | no | missing | missing | missing | real route at `server/plugins/rbac/route_registration.go:162`; frontend consumes it |
| `POST /api/users/:id/roles/assign` | yes | no | no | missing | missing | missing | real route at `server/plugins/rbac/route_registration.go:163-174`; frontend consumes it with handwritten payload |
| `GET /api/monitor/server-status` | yes | no | no | missing query schema | missing | missing | real route at `server/plugins/monitor/plugin.go:398-420`; frontend consumes it heavily |

### Excluded routes

- `GET /healthz`
  - real route at `server/internal/app/runtime.go:231-239`
  - still covered by `openapi/openapi.yaml:19-20`
  - this is operational surface, not plugin business API; keeping it in spec is acceptable as a runtime health contract, but it should not be used as evidence that business API governance is complete
- `GET /docs`
  - real route at `server/internal/app/runtime.go:252-253`
  - intentionally outside business API contract governance
  - topic docs already define it as docs UI exposure, not plugin API truth: `ai-plan/public/openapi-docs-mvp/README.md`
- `GET /openapi.json`
  - real route at `server/internal/app/runtime.go:246-247`
  - intentionally outside business API contract governance
  - it is a distribution artifact endpoint for the spec itself, not a business API
- `GET /openapi.yaml`
  - same reasoning as `/openapi.json`

### Route coverage summary

- total backend routes in requested scope: `29`
- covered by OpenAPI: `13`
- missing OpenAPI paths: `16`

The missing set is concentrated in three already-live areas:

- auth session/password routes
- user detail/delete/admin-session routes
- rbac user-role and role-permission snapshot routes
- monitor server-status route

## OpenAPI Contract Quality

### What is already good

- Root spec is syntactically valid and bundled docs asset exists:
  - `openapi/openapi.yaml:1-45`
  - `openapi/dist/openapi.bundle.json`
  - backend docs loader validates both root and bundled spec in `server/internal/app/openapi_docs.go:50-83`
- User and RBAC write request schemas are aligned well enough that backend handlers bind generated request aliases directly:
  - `server/plugins/user/route_user_handlers.go:60-75`, `96-113`, `120-139`, `148-172`
  - `server/plugins/rbac/route_write_handlers.go:53-75`, `98-126`, `129-163`
- Historical field drift appears largely cleaned up for covered write routes:
  - user create/update use `display`, not `name`: `openapi/components/schemas/create-user-request.yaml:2-12`, `update-user-request.yaml:1-6`
  - reset password uses `new_password`, not `password`: `openapi/components/schemas/reset-user-password-request.yaml:1-7`
  - role APIs use `display` and `description`, not older mixed naming: `create-role-request.yaml:1-13`, `update-role-request.yaml:1-13`

### Contract quality gaps

1. The top-level success envelope is still weakly typed.
   - `openapi/components/schemas/api-envelope.yaml:1-20` leaves `data: {}`
   - this is acceptable only because concrete enveloped schemas refine it downstream; it is not by itself a strong governance surface

2. Error envelope `data` remains intentionally loose.
   - `openapi/components/schemas/error-response.yaml:18-20`
   - this means field-level drift can still happen inside error payload details without compile-time detection

3. Covered auth routes still under-spec malformed request behavior.
   - `POST /api/auth/login` handler returns invalid-argument errors for body / username / password issues at `server/plugins/auth/route_handlers.go:18-36`
   - spec for `openapi/paths/auth.login.yaml:17-36` declares only `200/401/500`
   - practical mismatch: real `400` path exists but spec omits it

4. Entire live route families are absent, so schema quality there is effectively zero.
   - example: `GET /api/monitor/server-status` has live query semantics `trend_range` at `server/plugins/monitor/plugin.go:409-410`, but no OpenAPI path
   - example: `GET /api/roles/:id/permissions` and `GET /api/users/:id/roles` return stable binding snapshots consumed by frontend, but no schema in spec

5. Frontend monitor and some RBAC/auth contracts still rely on handwritten interfaces because the spec does not cover them.
   - `web/src/modules/monitor/types/server-status.ts:91-100`
   - `web/src/modules/rbac/types/rbac.ts:3-9`
   - `web/src/modules/rbac/contract/role.ts:6-8`
   - `web/src/modules/auth/contract/types.ts:12-13`

## oapi-codegen Stage Audit

### Current classification

- `A`: yes
  - evidence:
    - `server/internal/contract/openapi/generate.go:3`
    - `server/internal/contract/openapi/generated/types.gen.go:1-4`
- `B`: yes, partial-to-broad on frontend response type consumption
  - evidence:
    - `web/src/modules/auth/contract/types.ts:5-10`
    - `web/src/modules/user/types/user.ts:5-18`
    - `web/src/modules/rbac/contract/role.ts:3-4`
    - `web/src/modules/rbac/types/permission.ts:4`
- `C`: yes, partial
  - evidence:
    - `web/src/modules/auth/contract/types.ts:10`
    - `web/src/modules/user/types/user.ts:15-18`
    - `web/src/modules/rbac/types/rbac.ts:7-9`
  - still not complete because some request payloads remain handwritten:
    - `web/src/modules/auth/contract/types.ts:12-13`
    - `web/src/modules/user/api/user-roles.ts:5-7`
- `D`: yes, partial and beyond the original "types-only spike"
  - evidence:
    - `server/plugins/user/route_user_handlers.go:60`, `102`, `126`, `154`
    - `server/plugins/rbac/route_write_handlers.go:53`, `98`
    - generated aliases are exposed in `server/internal/contract/openapi/types.go:8-30`
- `E`: no
  - no generated server/client integration found

### Target stage

Current repository intent is still "types-first, not generated runtime":

- intentional target: `A + B + C`, with selective `D` already adopted for backend request binding
- intentionally deferred: `E`
- originally deferred in the spike plan:
  - `ai-plan/public/oapi-codegen-types-only-spike/design/spike-plan.md`
  - it explicitly rejects generated server interfaces, strict server stubs, or runtime clients

### Important governance note

The repository has already moved beyond a pure A-only spike. Backend handlers in `user` and `rbac` now consume generated request aliases directly, so the practical current stage is:

- current: `["A", "B", "C", "D"]`
- intentionally deferred: `["E"]`

## Frontend Consumption Audit

### Closed parts

- `request.ts` remains the single transport/runtime truth:
  - `web/src/utils/request.ts:45-108`, `154-199`, `245-258`
- covered API paths are mostly routed through module-owned path constants:
  - `web/src/modules/auth/contract/paths.ts:1-8`
  - `web/src/modules/user/contract/paths.ts:16-26`
  - `web/src/modules/rbac/contract/paths.ts:1-8`
  - `web/src/modules/monitor/contract/paths.ts:8-10`
- generated types are mostly consumed via module-local thin type files rather than raw page imports:
  - `web/src/modules/auth/contract/types.ts`
  - `web/src/modules/user/types/user.ts`
  - `web/src/modules/rbac/contract/role.ts`
  - `web/src/modules/rbac/types/permission.ts`

### Gaps

1. Manual DTOs still exist for OpenAPI-adjacent routes.
   - `web/src/modules/auth/contract/types.ts:12-13`
   - `web/src/modules/rbac/types/rbac.ts:3-5`
   - `web/src/modules/rbac/contract/role.ts:6-8`
   - `web/src/modules/user/api/user-roles.ts:5-7`
   - `web/src/modules/monitor/types/server-status.ts:3-100`

2. Some hardcoded API paths remain outside the audited business modules.
   - scanner findings in `web/mock/index.ts`
   - those are not in the requested auth/user/rbac/permissions/monitor modules, but they prove the gate is repo-wide imperfect

3. Frontend is already consuming non-spec backend routes.
   - `web/src/modules/rbac/api/rbac.ts:25-28` hits `GET /api/roles/:id/permissions`
   - `web/src/modules/user/api/user-roles.ts:15-24` hits `GET/POST /api/users/:id/roles*`
   - `web/src/modules/monitor/api/server-status.ts:7-13` hits `GET /api/monitor/server-status`
   - `web/src/modules/user/api/users.ts:124-127` hits `POST /api/users/:id/delete`

### Frontend markers

- generated_response_consumed: `true`
- generated_request_consumed: `true`
- local_form_model_allowed: `true`
  - page-local UI form models in `web/src/modules/user/pages/index.vue` are acceptable because they are UI state, not transport DTO truth
- manual_dto_left:
  - `web/src/modules/auth/contract/types.ts`
  - `web/src/modules/rbac/types/rbac.ts`
  - `web/src/modules/rbac/contract/role.ts`
  - `web/src/modules/user/api/user-roles.ts`
  - `web/src/modules/monitor/types/server-status.ts`
- hardcoded_api_path_left:
  - business modules in requested scope: no critical direct hard-coded paths found; module API calls route through contract path constants
  - repo-wide scanner still reports `web/mock/index.ts`

## Validation Gate Audit

### Executed commands

- `cd web && bun run openapi:types:check`
  - passed
  - this catches `openapi/openapi.yaml` to `web/src/contracts/openapi/generated/schema.ts` drift
- `cd web && bun run check`
  - passed
  - this catches frontend typecheck, lint, tests, and build regressions
- `cd server && go run ./cmd/graft validate backend`
  - before this closure round, backend validation could only prove syntax/compile correctness for already-documented paths
  - after this closure round, `server/internal/contract/openapi/route_coverage_test.go` adds a minimal guard for the live-route closure set:
    - required covered routes:
      - `/api/auth/sessions`
      - `/api/auth/sessions/revoke-all`
      - `/api/auth/sessions/revoke-others`
      - `/api/auth/sessions/{sessionID}/revoke`
      - `/api/auth/change-password`
      - `/api/auth/complete-required-password-change`
      - `/api/users/{id}`
      - `/api/users/{id}/delete`
      - `/api/users/{id}/sessions`
      - `/api/users/{id}/sessions/{sessionID}/revoke`
      - `/api/users/{id}/sessions/revoke-all`
      - `/api/roles/{id}/permissions`
      - `/api/users/{id}/roles`
      - `/api/users/{id}/roles/assign`
      - `/api/monitor/server-status`
    - excluded routes recorded intentionally:
      - `/healthz`
      - `/docs`
      - `/openapi.json`
      - `/openapi.yaml`
  - passed
  - this validates backend OpenAPI stage, lint/build/test chain
- requested `python scripts/magic_value/check_magic_values.py`
  - failed as-written because `python` is unavailable in repo environment
- actual repository scanner entrypoint used instead:
  - `scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci`
  - passed with findings

### What the current gates can stop

- OpenAPI changed but frontend generated schema not updated
  - yes
  - `web/package.json:28-29`
- Frontend generated type consumption incompatible with current code
  - yes, for already-generated surfaces
  - `bun run check` typecheck covers this
- Backend lint/build/test regressions
  - yes
  - `server/internal/cli/validate.go:187-227`
- API path string sprawl
  - partially
  - magic-value scanner flags hard-coded API literals, but currently still exits zero with findings and reports unrelated mock/starter debt
- Contract field-name drift
  - partially
  - generated type consumption catches drift only for covered paths and covered payloads
  - missing spec paths and loose error/data envelopes still leave drift space

### What the current gates do not stop

- a real backend route being added or already existing without any OpenAPI path
  - no dedicated gate found
- frontend consuming a live backend route that is absent from the root spec
  - no direct gate found
- monitor/auth session/user-role snapshot routes staying entirely outside generated contracts
  - no direct gate found

## Final Closeout

- closeout_status: `incomplete`
- remaining_gaps:
  - root OpenAPI spec covers only `13/29` backend routes in requested scope
  - live monitor route missing from spec
  - live auth session/password-management routes missing from spec
  - live user detail/delete/admin-session routes missing from spec
  - live RBAC role-permission snapshot and user-role routes missing from spec
  - frontend still carries manual DTOs for uncovered paths
  - route-presence governance gate is missing
- must_fix_before_archive:
  - add missing live backend routes to `openapi/openapi.yaml` and fragments
  - generate and consume types for monitor/user-role/role-permission/auth-password-session gaps
  - add a route-coverage audit gate or documented inventory check so real routes cannot exist outside spec silently
- can_continue_feature_work: `true`
  - but feature work should treat OpenAPI governance as active debt, not archived
- recommended_next_topic:
  - `openapi-route-coverage-closure`
  - close missing live route families before any attempt to archive the governance topic

### Minimal fix plan

1. Add OpenAPI fragments for the already-live missing routes without changing runtime behavior:
   - monitor server-status
   - rbac role-permission binding and user-role binding/assign
   - user delete and admin-session routes
   - auth session and password-management routes
2. Regenerate frontend and Go contract types.
3. Replace the remaining handwritten transport DTOs where those routes are now covered.
4. Add a route inventory check that compares real Gin registrations against the root spec, or at minimum codify the inventory in a maintained audit test.

## Human-Readable Summary

The OpenAPI / oapi-codegen governance work is not ready for archive. The generated-type toolchain itself is healthy, and the repository has already advanced to `A+B+C+D` in practice for covered user/RBAC write routes, but the governance perimeter is still incomplete because many real backend routes remain outside the spec while the frontend already consumes some of them. Current validation catches stale generated artifacts and typed drift on covered surfaces, but it does not prevent uncovered live routes from bypassing governance entirely.

## Machine-Readable JSON Closeout

```json
{
  "status": "incomplete",
  "continue": true,
  "closeout_status": "OpenAPI and oapi-codegen governance are partially established but not archive-ready because many live backend routes remain outside the root spec and generated type surface.",
  "route_coverage": {
    "total_backend_routes_in_scope": 29,
    "covered_by_openapi": 13,
    "missing_openapi_paths": [
      "GET /api/auth/sessions",
      "POST /api/auth/sessions/revoke-all",
      "POST /api/auth/sessions/revoke-others",
      "POST /api/auth/sessions/:sessionID/revoke",
      "POST /api/auth/change-password",
      "POST /api/auth/complete-required-password-change",
      "GET /api/users/:id",
      "POST /api/users/:id/delete",
      "GET /api/users/:id/sessions",
      "POST /api/users/:id/sessions/:sessionID/revoke",
      "POST /api/users/:id/sessions/revoke-all",
      "GET /api/roles/:id/permissions",
      "GET /api/users/:id/roles",
      "POST /api/users/:id/roles/assign",
      "GET /api/monitor/server-status"
    ],
    "excluded_routes": [
      "GET /healthz",
      "GET /docs",
      "GET /openapi.json",
      "GET /openapi.yaml"
    ]
  },
  "oapi_codegen_stage": {
    "current": ["A", "B", "C", "D"],
    "intentionally_deferred": ["E"],
    "evidence": [
      "server/internal/contract/openapi/generate.go:3",
      "server/internal/contract/openapi/generated/types.gen.go:1",
      "server/plugins/user/route_user_handlers.go:60",
      "server/plugins/rbac/route_write_handlers.go:53",
      "web/src/modules/user/types/user.ts:15",
      "web/src/modules/auth/contract/types.ts:10"
    ]
  },
  "frontend_consumption": {
    "generated_response_consumed": true,
    "generated_request_consumed": true,
    "manual_dto_left": [
      "web/src/modules/auth/contract/types.ts",
      "web/src/modules/rbac/types/rbac.ts",
      "web/src/modules/rbac/contract/role.ts",
      "web/src/modules/user/api/user-roles.ts",
      "web/src/modules/monitor/types/server-status.ts"
    ],
    "hardcoded_api_path_left": [
      "web/mock/index.ts"
    ]
  },
  "validation": {
    "commands": [
      "cd web && bun run openapi:types:check",
      "cd web && bun run check",
      "cd server && go run ./cmd/graft validate backend",
      "python scripts/magic_value/check_magic_values.py",
      "scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci"
    ],
    "passed": [
      "cd web && bun run openapi:types:check",
      "cd web && bun run check",
      "cd server && go run ./cmd/graft validate backend",
      "scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci"
    ],
    "failed": [
      "python scripts/magic_value/check_magic_values.py"
    ],
    "not_run": []
  },
  "remaining_gaps": [
    "Real backend route coverage is incomplete: only 13 of 29 in-scope routes are represented in the root OpenAPI spec.",
    "Frontend consumes live non-spec routes in auth, user-role, role-permission, delete, and monitor flows.",
    "No gate was found that fails when a live Gin route exists outside the root OpenAPI spec.",
    "Some request/response DTOs remain handwritten because their routes are not covered by the spec."
  ],
  "must_fix_before_archive": [
    "Add missing live backend routes to the root OpenAPI spec and fragments.",
    "Generate and consume contract types for the newly covered auth/user/rbac/monitor routes.",
    "Add a route-coverage gate that cross-checks real route registrations against the spec."
  ],
  "can_continue_feature_work": true,
  "recommended_next_topic": "openapi-route-coverage-closure",
  "next_prompt": "Rerun startup preflight from root AGENTS.md as a cross-boundary task, recover from current repository state plus ai-plan/public/openapi-governance-closeout-audit, and close the missing live-route coverage before attempting to archive OpenAPI governance."
}
```

## Loop Note

This audit used the `graft-multi-agent-loop` orchestration shape, but the delegated read-only explorer round did not return a usable closeout or checkpoint response within budget. The main agent completed the evidence collection locally and fail-closed the loop outcome as a worker closeout failure rather than relying on an incomplete delegated result.

## Route Coverage Closure Round

Date:

- `2026-05-24`

Round scope:

- `openapi-route-coverage-closure`
- owned scope:
  - `openapi path fragments`
  - `openapi root spec wiring`
  - `generated OpenAPI types`
  - `frontend thin alias / contract consumption`
  - `minimal route-coverage validation`
  - `topic trace docs`

### Before / After

- route coverage before:
  - recounted in-scope business routes covered by root spec: `14 / 29`
  - missing business OpenAPI paths: `15`
- route coverage after:
  - in-scope business routes covered by root spec: `29 / 29`
  - missing business OpenAPI paths: `0`

Closed routes in this round:

- `GET /api/auth/sessions`
- `POST /api/auth/sessions/revoke-all`
- `POST /api/auth/sessions/revoke-others`
- `POST /api/auth/sessions/{sessionID}/revoke`
- `POST /api/auth/change-password`
- `POST /api/auth/complete-required-password-change`
- `GET /api/users/{id}`
- `POST /api/users/{id}/delete`
- `GET /api/users/{id}/sessions`
- `POST /api/users/{id}/sessions/{sessionID}/revoke`
- `POST /api/users/{id}/sessions/revoke-all`
- `GET /api/roles/{id}/permissions`
- `GET /api/users/{id}/roles`
- `POST /api/users/{id}/roles/assign`
- `GET /api/monitor/server-status`

Already covered notes:

- `GET /healthz`
  - still present in `openapi/openapi.yaml`
  - treated as an operational runtime endpoint, not as part of the business OpenAPI governance closure set

### Excluded Routes

- `GET /healthz`
  - excluded from business API governance closure because it is a core runtime health endpoint, not a plugin business route
  - intentionally still documented in the root spec as an operational contract
- `GET /docs`
  - excluded because it serves docs UI exposure, not business API semantics
- `GET /openapi.json`
  - excluded because it distributes the bundled spec artifact itself
- `GET /openapi.yaml`
  - excluded because it distributes the root spec artifact itself

### Frontend Contract Closure

- replaced handwritten thin DTOs in owned scope with generated aliases for:
  - auth restricted password-change payload
  - rbac role-permission binding response
  - rbac user-role binding response
  - user-role assign request payload
  - monitor server-status response family
- kept local UI form models unchanged
- `request.ts` remained the only transport/runtime truth

### Validation Result

- `cd web && bun run openapi:types:check`
  - passed
- `cd web && bun run check`
  - passed
- `cd server && go run ./cmd/graft validate backend`
  - passed
- `scripts/run_python.sh scripts/magic_value/check_magic_values.py --mode ci`
  - passed

Additional guard added in this round:

- `server/internal/contract/openapi/route_coverage_test.go`
  - validates that the closure-set live routes remain present in the root spec
  - keeps `/docs`, `/openapi.json`, and `/openapi.yaml` out of the root business spec
  - records `/healthz` as operationally documented but excluded from business closure semantics

### Remaining Gaps

- no remaining missing routes in the requested closure set
- no remaining handwritten request/response transport DTOs in the requested auth/user/rbac/monitor owned scope
- residual non-blocking governance debt remains outside this round:
  - top-level success envelope base schema is still intentionally weakly typed
  - error envelope `data` remains intentionally loose
  - the new route-coverage gate is a minimal closure guard for the audited route set, not a full automatic Gin-to-spec inventory diff

### Archive Readiness

- can_archive_openapi_governance: `true`
- rationale:
  - all requested missing live business routes are now represented in the root spec
  - frontend thin alias consumption is aligned for the newly covered routes in owned scope
  - backend and frontend validation entrypoints are green
  - excluded operational/docs routes are explicitly documented with reasons
