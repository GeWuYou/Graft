# Registry Management Subtopic Trace

## 2026-08-11: Authority And Phase 1 Boundary

- The active `build-domain-v2` topic already owns Registry Connection and Artifact Repository authority, so Registry management is tracked as a subtopic rather than a parallel topic.
- The existing Build v2 destination contract is authoritative: `connection_ref`, `repository_ref`, and mutable `reference` are distinct non-secret facts. No `ImageRegistry`, `RegistryRepository`, or credential persistence model will be introduced.
- Phase 1 is a Generic OCI management and selection slice. Build Runtime publication, ECR, secret creation/rotation, private Registry trust zones, image retention and Harbor administration remain out of scope.

## 2026-08-11: Phase 1 Delivery Evidence

- Registry management uses the existing `registry_connections`, `artifact_repositories`, and `artifact_repository_user_assignments` facts. It adds no parallel image registry, repository, or credential entity.
- `/api/registries/available-destinations` is actor-filtered by assignment, enabled connection state, verified availability, and `allow_push`; Build creates continue to submit the existing destination tuple rather than a computed image string.
- The Infrastructure Registry page manages Generic OCI connections, non-secret credential configuration state, repository paths and their pull/push capabilities, assignments by user ID, and V2 reachability verification. Credential material and credential references are not displayed.
- The Build creation page uses a Registry Connection Select, a dependent Repository Select, and a Tag input; the repository field is no longer free text and no default connection is injected.
- Static, component, and authenticated browser validation completed. Public Registry verification remains environment-bound because outbound access returned the sanitized `network_failed` status.

## 2026-08-11: Assignment Integrity Repair

- Assignment grant now verifies the referenced user exists before any repository assignment write. This aligns the management API's documented `404` outcome with durable data and prevents unresolvable assignment rows.
- The check stays in Registry's SQL repository because it enforces a relationship to the platform identity table; no User module internal service is imported and no new credential or Registry model is introduced.
- Regression coverage verifies that a missing user returns `ErrNotFound` before writing. Existing route tests continue to cover the successful grant/revoke HTTP flow.

## 2026-08-14: Credential Catalog Authority And Publication Binding Repair

- Phase 2 authority discovery confirmed that `CredentialProvider` is currently limited to `Prepare`, `Inject` and `Revoke`; its file-backed source has no secret-free catalog or eligibility projection. Registry must not parse that source or create a parallel credential catalog.
- The existing Generic OCI Registry verifier only proves V2 reachability: a `401` is an expected challenge, not authenticated validation. Authenticated verification therefore requires a new CredentialProvider/Runtime Target seam that prepares, injects and revokes an isolated session.
- Runtime Target now checks complete, matching Registry-owned destination bindings before it requests ephemeral credentials for image push, manifest publication or artifact copy. This preserves `connection_ref`, `repository_ref` and `reference` as the Build v2 non-secret identity and prevents a mismatched result from causing credential issuance.
- Amazon ECR remains a provider-specific future extension. It needs only an ECR-aware CredentialProvider issuance/refresh and endpoint/repository-scope conformance path; it must not add Build-side branches or new Registry, Repository or Credential persistence models.

## 2026-08-14: Scoped Credential Eligibility

- `CredentialProvider.Assess` now evaluates one known opaque `credential_ref` against endpoint, repository and operation scope without issuing a session or enumerating the secret source. The only returned states are `eligible` and `ineligible`.
- The file-backed provider reloads and normalizes its deployment source for each assessment, applies the same scope rules as `Prepare`, and treats invalid scope or expiry as `ineligible`; source failures remain provider failures. `Prepare` remains the final signing boundary and repeats validation.
- No Registry, Build, Runtime Target execution, OpenAPI, Web, persistence or configuration schema changed in this batch. Runtime Target authorization remains outside CredentialProvider.
- Focused provider/runtime tests, the full server test suite, backend validation, the bounded ai-plan structure guard and diff whitespace validation passed. No live Registry authentication ran because this batch has no authenticated execution seam.

## 2026-08-14: Registry Authentication Verification Projection

- Registry now consumes the existing private Runtime Target Generic OCI verification seam. The verify request contains only an existing `repository_ref` and a Runtime Target identity; Registry resolves endpoint and opaque `credential_ref` from its own Connection/Repository facts and no secret or session crosses the HTTP boundary.
- Registry reuses `RuntimeTargetBuildAssignmentReader` to reject an actor who cannot use the selected target before calling `RuntimeExecutionAdapter`. The repository reference constrains the Credential Provider scope only; neither a successful nor failed verification claims repository pull/push authorization.
- The persisted and public result is restricted to `verified` / `failed`, time, and a stable sanitized error code. The OpenAPI `diagnostic` field was removed so headers, remote response text, source paths, endpoint credentials and session detail have no Registry verification projection path.
- The Registry Web page loads only existing Repository options and actor-authorized Build Runtime Targets for the selected Connection, then sends those two non-secret identifiers. UI text now describes authentication verification rather than connection availability or publish authorization.
