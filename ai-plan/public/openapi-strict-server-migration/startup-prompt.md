# Startup Prompt: OpenAPI Strict Server Migration

1. Run root preflight per `AGENTS.md` first. This prompt is a recovery entry, not an
   authority replacement.
2. Emit the receipt: `governance source: AGENTS.md`; `task class: cross-boundary`;
   `recovery source: ai-plan/public/openapi-strict-server-migration`;
   `authority summary: OpenAPI is the wire authority, internal/contract/openapi is the
   derived consumption surface, internal/httpx owns the envelope, module contract owns
   route and permission constants`.
3. Read `ai-plan/public/openapi-strict-server-migration/README.md` for scope and design
   summary, then `todos/openapi-strict-server-migration-tracking.md` for the current
   recovery point and batch state.
4. Owned scope and forbidden scope are listed in the topic README. Do not edit OpenAPI
   semantic sources, web, Justfile, workflows, or core route owners under
   `server/internal/**`.
5. Validation: `just openapi-check`,
   `cd server && go run ./cmd/graft validate backend` at milestones, and
   `python3 scripts/openapi_generated_freshness_check.py --target backend-openapi-servers`.
   Commits use conventional style with explicit paths; never `git add .`.
