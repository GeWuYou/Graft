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
- [x] Run backend, OpenAPI and web verification.
- [x] Verify the authenticated Infrastructure Registry page against the current primary-checkout runtime, including Connection creation, Repository creation, assignment, verify invocation and cleanup.

## Validation Evidence

- Backend: `cd server && go test ./modules/registry/... ./internal/moduleregistry/... ./modules/rbac/...`, `python3 scripts/validate_sql_migrations.py`, `just openapi-check`, and `go run ./cmd/graft validate backend` passed before Web integration.
- Web: `cd web && bun run check` passed after the Registry management page and Build destination selector were added. The focused `src/modules/build/pages/create/index.test.ts` proves that Registry Connection and Repository options derive only from the actor-authorized destination list.
- Browser: the primary-checkout runtime `feature/agent-bootstrap-security@af505ddf04f65fef23308234f287083f5c70d7c4` was verified at Web `127.0.0.1:3002` and server `127.0.0.1:8080`. The Registry Drawer opened under authenticated interaction; Connection `registry:phase1-ui-qa`, Repository `graft/phase1-ui-qa` and assignment to user `1` were created through the page, then deleted through the protected API in dependency order. The saved artifacts are `.ai/artifacts/browser/registry-drawer-repro-2` and `.ai/artifacts/browser/registry-phase1-managed`.
- Browser verification note: `https://registry-1.docker.io` reached the managed verify route but the local server's direct outbound request failed with the sanitized `network_failed` status. The data was cleaned up. This environment cannot provide a successful public-registry verification sample, so the actor-authorized Build selector's successful live option population remains covered by the focused component test and route/service tests rather than a fabricated database availability state.

## Next-Task Startup Prompt

`Read root AGENTS.md; task class: cross-boundary; recovery source: build-domain-v2/registry-management subtopic; owned scope: validate a successful Generic OCI verification and Build destination selection against a primary-checkout runtime that has approved outbound Registry access, then address only observed credential-provider or Build Runtime publication gaps. Preserve server/modules/registry and Build v2 destination contract as authority; do not add parallel Registry, Repository, or Credential models.`
