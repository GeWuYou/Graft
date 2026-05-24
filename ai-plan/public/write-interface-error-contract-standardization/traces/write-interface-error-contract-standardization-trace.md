# Write-Interface Error Contract Standardization Trace

## 2026-05-23 `POST /api/users` sample closure in current worktree

- Bound this topic to the existing `feat/wt-openapi-contract-governance` worktree instead of creating a new worktree.
- Kept the accepted baseline unchanged:
  - `spec-first + TS-first + explicit server DTOs`
  - `web/src/utils/request.ts` remains the only frontend runtime transport truth
  - `server/internal/httpx` remains the backend envelope and localized-error owner
  - no `oapi-codegen`, `openapi-fetch`, TS runtime SDK, or generated Go server runtime wiring
- Standardized the `POST /api/users` sample decision that `data.field` uses the current request-contract field name.
- Narrowed the create-user password-policy error field from `new_password` to `password` at the user-plugin create route boundary only.
- Kept the mapping out of `httpx` and out of `request.ts`.
- Added the matching OpenAPI `400` examples for invalid argument and password-policy violation.
- Kept the follow-up conclusion unchanged: `oapi-codegen` is still deferred, and a future Go types-only spike is still blocked on broader write-interface hardening beyond this one sample.

## 2026-05-24 Phase 1 audit for covered write-route rollout

- Re-ran startup governance locally for this delegated round against root `AGENTS.md`, `.ai/environment/tools.ai.yaml`,
  `server/AGENTS.md`, `web/AGENTS.md`, `ai-plan/public/README.md`, and the active recovery traces.
- Audited the current covered rollout against the accepted `POST /api/users` sample and recorded the full matrix in
  `ai-plan/public/write-interface-error-contract-standardization/design/phase-1-audit-and-rollout-plan.md`.
- Confirmed the current Phase 1 status by route:
  - `POST /api/users`
    - aligned baseline
  - `POST /api/users/{id}/update`
    - partial: server field semantics exist, OpenAPI examples are generic, frontend update handling is still generic
  - `POST /api/users/{id}/status`
    - partial: server field semantics exist, OpenAPI examples are generic, frontend status handling is still generic
  - `POST /api/users/{id}/reset-password`
    - partial: server still emits `data.field=new_password`, which conflicts with the accepted request-field naming rule
  - `POST /api/roles`
    - partial: server field semantics exist, OpenAPI `400` is generic, frontend still uses generic error handling
  - `POST /api/roles/{id}/update`
    - partial: same gap pattern as role create, plus generic `404`
  - `POST /api/roles/{id}/permissions/assign`
    - partial: backend field semantics exist, OpenAPI write-error coverage is still incomplete, frontend still uses
      generic error handling
  - `POST /api/auth/login`
    - modeled but deferred from the rollout because it does not currently decide the write-route `data.field`
      convention
- Recorded the honest current readiness verdict:
  - `ready_for_oapi_codegen_types_only_spike: false`
- Locked the remaining loop plan as:
  - Phase 2: shared server-side field/data/error-contract alignment
  - Phase 3: OpenAPI error responses/examples alignment
  - Phase 4: web module-level structured field-error consumption alignment
  - Phase 5: focused validation closure and final readiness verdict docs

## 2026-05-24 Phase 2 server-side alignment evidence closure

- Re-checked the covered backend write routes against the actual request DTOs and route handlers before changing runtime
  code.
- Corrected the earlier Phase 1 audit note for `POST /api/users/{id}/reset-password`:
  - the current request-contract field name is already `new_password`
  - the existing backend `data.field=new_password` mapping is therefore aligned with the accepted rule that
    `data.field` must use the current request-contract field name
  - no `httpx` or plugin-runtime refactor was justified for this route in Phase 2
- Added focused user-plugin route tests proving the covered reset-password error behavior:
  - password-policy violation returns `AuthPasswordPolicyViolation` with `data.field=new_password`
  - password-reuse rejection returns `AuthPasswordReuseForbidden` with `data.field=new_password`
- Re-confirmed the covered RBAC write routes already have focused field-level error assertions for:
  - role create/update name conflicts -> `data.field=name`
  - role permission assignment invalid inputs -> `data.field=permission_ids`
- Phase 2 conclusion:
  - the covered server-side write-error contract surface is aligned without broadening ownership or introducing new
    abstractions

## 2026-05-24 Phase 3 OpenAPI alignment closure

- Reconciled the stale Phase 1 reset-password audit note with the Phase 2 evidence:
  - `POST /api/users/{id}/reset-password` already follows the request-contract field rule with `data.field=new_password`
  - the remaining gap for that route was OpenAPI/web alignment, not backend runtime behavior
- Updated covered OpenAPI write-route responses/examples to match the tested backend contracts:
  - `POST /api/users/{id}/update`
    - concrete `400` invalid-argument example for `data.field=username`
    - concrete `404` `user.not_found` example
  - `POST /api/users/{id}/status`
    - concrete `400` invalid-argument example for `data.field=status`
    - concrete `404` `user.not_found` example
  - `POST /api/users/{id}/reset-password`
    - concrete `400` examples for `AuthPasswordPolicyViolation` and `AuthPasswordReuseForbidden`
    - both examples use `data.field=new_password`
    - concrete `404` `user.not_found` example
  - `POST /api/roles`
    - concrete `400` invalid-argument example for `data.field=name`
  - `POST /api/roles/{id}/update`
    - concrete `400` invalid-argument example for `data.field=name`
    - concrete `404` `role.not_found` example
  - `POST /api/roles/{id}/permissions/assign`
    - added missing `400` and `404` response entries
    - concrete `400` invalid-argument example for `data.field=permission_ids`
    - concrete `404` `role.not_found` example
