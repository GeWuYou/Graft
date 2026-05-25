# OpenAPI Codegen Governance Audit

## Scope

This topic records a read-first audit of the current Graft OpenAPI / generated-type / docs-page state.

- Topic: `openapi-codegen-governance-audit`
- Task class: `cross-boundary`
- Branch: `feat/wt-oapi-codegen-types-only-spike`
- Audit mode: read-first; no business-runtime refactor
- Writable scope for this topic:
  - `ai-plan/public/openapi-codegen-governance-audit/**`
- Read-only inspected scope:
  - `openapi/**`
  - `server/**`
  - `web/**`
  - `scripts/**`
  - `.github/**`

Boundary note:

- This topic directory is intentionally created under the allowed scope.
- The active-topic index at `ai-plan/public/README.md` was not updated in this slice because the declared owned scope did not include that file.

## Startup Receipt

- Governance source: root `AGENTS.md`
- Task class: `cross-boundary`
- Recovery source: current repository state + `ai-plan/public`
- Repository root: current worktree
- Active branch: `feat/wt-oapi-codegen-types-only-spike`
- Worktree at audit start: clean
- Owned scope: this audit directory only for writes
- Forbidden scope: business handlers/services/repositories, schema/migration, generated files, CI hook behavior changes, Swagger UI / Scalar / Redoc dependency introduction

## Toolchain Map

| Dimension | Current State | Evidence | Completion | Risk | Next Step |
| --- | --- | --- | --- | --- | --- |
| Root OpenAPI source | Root spec lives at `openapi/openapi.yaml` and assembles split `paths/**` plus `components/**` fragments. | `openapi/openapi.yaml`, `openapi/paths/**`, `openapi/components/**` | Done | Coverage still limited to health/auth/users/roles/permissions subset. | Keep extending the root spec instead of creating parallel copies. |
| Envelope and error schema | Shared `ApiEnvelope` and `ErrorResponse` schemas exist; localized `messageKey` / `locale` / `data.field` semantics are modeled. | `openapi/components/schemas/api-envelope.yaml`, `openapi/components/schemas/error-response.yaml` | Done | Error examples are uneven across routes; `409` remains under-modeled. | Standardize write-route examples and conflict semantics route by route. |
| Backend OpenAPI validation | Backend validate entrypoint runs root spec validation through `kin-openapi`. | `server/internal/cli/validate.go` | Done | Validates spec shape, not generated-Go freshness. | Keep as canonical spec gate; add separate generated drift gate later. |
| Backend `oapi-codegen` config | A scoped Go types-only config exists under a non-runtime contract package. | `server/internal/contract/openapi/oapi-codegen.yaml` | Done | Historical docs saying `oapi-codegen` is deferred are now partially stale relative to this branch. | Update governance docs to distinguish "no generated server interfaces" from "types-only spike exists". |
| Backend generation entrypoint | `go generate` entrypoint exists for the contract package. | `server/internal/contract/openapi/generate.go` | Done | Not part of blocking validate or CI. | Add a future stale check without widening runtime ownership. |
| Backend generated output | Generated Go models are checked in under `server/internal/contract/openapi/generated/**`. | `server/internal/contract/openapi/generated/types.gen.go` | Done | Checked-in artifact can silently drift. | Add compare/regenerate validation in a follow-up slice. |
| Backend generation mode | Current output is `models: true` only; no strict server, server interfaces, client, or embedded spec. | `server/internal/contract/openapi/oapi-codegen.yaml` | Done | Team may still confuse this with full server-codegen rollout. | Keep documenting this as a constrained `types-only` path. |
| Frontend generation | `openapi-typescript` generates tracked schema types to `web/src/contracts/openapi/generated/schema.ts`. | `web/package.json`, `web/src/contracts/openapi/generated/schema.ts` | Done | None for the basic generator path. | Keep as the canonical frontend generated schema source. |
| Frontend stale gate | `openapi:types:check` regenerates to `.tmp` and compares against tracked output. | `web/package.json` | Done | Not automatically run by `bun run check`. | Decide whether to add it to `check` or CI in a later governance slice. |
| Web validation coverage | `bun run check` is the frontend completion entrypoint but does not directly call `openapi:types:check`. | `web/package.json`, `.github/workflows/pull-request-validation.yml` | Partial | TS generated schema drift may slip if typecheck/lint do not notice. | Add an explicit stale step in CI or `check`. |
| Backend generated-Go stale gate | No repo entrypoint compares regenerated Go types against tracked output. | Absence in `server/internal/cli/validate.go`, `.github/workflows/pull-request-validation.yml` | Missing | Highest drift risk in the current toolchain. | Add a narrow `go generate` + diff/compare gate in a future slice. |
| Local hooks | `pre-commit` runs lint-staged and contract governance; `pre-push` runs web hygiene and backend lint only. | `.husky/pre-commit`, `.husky/pre-push` | Partial | Hooks do not enforce OpenAPI-generated freshness. | Keep hook changes out of this audit; propose follow-up only. |
| CI coverage | CI runs contract governance, web check, backend lint, backend build/test. | `.github/workflows/pull-request-validation.yml` | Partial | OpenAPI generated drift is not a dedicated blocking job. | Add explicit OpenAPI generated drift checks later. |

