# Credential Vault And Runtime Target Agent Protocol RFC

## Status

Accepted design authority. This RFC implements [ADR-023](../decisions/ADR-023-runtime-target-agent-trust-model.md)
and defines the deployment protocol required before Phase 4 dynamic Provider admission.

## Scope

`credential-vault` owns credential lifecycle semantics. Runtime Target owns the Agent binding and operator lifecycle.
`openapi/**` owns the wire contract. This document contains no implementation change.

## Credential Vault Contract

`credential-vault` owns opaque Credential References, Machine Identities and their generations, Enrollment Grants,
Trust Bundles, expiry, rotation, revocation and redacted evidence. Vault PKI owns certificate/private-key material and
cryptographic operations. Runtime Target stores only Agent binding, identity reference, generation, certificate
fingerprint, status and audit correlation. Registry owns endpoint/repository policy and operation `credential_ref`.

Private keys, certificate PEM, CA private material, enrollment secrets and Registry plaintext are not stored in module
tables or exposed through `moduleapi`. The existing file-backed Registry provider may implement
`OperationCredentialIssuer`; it cannot implement `MachineIdentityAuthority`.

```go
type MachineIdentityAuthority interface {
    CreateEnrollment(context.Context, MachineEnrollmentRequest) (MachineEnrollment, error)
    ActivateGeneration(context.Context, MachineIdentityActivation) error
    RotateGeneration(context.Context, MachineIdentityRotationRequest) (MachineEnrollment, error)
    RevokeGeneration(context.Context, MachineIdentityRevocation) error
    ReadTrustBundle(context.Context, TrustBundleRequest) (TrustBundleReference, error)
}

type OperationCredentialIssuer interface {
    Prepare(context.Context, CredentialRequest) (EphemeralCredentialSession, error)
    Inject(context.Context, EphemeralCredentialSession, CredentialInjectionTarget) error
    Revoke(context.Context, EphemeralCredentialSession) error
}
```

`MachineEnrollment` contains only identity ID, generation, enrollment reference, expiry and non-secret trust metadata.
Vault-managed deployment delivery materializes private-key and enrollment material for the Agent; Runtime Target never
receives it.

## HTTP Protocol

Operator endpoints are protected by existing bearer authentication, Runtime Target permission, Target-level operator
authorization and mandatory audit:

| Operation | Path | Result |
| --- | --- | --- |
| Create enrollment | `POST /api/runtime-targets/{id}/agent-enrollments` | `201` non-secret summary |
| Rotate generation | `POST /api/runtime-targets/{id}/agents/{agentId}/rotations` | `201` pending generation |
| Revoke generation | `POST /api/runtime-targets/{id}/agents/{agentId}/revocations` | `200`, idempotent |
| Read binding | `GET /api/runtime-targets/{id}/agents/{agentId}` | redacted status and diagnostics |

The enrollment summary includes `agent_id`, `generation`, `enrollment_ref`, `expires_at` and
`trust_bundle_version`, never bootstrap secret, certificate, private key, endpoint credential, Registry credential or
Docker data.

The dedicated Agent listener accepts only OpenAPI `mutualTLS`, never bearer tokens or cookies. Identity is derived from
the verified certificate URI SAN and active generation, never a request body field.

| Operation | Path | Semantics |
| --- | --- | --- |
| Pull controller snapshot | `GET /agent/v1/ledger-snapshot` | returns one certificate-bound Driver-controller snapshot |
| Submit telemetry acknowledgement | `POST /agent/v1/telemetry-reports` | admits one issued snapshot as an observation |

Snapshots carry generation, sequence, snapshot ID/digest, observation and expiry timestamps, and canonical ledger
values. Reports echo generation, snapshot ID and digest and may contain only bounded liveness/implementation
diagnostics. Runtime Target validates identity, generation, issuance, expiry, digest and single use before storage.

`agent_identity_not_active` is `403`; `agent_generation_mismatch`, `agent_snapshot_replayed` and
`agent_snapshot_expired` are `409`; `agent_report_invalid` is `400`; and `agent_controller_unavailable` or
`credential_vault_unavailable` is `503`. All use the existing error envelope and stable message-key conventions.

## Ledger And Migration

Agent pull is the sole Phase 4 feed: snapshot issuance atomically allocates its monotonic sequence and one-time ID.
The Agent cannot change ledger values or synthesize its source. Server push and streaming are rejected because they
would add connection-lifecycle authority. Expired or unacknowledged snapshots are not placement input.

Migration creates generation-aware Agent identity and enrollment tables without reinterpreting Ed25519 public-key rows.
Legacy rows and observations remain readable with legacy provenance but are non-admissible; old ingress receives no new
reports. Dynamic policy stays disabled until selected Targets complete vault-issued mTLS enrollment and conformance.
There is no dual acceptance. Rollback disables new binding and dynamic admission; it never reactivates legacy trust.

## Delivery Order

| PR | Scope | Validation |
| --- | --- | --- |
| PR1 | additive experimental OpenAPI and `moduleapi` contracts; no listener or activation | OpenAPI checks and generated-contract checks |
| PR2 | credential-vault module, adapter seam, metadata migration, operator APIs and legacy disablement | migration, redaction/audit tests, backend validation |
| PR3 | mTLS listener, Agent package, pull protocol, conformance and old-ingress removal | mTLS negatives, rotation/revocation/reconnect and Docker ledger proof |

PR3 uses a vault-owned integration fixture for test certificates. An Agent never self-signs or owns that CA.
