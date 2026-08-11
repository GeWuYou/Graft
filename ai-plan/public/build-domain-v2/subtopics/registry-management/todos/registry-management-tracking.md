# Registry Management Subtopic Tracking

## Recovery Receipt

- governance source: root `AGENTS.md`
- task class: `cross-boundary`
- recovery source: `subtopic` under `build-domain-v2`
- authority summary: Registry owns Registry Connection, Artifact Repository and assignment facts; Build v2 owns only the non-secret destination tuple; Runtime Target owns execution; Credential Provider owns secret material.

## Phase 1 Objective

Deliver a page-verifiable Generic OCI Registry management slice without adding a parallel registry, repository, or credential model:

1. Registry Connection CRUD and status
2. Artifact Repository and user assignment management
3. Registry verification
4. Infrastructure Registry page
5. Build create selector: Registry Connection -> assigned Artifact Repository -> Tag

## Locked Boundaries

- Existing `registry_connections`, `artifact_repositories`, and `artifact_repository_user_assignments` remain canonical persistence facts.
- Build continues to submit `destination.connection_ref`, `destination.repository_ref`, and `destination.reference`; repository input is never free text.
- Credentials remain opaque `credential_ref` values. No HTTP API, table, plan, task, telemetry, audit record, or browser state receives a secret.
- This first slice does not alter the Build Runtime publication chain or add ECR, a managed Secret backend, private-network Registry trust zones, image lifecycle controls, or a default Registry.

## Batch Checklist

- [x] Add Registry management contract, migration, API, permission and verification seam.
- [x] Add Registry management Web module and Infrastructure route.
- [x] Replace the Build create page's hard-coded Registry connection with an assigned destination selector.
- [x] Run backend, OpenAPI and web verification; page browser verification is deferred because no local Graft runtime is listening.

## Validation Evidence

- Backend: `cd server && go test ./modules/registry/... ./internal/moduleregistry/... ./modules/rbac/...`, `python3 scripts/validate_sql_migrations.py`, `just openapi-check`, and `go run ./cmd/graft validate backend` passed before Web integration.
- Web: `cd web && bun run check` passed after the Registry management page and Build destination selector were added. The focused `src/modules/build/pages/create/index.test.ts` proves that Registry Connection and Repository options derive only from the actor-authorized destination list.
- Browser: no local Web/Backend process was listening on the repository's configured development ports during this slice. Browser screenshots and authenticated interaction remain a deployment-environment follow-up, not a substitute for the completed static and component-level validation.

## Next-Task Startup Prompt

`Read root AGENTS.md; task class: cross-boundary; recovery source: build-domain-v2/registry-management subtopic; owned scope: validate Registry management against a running primary-checkout Graft runtime, then address only observed credential-provider or Build Runtime publication gaps. Preserve server/modules/registry and Build v2 destination contract as authority; do not add parallel Registry, Repository, or Credential models.`