## Backend Generated-Type Consumption Map

| Method | Path | In OpenAPI | Request Uses Generated Type | Response Uses OpenAPI Envelope | Error Contract | Handler Mapping | Test Coverage | Status |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| `GET` | `/healthz` | Yes | n/a | No; plain health payload by design | Minimal | Direct route handler | Covered by runtime validation, not OpenAPI DTO tests | Partial |
| `POST` | `/api/auth/login` | Yes | No; handwritten `loginRequest` | Yes via `httpx.WriteSuccess` | `400/401/500` style runtime contract exists | Direct bind + normalization in handler | Existing auth tests; not generated-type based | Partial |
| `POST` | `/api/auth/refresh` | Yes | n/a | Yes | `401/500` | Cookie-driven, no body DTO | Existing auth tests | Partial |
| `POST` | `/api/auth/logout` | Yes | n/a | Yes | `401/500` | Cookie-driven, no body DTO | Existing auth tests | Partial |
| `GET` | `/api/auth/bootstrap` | Yes | n/a | Yes | `401/500` | Direct service mapping | Existing auth tests | Partial |
| `POST` | `/api/users` | Yes | Yes; `PostUsersJSONRequestBody` | Yes | `400/401/403/500`, password policy example present | Generated request -> mapper -> command/service | Mapper tests in `server/plugins/user/plugin_test.go` plus route tests | Done |
| `POST` | `/api/users/{id}/update` | Yes | Yes; `PostUserUpdateJSONRequestBody` | Yes | `400/401/403/404/500` | Generated request -> mapper -> command/service | Mapper tests plus route tests | Done |
| `POST` | `/api/users/{id}/status` | Yes | Yes; `PostUserStatusJSONRequestBody` | Yes | `400/401/403/404/500` | Generated request -> enum normalization -> command/service | Mapper tests plus route tests | Done |
| `POST` | `/api/users/{id}/reset-password` | Yes | Yes; `PostUserResetPasswordJSONRequestBody` | Yes; empty envelope | `400/401/403/404/500`, field mapping handled by adapters | Generated request -> direct service call | Route/service tests | Done |
| `GET` | `/api/roles` | Yes | n/a | Yes | `401/403/500` | Read handler -> response DTO mapping | Existing RBAC tests | Partial |
| `POST` | `/api/roles` | Yes | Yes; `PostRolesJSONRequestBody` | Yes | `400/401/403/500` | Generated request -> normalize -> store input | Route tests and repository tests | Done |
| `POST` | `/api/roles/{id}/update` | Yes | Yes; `PostRoleUpdateJSONRequestBody` | Yes | `400/401/403/404/500` | Generated request -> normalize -> store input | Route tests and repository tests | Done |
| `POST` | `/api/roles/{id}/permissions/assign` | Yes | Yes; `PostRolePermissionAssignJSONRequestBody` | Yes; empty envelope | `400/401/403/404/500` | Generated request -> read IDs -> write service | Route tests and repository tests | Done |
| `GET` | `/api/permissions` | Yes | n/a | Yes | `401/403/500` | Read handler -> response DTO mapping | Existing RBAC tests | Partial |

