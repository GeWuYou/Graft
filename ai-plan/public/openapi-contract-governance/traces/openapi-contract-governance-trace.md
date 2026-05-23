# OpenAPI Contract Governance Trace

## 2026-05-23

- Created dedicated worktree `feat/wt-openapi-contract-governance`.
- Established public recovery topic `openapi-contract-governance`.
- Captured Phase 1 planning scope for OpenAPI First governance.
- Kept implementation untouched.
- Completed Phase 1.5 server boundary review for OpenAPI follow-up planning.
- Completed Phase 1.6 same-package lightweight file reorganization for `server/plugins/user` and `server/plugins/rbac`.
- Kept `package user` / `package rbac` unchanged and did not introduce subpackages, Go generated models, or OpenAPI files.
- Recorded Phase 1.6 as preparation for a future package-boundary refactor, not as the final server directory architecture.
- Completed Phase 2A minimal OpenAPI First baseline.
- Added root `openapi/` spec, path fragments, reusable schemas, security docs, and common error responses.
- Preserved the actual `/healthz` plain JSON contract instead of forcing it into the success envelope.
- Wired OpenAPI validation into `graft validate openapi` and `graft validate backend --stage openapi`, and inserted it at
  the front of the full backend validation chain.
- Kept `server/plugins/user` and `server/plugins/rbac` business logic unchanged.
- Kept `web/src/utils/request.ts` untouched and did not start generated TypeScript runtime consumption.
- Audited the existing Phase 2A partial diff and confirmed the owned scope stayed within `openapi/**`, backend OpenAPI
  validation wiring, and topic recovery docs.
- Repaired the root spec after `kin-openapi` rejected `info.summary` for the current validation path.
- Validated the repaired slice with `go run ./cmd/graft validate openapi`, `go run ./cmd/graft validate backend --stage openapi`,
  focused `go test ./internal/cli`, and `go build ./cmd/graft`.
- Completed Phase 2B minimal web TypeScript generation wiring with `openapi-typescript`.
- Added tracked generated output at `web/src/contracts/openapi/generated/schema.ts`.
- Added `web` scripts for generation and freshness checking without changing `request.ts` or consuming generated types in module APIs.
- Confirmed the generated file must be formatted after generation to satisfy the existing frontend Prettier gate.
- Validated Phase 2B with `bun run openapi:types`, `bun run openapi:types:check`, `bun run check`, and `go run ./cmd/graft validate backend --stage openapi`.
