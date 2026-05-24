# OpenAPI Contract Governance Trace

## 2026-05-23 topic closeout

- Marked `openapi-contract-governance` as completed and inactive without moving the topic directory.
- Archived the detailed tracking and trace history into topic-local `archive/` snapshots, keeping the current files as concise closeout entries.
- Preserved the accepted final decisions in place:
  - Phase 3 stays on generated schema types + module-local alias layers + existing `request.ts`.
  - Phase 4 stays deferred/no-go for `oapi-codegen`; keep `spec-first + TS-first + explicit server DTOs`.
- Recorded the final validation evidence for the completed topic:
  - `cd web && bun run openapi:types:check`
  - `cd web && bun run check`
  - `cd server && go run ./cmd/graft validate backend --stage openapi`
  - `cd server && go run ./cmd/graft validate backend`
- Recorded the final closeout commits:
  - `3d7a16a docs(openapi-contract-governance): close phase 3 evaluation`
  - `3765d6a docs(openapi-contract-governance): close phase 4 evaluation`
- Added archive pointers:
  - `ai-plan/public/openapi-contract-governance/archive/todos/openapi-contract-governance-tracking-pre-closeout-2026-05-23.md`
  - `ai-plan/public/openapi-contract-governance/archive/traces/openapi-contract-governance-trace-pre-closeout-2026-05-23.md`
- Declared that there is no next-session startup prompt for continuing this topic.
- Added the deferred follow-up evaluation and next-topic recommendation at
  `ai-plan/public/openapi-contract-governance/traces/oapi-codegen-followup-evaluation.md`.
- Started the next user write-path payload slice for `POST /api/users/{id}/update`, `POST /api/users/{id}/status`, and `POST /api/users/{id}/reset-password`.
- Added root OpenAPI request schemas and path fragments for the user update, status update, and password reset write paths, keeping the existing server DTO shapes unchanged.
- Switched `web/src/modules/user/types/user.ts` to thin generated aliases for `UpdateUserPayload`, `UpdateUserStatusPayload`, and `ResetUserPasswordPayload`.
- Ran the Phase 2E user write payload rollout validation closeout for the `web` side.
- Rechecked `web/package.json` `openapi:types:check` against the current worktree and confirmed the repo-local `.tmp/` plus project-Prettier alignment fix is already present; no additional `web/package.json` or `.prettierignore` change was required in this slice.
- Validated the closeout state with `cd web && bun run openapi:types:check` and the full frontend entrypoint `cd web && bun run check`.
- Audited the current codebase against the original Phase 3 goal and confirmed the generated TypeScript consumption target is already satisfied by the existing `auth`, `user`, and `rbac` alias layers over `web/src/contracts/openapi/generated/schema.ts`.
- Confirmed `web/package.json` has no `openapi-fetch` dependency and no separate generated runtime client exists in the current web tree.
- Closed the optional lightweight client evaluation with a "do not introduce `openapi-fetch` now" result because `web/src/utils/request.ts` still owns the repository's token refresh, locale header propagation, envelope unwrap, auth-failure redirect, and API error normalization semantics.
- Recorded Phase 3 as complete without widening runtime scope, keeping generated schema aliases as the preferred API-boundary consumption pattern and leaving Phase 4 Go-generation evaluation unchanged.
- Revalidated the accepted Phase 3 closeout state with `cd web && bun run openapi:types:check`, `cd web && bun run check`, and `cd server && go run ./cmd/graft validate backend --stage openapi`.
- Audited the current codebase for Phase 4 and confirmed there is still no `oapi-codegen` dependency, no generated Go contract directory, and no generated server-interface wiring in the active backend validation or plugin lifecycle path.
- Used the current backend DTO layout in `server/plugins/user/dto_http_request.go` and `server/plugins/rbac/dto_http_request.go` as concrete evidence that introducing generated Go models now would create a second server-side DTO truth without a matching ownership or validation benefit.
- Closed Phase 4 as an evaluated deferred/no-go result for the current rollout scope: keep `spec-first + TS-first + explicit server DTOs`, do not let `oapi-codegen` take over server interfaces, and do not invent new generated-Go runtime wiring in this topic.
- Revalidated the accepted Phase 4 closeout state with `cd web && bun run check` and `cd server && go run ./cmd/graft validate backend`.