- Kept shared envelope semantics unchanged:
  - `success`, `code`, `message`, `messageKey`, `locale`, `traceId`, `data`
- Left `POST /api/auth/login` unchanged in Phase 3 because the currently modeled auth route does not decide the covered
  write-route field-error convention and no shared-envelope mismatch was found during this phase

## 2026-05-24 Phase 4 web alignment attempt and validation blocker

- Implemented bounded module-local write-error consumption for the covered `user` and `rbac` pages without moving field
  semantics into `request.ts`:
  - `web/src/modules/user/**`
    - extended the existing user error adapter so create and edit forms consume `data.field`
    - bound reset-password API field errors to the dialog password field using the route contract field
      `new_password -> password`
    - switched covered status-update failures to use backend `messageKey` / message fallback instead of a generic
      module toast when the API returns a structured error such as `user.not_found`
  - `web/src/modules/rbac/**`
    - added a module-local error adapter for role form and permission-assignment write errors
    - bound covered role create/update invalid-argument errors to the role form field surface
    - kept `permission_ids` and `role.not_found` assignment failures inside the permission drawer feedback surface
      instead of collapsing them into a generic transport-level toast
- Added focused page-test evidence for those covered routes:
  - `src/modules/user/pages/index.test.ts`
    - edit-field invalid argument
    - reset-password password-policy violation
    - status-update not-found feedback
  - `src/modules/rbac/pages/index.test.ts`
    - role form `name` field invalid argument
    - permission assignment `permission_ids` invalid argument
    - permission assignment `role.not_found` feedback
- Validation results for the Phase 4 attempt:
  - passed
    - `git diff --check`
    - `cd web && bun run test:run src/modules/user/pages/index.test.ts`
    - `cd web && bun run test:run src/modules/rbac/pages/index.test.ts`
    - route-surface `rg` consistency scans across touched `web` and topic-trace files
  - blocked
    - `cd web && bun run openapi:types:check`
    - blocker detail:
      - the generated `web/src/contracts/openapi/generated/schema.ts` is behind the Phase 3 OpenAPI change for
        `POST /api/roles/{id}/permissions/assign`
      - the generated diff adds the newly modeled `400` and `404` response shapes for that operation
      - regenerating that generated file would be the correct repair path, but it is outside this delegated Phase 4
        round's owned scope
- Phase 4 status at this checkpoint:
  - implementation evidence exists
  - phase validation is not yet complete because the required generated-type check is red
  - no Phase 4 commit was created in this round

## 2026-05-24 Phase 4 validation unblock and closure

- Resumed the blocked Phase 4 round within the same owned scope and used the repository generation entrypoint:
  - `cd web && bun run openapi:types`
  - no hand edits were made to `web/src/contracts/openapi/generated/schema.ts`
- The regenerated TypeScript contract now matches the Phase 3 OpenAPI surface for
  `POST /api/roles/{id}/permissions/assign`:
  - includes the modeled `400` invalid-request response shape
  - includes the modeled `404` role-not-found response shape
- Revalidated the existing Phase 4 web implementation without broadening runtime ownership:
  - passed
    - `git diff --check`
    - `git status --short`
    - route-surface `rg` consistency scans across touched `web` and trace files
    - `cd web && bun run openapi:types:check`
    - `cd web && bun run test:run src/modules/user/pages/index.test.ts src/modules/rbac/pages/index.test.ts`
    - `cd web && bun run check`
- Phase 4 final conclusion:
  - covered `user` and `rbac` write routes now consume structured field errors in module-local surfaces
  - `request.ts` remains the transport truth
  - backend envelope ownership remains in `httpx`
  - generated OpenAPI web types are aligned with the covered write-route contract surface

## 2026-05-24 Phase 5 validation closure and readiness verdict

- Executed the final cross-boundary validation closure for the covered rollout in the current worktree:
  - `git diff --check`
  - `git status --short`
  - `rg` consistency scans across topic docs, `openapi`, `server/plugins/user`, `server/plugins/rbac`,
    `web/src/modules/user`, and `web/src/modules/rbac`
  - `cd server && go test ./plugins/user/...`
  - `cd server && go test ./plugins/rbac/...`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run test:run src/modules/user/pages/index.test.ts src/modules/rbac/pages/index.test.ts`
  - `cd web && bun run check`
- Confirmed the covered route verdict after the full validation rerun:
  - `POST /api/users`
    - aligned
  - `POST /api/users/{id}/update`
    - aligned
  - `POST /api/users/{id}/status`
    - aligned
  - `POST /api/users/{id}/reset-password`
    - aligned
  - `POST /api/roles`
    - aligned
  - `POST /api/roles/{id}/update`
    - aligned
  - `POST /api/roles/{id}/permissions/assign`
    - aligned
  - `POST /api/auth/login`
    - still deferred, but no longer a blocker for the covered write-route readiness judgment
- Reconfirmed the final contract-governance constraints stayed intact:
  - canonical envelope semantics remain `success`, `code`, `message`, `messageKey`, `locale`, `traceId`, `data`
  - `server/internal/httpx` remains the only backend envelope owner
  - `web/src/utils/request.ts` remains the only frontend transport/runtime owner
  - plugin-local handlers and DTOs remain the backend runtime truth
  - module-local adapters remain the frontend owner of field-surface error handling
- Final topic verdict:
  - `ready_for_oapi_codegen_types_only_spike: true`
  - no remaining blocker was found inside the accepted covered rollout for a future isolated Go types-only evaluation
