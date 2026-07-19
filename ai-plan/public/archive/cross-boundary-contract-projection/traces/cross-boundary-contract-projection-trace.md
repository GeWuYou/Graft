# Cross-Boundary Contract Projection Trace

## 2026-07-19 Batch 0: intake, authority and roadmap

- Work Intake classified the effort as long-running `refactor`; it needs an active topic, repository-wide design and roadmap, but not an ADR.
- Locked the authority split: OpenAPI owns HTTP wire schemas, paths and public wire enums; existing Go server contracts own non-HTTP stable values; web consumes generated artifacts only.
- Locked projection metadata as an index that references existing typed Go constants. It includes explicit owner/lifecycle/visibility and does not repeat values.
- Locked API compatibility: error code and message key stay open strings and web must retain fallback for unknown future values.
- Locked runtime boundary: generated permission/menu/capability values cannot replace server bootstrap or runtime availability decisions.
- Defined Phase 1 generation/drift baseline, Phase 2 pilot migration and Phase 3 broad convergence.
- Committed this intake batch after `git diff --check` and the bounded ai-plan structure guard passed.

## Locked Decisions

- Do not introduce protobuf, a shared runtime package, a new hand-written IDL, or a second contract authority.
- Preserve existing OpenAPI generation; projection follows it and reuses its artifacts.
- Run generation in authority order and block uncommitted derived output, duplicate semantic ownership, invalid visibility and lifecycle drift.

## 2026-07-19 Batch 1: generator foundation

- Added `server/internal/contract/projection` as a metadata index and renderer, not a new contract authority. `Registry` values reference existing `errorcode.Code`, `message.Key`, `httpheader.Name`, and `auth.Scheme` constants.
- Added an explicit Go generator with write and `--check` modes. Check mode renders in memory and compares the tracked `web/src/contracts/generated/platform.ts` artifact without writing it.
- Generated platform error code, message key, HTTP header, and auth-scheme literal objects plus union types. An internal-only header descriptor proves visibility filtering and is absent from web output.
- Added focused validation for deterministic generation order, duplicate/lifecycle metadata rejection, visibility filtering, and AST verification that registry `Value` fields are existing typed constant selectors rather than copied literals.
- Updated `just generate` to run OpenAPI bundle before Go bindings, then web OpenAPI types and cross-boundary projection. `just openapi-check` now includes projection freshness; PR CI wiring remains the later `ci-integration` batch.

## 2026-07-19 Batch 2: platform pilot migration

- Expanded the platform projection registry with every value previously hand-written in `web/src/contracts/api/codes.ts`, `messages.ts`, and `headers.ts`. Each descriptor directly references an existing Go typed constant; no canonical literal was copied into the registry.
- Regenerated `web/src/contracts/generated/platform.ts` and replaced the three API contract files with thin compatibility exports. Existing `API_CODE`, `ApiCode`, `ApiResponseCode`, `MESSAGE_KEY`, `MessageKey`, `AUTH_SCHEME`, and `HTTP_HEADER` import names remain available.
- `ApiResponseCode` remains `string`, so server-first new response codes remain compatible. Generated literal unions are only static web references.
- Added focused web coverage that checks compatibility exports are the generated objects and statically rejects projected value literals in the compatibility files.
- Validation passed: focused Go projection tests, generator freshness check, focused web Vitest suite, web typecheck, web lint, formatting check, and `git diff --check`.

## 2026-07-19 Batch 3: CI integration

- Added a blocking PR job for the existing cross-boundary freshness chain: backend OpenAPI validation and generated artifact checks, web OpenAPI schema comparison, then Go-to-TypeScript projection comparison. The job composes existing commands and does not create another generator or scanner.
- Updated pre-push to run the same chain only when OpenAPI authority, Go cross-boundary contract owners, generated web contracts, or their generation entrypoints change. Pure consumer and unrelated changes keep the existing narrower hook path.
- Kept `bun run check` unchanged: the explicit PR job remains the ownership point for generated-output freshness, avoiding a redundant schema generation pass in every full web validation.
- Validation passed: `just openapi-check`, `git diff --check`, the active-topic structure guard, workflow YAML parsing and pre-push shell syntax validation. The existing backend boundary audit remains `PASS_WITH_WARNINGS` for three pre-existing runtime-marker warnings.

## 2026-07-19 Batch 4: broader migration and archive readiness

- Extended the projection generator from a platform-only output to explicit generated targets. Each target remains a derived artifact; `platform.ts` does not absorb module values.
- Added the container target with canonical references to all container permission constants, five realtime topic constants and all Docker image remove error constants. The existing plain-string realtime constants remain their server runtime authority; the projection supports both string and typed-string constants without copying values.
- Generated `web/src/contracts/generated/modules/container.ts`, replaced all container permission and Docker error hand mirrors, and changed realtime helpers to consume the generated topic object while retaining only parsing and topic-building behavior locally.
- Final acceptance passed: `graft validate backend`, `bun run check`, `just openapi-check`, focused projection/container tests and `git diff --check`. The backend boundary audit retains three pre-existing runtime-marker warnings and reports no violation.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["batch-0-contract-projection-intake", "generator-foundation", "pilot-migration", "ci-integration", "broader-migration-and-final-archive-readiness"],
  "pending_batches": [],
  "current_batch": "broader-migration-and-final-archive-readiness",
  "next_batch": null,
  "closeout_status": "archive-ready"
}
```
