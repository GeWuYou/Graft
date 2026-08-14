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

## Phase 2 Assessment: Managed Credential Catalog And Authenticated Publication

- [x] Keep the Runtime Target execution boundary fail-closed: Generic OCI publish, manifest publication and artifact copy now reject incomplete or mismatched Registry-owned destination bindings before `CredentialProvider.Prepare`.
- [x] Add a CredentialProvider-owned, secret-free scoped eligibility projection. `Assess` accepts only a known opaque `credential_ref` plus endpoint/repository/operation scope and returns `eligible` or `ineligible`; Registry still cannot parse the deployment secret file or enumerate its entries.
- [x] Add authenticated Generic OCI verification through an isolated Runtime Target execution seam. The private result separately records reachability, OCI V2 compatibility, authentication challenge, V2 authentication success and CredentialProvider scope conformance; it does not claim repository pull or push authorization.
- [x] Make Registry the first consumer of that seam: the Verify API accepts only an existing `repository_ref` scope and an actor-authorized `runtime_target_id`; Registry resolves its private binding, records only a bounded authentication outcome, and Web never receives a secret or execution diagnostic.
- [x] Assess Amazon ECR: no Build destination, Registry Connection, Repository or Credential model extension is justified. An ECR-aware CredentialProvider may later issue scoped, short-lived ECR credentials and conform to the existing Runtime Target adapter boundary.

## Phase 2 Authority Decision

CredentialProvider owns a provider-approved, secret-free eligibility result for a known opaque `credential_ref`; Runtime Target owns target executability and actor-target authorization; Registry owns the persisted connection-level authentication outcome. Registry uses the selected Artifact Repository only as credential scope and never reinterprets the result as repository pull/push authorization.

## Validation Evidence

- Batch N CredentialProvider eligibility: `cd server && go test ./internal/credential ./internal/app ./modules/runtime-target`, `cd server && go test ./...`, `cd server && go run ./cmd/graft validate backend`, `python3 scripts/validate_ai_plan_structure.py`, and `git diff --check` passed. The backend entrypoint emitted only its existing OpenAPI 3.1 and DTO-boundary warnings.
- Backend: `cd server && go test ./modules/registry/... ./internal/moduleregistry/... ./modules/rbac/...`, `python3 scripts/validate_sql_migrations.py`, `just openapi-check`, and `go run ./cmd/graft validate backend` passed before Web integration.
- Web: `cd web && bun run check` passed after the Registry management page and Build destination selector were added. The focused `src/modules/build/pages/create/index.test.ts` proves that Registry Connection and Repository options derive only from the actor-authorized destination list.
- Browser: the primary-checkout runtime was verified at Web `127.0.0.1:3002` and server `127.0.0.1:8080`. The Registry Drawer opened under authenticated interaction; Connection `registry:phase1-ui-qa`, Repository `graft/phase1-ui-qa` and assignment to user `1` were created through the page, then deleted through the protected API in dependency order.
- Browser verification note: `https://registry-1.docker.io` reached the managed verify route but the local server's direct outbound request failed with the sanitized `network_failed` status. The data was cleaned up. This environment cannot provide a successful public-registry verification sample, so the actor-authorized Build selector's successful live option population remains covered by the focused component test and route/service tests rather than a fabricated database availability state.

## Current Recovery Point

The Registry authentication-verification projection is implemented and awaits an operator-provisioned human acceptance run with a real CredentialProvider entry and actor-authorized Runtime Target. No further implementation batch is currently defined inside this subtopic.