Backend audit notes:

- `user` and `rbac` non-`auth` write routes have already moved to generated request types.
- `auth` write routes still intentionally keep handwritten request DTOs.
- OpenAPI uses `{id}` while Gin route contracts use `/:id`; current governance is explicit but manual, not auto-enforced.
- Error envelopes are runtime-owned by `server/internal/httpx/response.go`; generated types do not own runtime envelope emission.
- `409` is not consistently modeled as a first-class route-level response; several conflict-like cases still surface as `400`.

## Frontend Generated-Type Consumption Map

| Module | API Response Uses Generated Types | API Request Uses Generated Types | Form State Decoupled From DTO | Error Envelope Contracted | Handwritten Duplicate Types | Status |
| --- | --- | --- | --- | --- | --- | --- |
| `auth` | Yes for login/bootstrap response aliases | Yes for login payload; required-password-change payload still handwritten | Yes | Yes through `request.ts` | Small handwritten gap remains | Partial |
| `user` | Yes via `RawUserListItem` / `RawUserListResponse` aliases | Yes for create/update/status/reset-password payloads | Yes | Yes through `request.ts` + `user/error-adapter.ts` | Minimal; mostly intentional thin aliases | Done |
| `rbac` | Yes for permission and role list aliases; some binding response remains handwritten | Yes for create/update/assign payloads | Yes | Yes through `request.ts` + `rbac/error-adapter.ts` | `RolePermissionBindingResponse` still handwritten | Partial |
| platform request layer | Envelope unwrap is contract-aware, but not generated from OpenAPI | n/a | n/a | Yes; `ApiEnvelope` and `ApiErrorEnvelope` are runtime truth | Handwritten runtime envelope types remain by design | Done |

Frontend audit notes:

- Generated schema is used mainly as an API-boundary source, not as a page-state truth source.
- `web/src/utils/request.ts` remains the single transport/runtime truth, including token refresh, locale propagation, error normalization, and envelope unwrap.
- `readErrorField()` closes the `data.field` contract loop across modules.
- No generated client/runtime layer exists.

## Governance Progress

- Estimated progress: `50-75%`, best described as `about 60-70%`

Completed:

- Root OpenAPI spec exists and is split into reusable fragments.
- Shared success/error envelope baseline exists.
- Frontend generated schema flow is stable and checked in.
- Backend spec validation is wired into `graft validate backend`.
- Non-`auth` user/rbac write handlers already consume generated Go request types.
- Frontend `auth` / `user` / `rbac` modules already consume generated schema aliases at API boundaries.

Incomplete:

- No blocking Go generated-type stale gate.
- No explicit CI gate for frontend schema freshness.
- `auth` write interfaces remain handwritten DTO flows.
- OpenAPI coverage is still partial relative to the runtime surface.
- Error examples and conflict semantics are not uniformly modeled.
- No project-start docs-page loop exists.

High-risk gaps:

- Historical docs can be misread as "there is no `oapi-codegen` here" even though this branch now carries a scoped types-only path.
- OpenAPI `3.1.x` warning from `oapi-codegen` is known but not governance-enforced.
- Checked-in Go generated artifacts can silently drift.
- The root spec is validated, but generated outputs are not.

Recommended next-stage order:

1. Build the docs-page MVP loop without broadening DTO migration.
2. Add generated-Go and generated-TS freshness gates.
3. Tighten docs/spec exposure configuration for dev/test/prod.
4. Reassess whether `auth` write routes should ever join generated request adoption.

## Docs Page Options

### Option A: Server-served spec plus docs UI

