# OAPI Codegen Types-Only Spike Tracking

## Topic

- Topic: `oapi-codegen-types-only-spike`
- Status: `completed and archived`
- Goal: run one isolated `oapi-codegen` Go types-only spike without changing backend runtime ownership, plugin lifecycle wiring, or frontend runtime/client truth.
- Recovery source: closed by `openapi-governance-closeout-audit` after the route-coverage closure reached archive-ready state
- Branch: `feat/wt-oapi-codegen-types-only-spike`

## Archive Closeout

- Final status:
  - current stage is `A + B + C + D`
  - `E` remains intentionally deferred
  - generated server/client remains intentionally deferred, not missing
- Final operating conclusion:
  - use generated Go/TS contract types plus thin consumer aliases where applicable
  - do not switch this repository to generated server/client as the default runtime pattern
  - future work uses contract-first feature delivery instead of reopening this spike as a standalone governance track
- Final closure evidence:
  - current backend business route coverage in scope is `29/29`
  - route coverage minimal gate exists at `server/internal/contract/openapi/route_coverage_test.go`
  - stale generated schema gate exists at `cd web && bun run openapi:types:check`
  - the archive closeout is recorded at `ai-plan/public/openapi-governance-closeout-audit/traces/openapi-governance-closeout-audit.md`
- Continuation rule:
  - no active recovery prompt remains for this topic
  - recommended next topic is `feature-delivery-with-contract-first-rule`

## Scope

- Owned scope:
  - `ai-plan/public/oapi-codegen-types-only-spike/**`
  - `server/go.mod`
  - `server/go.sum`
  - `server/internal/contract/openapi/**`
  - `server/plugins/user/**`
  - `server/plugins/rbac/**`
  - `web/src/modules/user/**`
  - `web/src/modules/rbac/**`
  - `web/src/contracts/api/**`
  - `web/src/modules/auth/**`
- Source spec:
  - `openapi/openapi.yaml`
- Generated output:
  - `server/internal/contract/openapi/generated/**`

## Repository Truth

- `AGENTS.md`
- `server/AGENTS.md`
- `ai-plan/design/AI任务追踪与恢复设计.md`
- `ai-plan/design/项目设计.md`
- `ai-plan/design/插件与依赖注入设计.md`
- `ai-plan/design/契约治理与魔法值治理规范.md`
- `ai-plan/roadmap/MVP实施计划.md`

## Current Recovery Point

- `openapi-contract-governance` and `write-interface-error-contract-standardization` are both complete and archived.
- The accepted baseline remains:
  - `spec-first + TS-first + explicit server DTOs`
  - `server/internal/httpx` stays the only backend envelope owner
  - `web/src/utils/request.ts` stays the only frontend transport/runtime owner
  - plugin-local handlers, DTOs, and route registration stay the runtime truth
- The covered write-route rollout already closed with `ready_for_oapi_codegen_types_only_spike: true`.
- This topic does not reopen runtime OpenAPI rollout. It only evaluates whether generated Go types can live in one clear non-runtime contract boundary.
- The initial scaffold spike already proved:
  - a pinned `oapi-codegen` tool can be wired through `server/go.mod` plus `go generate`
  - generated types can be isolated under `server/internal/contract/openapi/generated/**`
  - the generated package is compile-test consumable for `ApiEnvelope` and the `POST /api/users` request shape
- The same scaffold also exposed the current first-class risk:
  - `oapi-codegen` emits a warning because the repository root spec is OpenAPI `3.1.x`
  - generated request type naming follows route-oriented output such as `PostUsersJSONRequestBody`, not the handwritten runtime DTO names
- The approved execution scope is now wider than the original isolated boundary:
  - exclude all `auth` routes from this slice
  - include the remaining non-`auth` interfaces already covered by `openapi/openapi.yaml`
  - allow generated type adoption in selected `server` handlers and `web` schema consumers without changing backend/runtime ownership truth

## Shared Hotspots

- `ai-plan/public/README.md`

## Ownership Boundary

- Standing ownership still does not include `server/internal/httpx/**` or `openapi/**`.
- This slice now extends beyond the original `server/internal/contract/openapi/**`-only spike boundary into selected runtime consumers under `server/plugins/user/**`, `server/plugins/rbac/**`, `web/src/modules/user/**`, `web/src/modules/rbac/**`, `web/src/contracts/api/**`, and `web/src/modules/auth/**`.
- The slice may adopt generated types in runtime-adjacent handlers and frontend schema consumers, but it must not change service/store ownership, plugin lifecycle wiring, backend envelope ownership, or frontend transport/runtime ownership.
- The spike must not introduce `strict-server`, server stubs, generated client runtime, or a second backend DTO truth in runtime code.

## Active Risks

- `oapi-codegen` package placement may still be awkward even in a types-only path; if the generated package cannot stay isolated, the spike should fail closed with a no-go result.
- The root spec is shared across the current covered rollout, so the spike must consume the existing `openapi/openapi.yaml` without inventing a parallel reduced spec copy.
- Validation must stay aligned with `graft validate backend --stage openapi` and the full backend entrypoint; a custom one-off script is not an acceptable replacement.
- The widened slice must not blur the accepted contract baseline:
  - `server/internal/httpx` remains the only backend envelope owner
  - `web/src/utils/request.ts` remains the only frontend transport/runtime owner
  - `auth` routes remain out of scope even if generated types exist for some of them

## Immediate Next Step

- Migrate the remaining non-`auth` interfaces already covered by the root OpenAPI spec while keeping the `types-only` boundary intact.
- Treat the current target surface as:
  - `GET/POST /api/users`
  - `POST /api/users/{id}/update`
  - `POST /api/users/{id}/status`
  - `POST /api/users/{id}/reset-password`
  - `GET/POST /api/roles`
  - `POST /api/roles/{id}/update`
  - `POST /api/roles/{id}/permissions/assign`
  - `GET /api/permissions`
  - `GET /healthz`
- Keep all `auth` routes excluded from the current slice even when generated request types exist.

## Current Migration Shape

- Keep the current primary server chain narrow:
  - `generated request body -> handler thin binding/thin validation -> mapper -> command/service`
- The current non-`auth` migration target is no longer limited to the original three-route comparison sample.
- `server` migration in this slice means generated request-body type adoption for non-`auth` OpenAPI-covered write interfaces.
- `web` migration in this slice means generated schema type adoption or cleanup for non-`auth` OpenAPI-covered consumers; it does not mean generated client/runtime rollout.

## Auth Write Interface Classification Conclusion

- `POST /auth/login` does not join the current generated request body primary sample chain:
  - generated types exist (`PostAuthLoginJSONBody` / `PostAuthLoginJSONRequestBody`)
  - runtime still binds handwritten `loginRequest`
  - the handler performs route-local normalization and explicit empty-field validation before service entry
- `POST /auth/change-password` does not join the current generated request body primary sample chain:
  - runtime still binds handwritten `changePasswordRequest`
  - the route and service semantics remain heavier than the current thin-route sample definition
- `POST /auth/complete-required-password-change` is only a possible future candidate for a separate handwritten-DTO pure-write subgroup:
  - runtime still binds handwritten `completeRequiredPasswordChangeRequest`
  - the current generated output does not expose a matching generated request body
  - this topic does not create that subgroup in the current slice
- `POST /api/users/{id}/reset-password` is now in scope because it is non-`auth` and covered by the root OpenAPI spec.
- If follow-up work is needed after the current slice, keep any deeper `auth` migration as a separate secondary classification rather than broadening this non-`auth` execution scope.
