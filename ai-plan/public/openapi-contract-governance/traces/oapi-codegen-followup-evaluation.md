# `oapi-codegen` Follow-up Evaluation

## Completed Decisions

- `openapi-contract-governance` is complete, archived in place, and inactive.
- The accepted current operating model remains:
  - OpenAPI spec-first
  - generated TypeScript schema types
  - module-local alias layers
  - existing `web/src/utils/request.ts` as the only transport/runtime truth
  - explicit `server` DTOs and explicit plugin-local Gin route wiring
- Phase 3 final state:
  - no `openapi-fetch`
  - no second runtime client
- Phase 4 final state:
  - no `oapi-codegen` adoption now
  - keep `spec-first + TS-first + explicit server DTOs`

## Deferred Options

### A. Current route

- Recommendation: keep as the default now.
- Fit with current constraints:
  - preserves `httpx` ownership of the success/error envelope
  - preserves the current error envelope fields: `success`, `code`, `message`, `messageKey`, `locale`, `traceId`, `data`
  - keeps plugin/module boundaries explicit
  - avoids a second Go DTO truth
  - keeps validation gates aligned with the current `graft validate backend` and `bun run check` flow
- Conclusion: keep.

### B. `oapi-codegen` types-only spike

- Recommendation: deferred, but this is the only future `oapi-codegen` path worth considering first.
- Preconditions:
  - write-interface error contract standardization is complete for at least one representative write route
  - OpenAPI write coverage is more hardened than the current bounded rollout
  - generated Go package placement is defined up front and kept isolated
- Expected scope if revisited:
  - generated Go types only
  - no generated server interfaces
  - no runtime handler wiring changes
- Conclusion: future isolated spike only.

### C. `strict-server` / server-stub spike

- Recommendation: defer; not planned.
- Reasons:
  - conflicts with explicit Gin route registration
  - conflicts with `httpx` envelope ownership
  - conflicts with plugin-local lifecycle and handler ownership
  - increases the chance of generated server interfaces becoming a second source of truth
- Conclusion: do not start with this path.

### D. Full server-side codegen adoption

- Recommendation: reject for now; not planned.
- Reasons:
  - generated Go models would currently duplicate handwritten DTOs in `server/plugins/user` and `server/plugins/rbac`
  - package layout and ownership would be unclear across plugin boundaries
  - would widen validation and regeneration workflow costs before contract hardening is complete
  - would likely standardize the wrong layer before write-interface error contracts are settled
- Conclusion: no adoption now.

### Overall Recommendation

- `recommendation: defer`
- `preferred_future_scope: types-only spike`
- Reason: generated Go server interfaces would currently fight the explicit plugin-local route/lifecycle model, and
  generated Go types would add DTO duplication before write-interface error contracts and OpenAPI write coverage are
  stable enough to justify it.

## Proposed Next Topic

### Recommended topic: `write-interface-error-contract-standardization`

- Reason:
  - the current repo already has the right first sample in `POST /api/users`
  - the route has runtime handler coverage, OpenAPI request/response coverage, a documented password-policy error
    example, and current `web` payload consumption via generated request aliases
  - standardizing write-route error contracts reduces second-source-of-truth risk before any Go codegen spike

### Scope

- Owned scope:
  - `ai-plan/public/write-interface-error-contract-standardization/**`
  - `openapi/**` only for `POST /api/users` and shared error-response components/examples needed by that route
  - `server/plugins/user/**` only for create-user error-contract alignment and focused tests
  - `web/src/modules/user/**` only for create-user error consumption and focused tests
  - `server/internal/contract/**` or `server/internal/httpx/**` only if required by the sample
- Forbidden scope:
  - no `oapi-codegen`
  - no generated Go code
  - no `request.ts` replacement
  - no broad multi-route sweep
  - no unrelated handler/service/router rewrites

### Deliverables

- standardized write-error contract decision for `POST /api/users`
- aligned OpenAPI error examples for `POST /api/users`
- aligned `server` and `web` sample behavior
- clear rollout criteria for expanding to more write routes

### Validation

- `cd server && go run ./cmd/graft validate backend --stage openapi`
- `cd server && go test ./plugins/user/...`
- `cd web && bun run openapi:types:check`
- `cd web && bun run check`

### Next-topic startup prompt

```text
governance source: root AGENTS.md
task class: cross-boundary
recovery source: none
owned scope:
  - ai-plan/public/write-interface-error-contract-standardization/**
  - openapi/** only for POST /api/users and shared error-response components/examples needed by that route
  - server/internal/contract/** or server/internal/httpx/** only if required by the sample
forbidden scope:
  - no oapi-codegen
  - no strict-server/server-stub generation
  - no request.ts replacement
  - no broad multi-route sweep
objective:
  Standardize the write-interface error contract starting with POST /api/users, then define rollout criteria for additional write routes.
```

### Optional later topic: `oapi-codegen-types-only-spike`

- Start only after `write-interface-error-contract-standardization` is complete.
- Owned scope:
  - `ai-plan/public/oapi-codegen-types-only-spike/**`
  - `server/go.mod`
  - `server/go.sum`
  - isolated generated package `server/internal/contract/openapi/generated/**`
  - optional compile/test-only comparison files under `server/internal/contract/openapi/**`
- Forbidden scope:
  - no `strict-server` or server interface generation
  - no plugin handler wiring changes
  - no replacement of explicit DTOs in runtime code
  - no `web` runtime/client changes
- Deliverables:
  - narrow generated-type spike for the shared error envelope plus `POST /api/users`
  - DTO clarity and package-ownership assessment
  - validation/regen workflow assessment
  - explicit go/no-go recommendation on whether any broader spike is justified
- Validation:
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - targeted `go test` for the isolated spike package
  - `cd server && go run ./cmd/graft validate backend`

## Non-Goals

- Do not reopen `openapi-contract-governance` as an active implementation topic.
- Do not introduce `oapi-codegen` in normal feature work without a dedicated governance topic.
- Do not introduce `strict-server`, server stubs, or a generated second runtime client.
- Do not replace `request.ts`.
- Do not broaden this follow-up into a full all-write-route contract rewrite.
