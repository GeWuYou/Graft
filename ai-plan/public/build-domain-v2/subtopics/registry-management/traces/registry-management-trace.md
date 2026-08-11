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
- Static/component validation completed. Browser evidence requires a running primary-checkout Graft runtime and remains the next environment-bound verification step.

## 2026-08-11: Assignment Integrity Repair

- Assignment grant now verifies the referenced user exists before any repository assignment write. This aligns the management API's documented `404` outcome with durable data and prevents unresolvable assignment rows.
- The check stays in Registry's SQL repository because it enforces a relationship to the platform identity table; no User module internal service is imported and no new credential or Registry model is introduced.
- Regression coverage verifies that a missing user returns `ErrNotFound` before writing. Existing route tests continue to cover the successful grant/revoke HTTP flow.
