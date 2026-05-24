# OAPI Codegen Types-Only Spike Tracking

## Topic

- Topic: `oapi-codegen-types-only-spike`
- Status: `active recovery entry`
- Goal: run one isolated `oapi-codegen` Go types-only spike without changing backend runtime ownership, plugin lifecycle wiring, or frontend runtime/client truth.
- Recovery source: new standalone topic after archiving two completed OpenAPI governance topics
- Worktree: `/home/gewuyou/project/go/Graft-wt/feat/wt-oapi-codegen-types-only-spike`
- Branch: `feat/wt-oapi-codegen-types-only-spike`

## Scope

- Owned scope:
  - `ai-plan/public/oapi-codegen-types-only-spike/**`
  - `server/go.mod`
  - `server/go.sum`
  - `server/internal/contract/openapi/**`
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

## Shared Hotspots

- `ai-plan/public/README.md`

## Ownership Boundary

- Standing ownership does not include `server/plugins/**`, `server/internal/httpx/**`, `web/**`, or `openapi/**`.
- The spike may add comparison-only code under `server/internal/contract/openapi/**`, but it must not push generated types into runtime handler, service, store, or plugin lifecycle paths.
- The spike must not introduce `strict-server`, server stubs, generated client runtime, or a second backend DTO truth in runtime code.

## Active Risks

- `oapi-codegen` package placement may still be awkward even in a types-only path; if the generated package cannot stay isolated, the spike should fail closed with a no-go result.
- The root spec is shared across the current covered rollout, so the spike must consume the existing `openapi/openapi.yaml` without inventing a parallel reduced spec copy.
- Validation must stay aligned with `graft validate backend --stage openapi` and the full backend entrypoint; a custom one-off script is not an acceptable replacement.

## Immediate Next Step

- Finish the topic-governance cutover by renaming the dedicated branch/worktree pair and moving the two completed predecessor topics into `ai-plan/public/archive/`.
- Record the first honest spike verdict:
  - generation is feasible in an isolated non-runtime boundary
  - OpenAPI 3.1 support warning remains a material caveat
  - current generated request naming and package shape are not yet evidence for replacing handwritten runtime DTOs
- Decide in the next bounded slice whether to stop at a documented no-go/defer result or add deeper comparison-only evaluation under the same isolated boundary.