- Suggested routes:
  - `GET /openapi.json`
  - `GET /docs`
- Characteristics:
  - no web build dependency
  - best fit for backend contract governance
  - lowest coupling for MVP
  - can be dev/test on by default and prod off by default
- Operational shape:
  - server serves current spec as JSON
  - server returns a minimal HTML page that loads a CDN-hosted UI

### Option B: Web-admin embedded docs page

- Suggested shape:
  - backend serves `/openapi.json`
  - web route such as `/developer/api-docs`
- Characteristics:
  - stronger long-term shell integration
  - more frontend coupling in the first slice
  - heavier than necessary for MVP

### Option C: Separate docs service

- Characteristics:
  - viable long-term docs-site pattern
  - wrong first step for current MVP

### UI Choice Comparison

| UI | Strength | Weakness | MVP Fit |
| --- | --- | --- | --- |
| Swagger UI | Mature, familiar, strong Try it out | Heavier look and more old-school UX | High |
| Scalar API Reference | Modern, lightweight, good as governance entry | CDN-page approach is less traditional | High |
| Redoc / Redocly | Strong reading experience | Weaker for interactive MVP path | Medium |

## Recommended Route

MVP recommendation:

- Choose `Option A`
- Prefer `Scalar` for the first HTML page
- Keep `Swagger UI` as a fallback if stronger interactive expectations outweigh the lighter presentation

Minimal closed loop:

1. Add `GET /openapi.json`
2. Add `GET /docs`
3. Dev/test enabled by default
4. Prod disabled by default, or guarded behind admin-only access if exposure is needed
5. Serve HTML directly from the Go server
6. Do not introduce Bun/Go package dependencies for the UI
7. Do not couple the first loop to the `web` shell

Why this is the MVP:

- It creates an immediately accessible contract-governance entrypoint after project startup.
- It does not disturb current `server` / `web` ownership lines.
- It avoids a generated client rollout, DTO migration, or frontend shell coupling.

## Validation Commands

Lightweight commands for this audit slice:

- `git diff --check`
- `git status --short`
- `cd web && bun run openapi:types:check`
- `cd server && go run ./cmd/graft validate backend --stage openapi`

Deferred or skipped heavier commands:

- `cd web && bun run check`
- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`

## Risks And Blockers

- The current branch has a real `oapi-codegen` types-only spike, but repository-wide governance still lacks a stable refresh gate for it.
- CI can currently miss generated artifact drift.
- `auth` remains intentionally outside the generated-request adoption path.
- The first docs-page implementation must avoid pulling in new dependencies or changing business handlers.

## Next Executable Slice

- Implement only the docs MVP closed loop:
  - `GET /openapi.json`
  - `GET /docs`
  - dev/test enabled by default
  - prod off by default or admin-guarded
  - server-served minimal HTML, no new dependencies

## Next-Session Startup Prompt

See `traces/openapi-codegen-governance-audit-trace.md` for the copy-ready startup prompt.

## Final Archive Record

- Status: `archived`
- Archive series: `ARCHIVED_OPENAPI_GOVERNANCE_SERIES`
- Archive reason: this audit completed its read-first mission and its recommendations were either implemented or superseded by later closeout topics.
- Final result: the audit remains as historical evidence for the docs MVP, bundled-spec fix, and final governance closeout decisions.
- Final validation summary:
  - docs-only archive update in this round
  - no server / web / openapi runtime files changed
  - backend and web validation not run in this round because the archive slice is documentation-only
- Follow-up status: `superseded`
- Superseded by:
  - `openapi-docs-mvp`
  - `openapi-docs-bundled-spec-fix`
  - `openapi-governance-closeout-audit`
- Next-session prompt: `No continuation required.`
- Archive note:
  - This topic is archived as part of the final OpenAPI/OAPI governance closeout. The relevant implementation, audit, generated-boundary, bundled-docs, monitor-coverage, and closeout findings have either been completed, superseded by later governance topics, or converted into guardrail-oriented follow-up guidance. No next-session recovery is required for this topic.
