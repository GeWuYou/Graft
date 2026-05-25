# OpenAPI Codegen Governance Audit Tracking

## Topic

- Topic: `openapi-codegen-governance-audit`
- Status: `audit complete`
- Goal: assess current `oapi-codegen` completion, OpenAPI governance maturity, and the minimum closed loop for an accessible docs page after project startup
- Branch: `feat/wt-oapi-codegen-types-only-spike`
- Recovery source: current repository state + `ai-plan/public`

## Scope

- Writable scope:
  - `ai-plan/public/openapi-codegen-governance-audit/**`
- Read-only inspected scope:
  - `openapi/**`
  - `server/**`
  - `web/**`
  - `scripts/**`
  - `.github/**`

## Current Conclusions

- The current branch already contains a constrained Go `oapi-codegen` types-only rollout.
- The repository-wide accepted baseline is still:
  - no generated server interfaces
  - no generated runtime client
  - `server/internal/httpx` remains the backend envelope owner
  - `web/src/utils/request.ts` remains the frontend transport/runtime owner
- Non-`auth` `user` and `rbac` write interfaces already consume generated Go request types.
- Frontend `auth` / `user` / `rbac` modules already consume generated schema aliases at API boundaries.
- OpenAPI governance maturity is materially advanced but not closed:
  - estimated `50-75%`
  - practical shorthand: `about 60-70%`

## Done In This Audit

- Mapped the OpenAPI source layout.
- Mapped the backend `oapi-codegen` generation path.
- Mapped the frontend generated-schema path.
- Classified backend runtime adoption of generated request types.
- Classified frontend runtime adoption of generated schema aliases.
- Audited hook / validate / CI coverage for generated drift.
- Compared docs-page option A/B/C and selected a recommended MVP route.
- Produced a next-session implementation prompt for the docs-page MVP slice.

## Missing Or Deferred

- No Go generated freshness gate.
- No explicit TS generated freshness gate in CI.
- No docs-page runtime implementation.
- No active-topic registration update in `ai-plan/public/README.md` from this slice because that file was outside the declared write scope.

## Recommended Next Slice

- Name: `openapi-docs-mvp`
- Shape:
  - implement `GET /openapi.json`
  - implement `GET /docs`
  - dev/test on by default
  - prod off by default or admin-guarded
  - no new dependencies
  - no web-shell coupling in the first slice

## Validation Targets

Expected lightweight validation for the audit slice:

- `git diff --check`
- `git status --short`
- `cd web && bun run openapi:types:check`
- `cd server && go run ./cmd/graft validate backend --stage openapi`

Expected heavier validation for the next implementation slice:

- `cd server && go test ./...`
- `cd server && go run ./cmd/graft validate backend`
- `cd web && bun run check` if any frontend or shared-contract runtime surface is touched

## Stop Condition

- This audit topic is complete as documentation.
- The next meaningful step is implementation under a new topic focused on the docs-page MVP.
