# OpenAPI Contract Governance Tracking

## Closeout Status

- Status: completed and inactive.
- Recovery status: this topic is no longer part of the active recovery path.
- Final Phase 3 conclusion: keep generated schema types + module-local alias layers + existing `web/src/utils/request.ts`; do not introduce `openapi-fetch` or a second runtime client.
- Final Phase 4 conclusion: do not adopt `oapi-codegen` now; keep `spec-first + TS-first + explicit server DTOs`.
- Final validation evidence from the completed topic:
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend`
- Final closeout commits:
  - `3d7a16a docs(openapi-contract-governance): close phase 3 evaluation`
  - `3765d6a docs(openapi-contract-governance): close phase 4 evaluation`
- Topic-local archive snapshots:
  - `ai-plan/public/openapi-contract-governance/archive/todos/openapi-contract-governance-tracking-pre-closeout-2026-05-23.md`
  - `ai-plan/public/openapi-contract-governance/archive/traces/openapi-contract-governance-trace-pre-closeout-2026-05-23.md`
- Continuation rule: there is no next-session startup prompt for continuing `openapi-contract-governance`.
- Follow-up rule: future work must start as a new topic. See `ai-plan/public/openapi-contract-governance/traces/oapi-codegen-followup-evaluation.md`.

## Current State

- OpenAPI First governance for the current rollout scope is complete.
- The settled implementation baseline remains:
  - generated TypeScript schema types
  - module-local alias layers
  - existing `request.ts` transport/runtime truth
  - explicit `server` DTOs and explicit plugin-local route wiring
- Root `openapi/` spec plus fragments exist for the currently covered endpoints, including the bounded write-path rollout already modeled in this topic.

## Final Outcomes

- Phase 3 closed with no `openapi-fetch` rollout.
- Phase 4 closed with no `oapi-codegen` adoption.
- `oapi-codegen` remains deferred to a separate future topic, and only a types-only spike is worth reconsidering first.
- The detailed phase ledger and historical validation evidence now live in the topic-local archive snapshot instead of the active recovery file.
