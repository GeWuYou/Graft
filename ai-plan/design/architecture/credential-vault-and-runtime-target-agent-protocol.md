# Credential Vault And Runtime Target Agent Protocol RFC

## Status

Accepted design authority. This RFC implements [ADR-023](../decisions/ADR-023-runtime-target-agent-trust-model.md)
and [ADR-024](../decisions/ADR-024-runtime-target-agent-delivery-grant-binding.md), and defines the deployment
protocol required before Phase 4 dynamic Provider admission.

## Scope

`credential-vault` owns CSR-based PKI issuance and certificate evidence. Runtime Target owns Agent enrollment, binding
and operator lifecycle. `openapi/**` owns the wire contract. This document contains no implementation change.

## Credential Vault Contract

`credential-vault` owns opaque Credential References, Trust Bundles, certificate expiry, certificate revocation and
redacted PKI evidence. Runtime Target owns provider-neutral Agent Identity, pending enrollment, delivery-grant
authorization, receipt evidence, Agent binding and lifecycle state. Vault PKI owns certificate/private-key material and
cryptographic operations. Registry owns endpoint/repository policy and operation `credential_ref`.

Private keys, certificate PEM, CA private material, enrollment secrets and Registry plaintext are not stored in module
tables or exposed through `moduleapi`. The existing file-backed Registry provider may implement
`OperationCredentialIssuer`; it cannot implement `AgentEnrollmentAuthority` or `AgentCertificateIssuer`.

```go
type AgentEnrollmentAuthority interface {
    CreateEnrollment(context.Context, AgentEnrollmentRequest) (AgentEnrollment, error)
    ActivateGeneration(context.Context, AgentEnrollmentActivation) error
    RotateGeneration(context.Context, AgentEnrollmentRotationRequest) (AgentEnrollment, error)
    RevokeGeneration(context.Context, AgentEnrollmentRevocation) error
}

type AgentCertificateIssuer interface {
    IssueCSR(context.Context, AgentCertificateIssuanceRequest) (IssuedAgentCertificate, error)
    ReadTrustBundle(context.Context, TrustBundleRequest) (TrustBundleReference, error)
    RevokeCertificate(context.Context, AgentCertificateRevocation) error
}

type OperationCredentialIssuer interface {
    Assess(context.Context, CredentialEligibilityRequest) (CredentialEligibility, error)
    Prepare(context.Context, CredentialRequest) (EphemeralCredentialSession, error)
    Inject(context.Context, EphemeralCredentialSession, CredentialInjectionTarget) error
    Revoke(context.Context, EphemeralCredentialSession) error
}
```

`AgentEnrollment` contains only identity ID, generation, enrollment reference, expiry and non-secret trust metadata.
Runtime Target creates a pending Agent enrollment generation and an expiring delivery grant whose only token verifier is
`SHA-256(token || server-side pepper)`. Enrollment, identity, generation and certificate facts are provider-neutral.
For the only active MVP provider, ADR-024 section 3 reuses the existing authenticated deployment trust boundary for the
one-time grant handoff and Docker receipt. Runtime Target consumes its already verified automation identity and never
introduces a deployment trust bundle, automation client PKI, listener, or alternate authentication scheme. The receipt
is durable delivery evidence only: it cannot activate a generation or establish Agent membership. Runtime Target never
stores or returns the token, while Graft never receives an Agent private key.

## HTTP Protocol

The following are the intended operator operations, protected by existing bearer authentication, Runtime Target
permission, Target-level operator authorization and mandatory audit. They are internal protocol contracts until the
complete Docker-only conformance gate passes; they are not yet published in canonical `openapi/**` or generated
projections:

| Operation | Path | Result |
| --- | --- | --- |
| Create enrollment | `POST /api/runtime-targets/{id}/agent-enrollments` | `201` non-secret summary |
| Rotate generation | `POST /api/runtime-targets/{id}/agents/{agentId}/rotations` | `201` pending generation |
| Revoke generation | `POST /api/runtime-targets/{id}/agents/{agentId}/revocations` | `200`, idempotent |
| Read binding | `GET /api/runtime-targets/{id}/agents/{agentId}` | redacted status and diagnostics |

The enrollment summary includes `agent_id`, `generation`, `enrollment_ref`, delivery-grant status, `expires_at` and
`trust_bundle_version`, never bootstrap secret, delivery token, certificate, private key, endpoint credential,
Registry credential or Docker secret data.

The bootstrap listener is a separate server-authenticated TLS surface and is likewise unpublished. It accepts the
one-time token and CSR, then uses a database transaction to check the token verifier, pending unexpired delivery grant,
exact target/Agent Identity/generation, current Docker delivery evidence and CSR public-key fingerprint, and records a
stable issuance key before requesting Vault PKI issuance for the exact SPIFFE URI SAN. Vault issuance is external to
that transaction: timeout or
restart recovery queries or reconciles the same issuance key and must not create a second certificate. After verified
issuance, Runtime Target initializes the Agent's durable execution ledger from the new Agent binding before activation;
that ledger has no foreign-key dependency on retired Ed25519 telemetry identities. One database transaction then consumes
the token and grant, stores only serial/fingerprint/expiry and redacted evidence, and activates the generation. A repeated same-token, same-CSR completion returns the known non-secret result
or resumes issuance reconciliation; a changed CSR, expired or consumed token, revoked grant, or binding mismatch fails
closed.

The dedicated Agent listener accepts only `mutualTLS`, never bearer tokens or cookies. It is not published until
bootstrap, issuance, activation, revocation and ledger conformance pass. Identity is derived from the verified
certificate URI SAN and active generation, never a request body field.

| Operation | Path | Semantics |
| --- | --- | --- |
| Pull controller snapshot | `GET /agent/v1/ledger-snapshot` | returns one certificate-bound Driver-controller snapshot |
| Submit telemetry acknowledgement | `POST /agent/v1/telemetry-reports` | admits one issued snapshot as an observation |
| Claim execution lease | `POST /agent/v1/execution-leases/claim` | claims one Task-owned, capability-bound Stage attempt |
| Renew execution lease | `POST /agent/v1/execution-leases/{leaseId}/renew` | renews the same fenced attempt within its absolute deadline |
| Append bounded logs | `POST /agent/v1/execution-leases/{leaseId}/logs` | appends redacted Stage logs with sequence and size limits |
| Settle execution receipt | `POST /agent/v1/execution-leases/{leaseId}/receipts` | idempotently settles one bound Stage attempt |

Snapshots carry generation, sequence, snapshot ID/digest, observation and expiry timestamps, and canonical ledger
values. Reports echo generation, snapshot ID and digest and may contain only bounded liveness/implementation
diagnostics. Runtime Target validates identity, generation, issuance, expiry, digest and single use before storage.

`agent_identity_not_active` is `403`; `agent_generation_mismatch`, `agent_snapshot_replayed` and
`agent_snapshot_expired` are `409`; `agent_report_invalid` is `400`; and `agent_controller_unavailable` or
`credential_vault_unavailable` is `503`. All use the existing error envelope and stable message-key conventions.

## Ledger And Migration

Agent pull is the sole Phase 4 telemetry and work feed: snapshot issuance atomically allocates its monotonic sequence
and one-time ID, while execution claim atomically leases one Task-owned Stage attempt. The Agent cannot change ledger
values, synthesize its source, choose work outside its target/capability binding or settle a mismatched fence. Server push
and server-initiated streaming remain rejected because they would add connection-lifecycle authority. Expired or
unacknowledged snapshots are not placement input; expired running execution leases enter Task Runtime
`unknown`/`needs_attention` recovery and are never silently reassigned.

Execution endpoints extend the existing mTLS listener; they do not publish an operator API or create an Agent-owned
queue. Identity comes only from the active certificate generation. Claim filters are derived from the authenticated
target and capability binding, never request-body Agent identity. Lease payloads exclude secrets and host endpoints;
operation-scoped credentials use a separate one-time mTLS delivery boundary. Log and receipt submissions are bounded,
redacted and bound to task, stage, attempt, lease, fence and payload digest.

Migration creates generation-aware, provider-neutral Agent Identity, pending enrollment, delivery-grant,
delivery-receipt and certificate-evidence facts without reinterpreting Ed25519 public-key rows. Delivery grants bind
exactly one Runtime Target, Agent Identity and generation and transition only through `pending`, `delivered`, `consumed`
or `revoked`. Docker secret and container facts are receipt evidence for the current delivery implementation, not
enrollment identity fields. Legacy rows and observations remain readable with legacy provenance but are non-admissible;
old ingress receives no new reports. Dynamic policy stays disabled until selected Targets complete verified delivery,
Vault-issued mTLS enrollment and conformance. There is no dual acceptance. Rollback disables new binding and dynamic
admission; it never reactivates legacy trust.

## Delivery Order

| PR | Scope | Validation |
| --- | --- | --- |
| PR1 | narrow internal `moduleapi` contracts plus credential-vault and Runtime Target persistence seams; no published HTTP surface | focused service, migration and redaction tests |
| PR2 | verified automation delivery grant/receipt, server-authenticated bootstrap, Vault CSR issuance, durable activation and legacy disablement | delivery/replay/restart, audit and backend validation |
| PR3 | mTLS listener, Agent package, pull protocol, revocation and Docker ledger conformance | mTLS negatives, rotation/revocation/reconnect and Docker ledger proof |
| PR4 | publish the complete additive OpenAPI surface and regenerate server/Web projections | OpenAPI checks, generated-contract route coverage and full cross-boundary validation |

No PR publishes an Operator or Agent API before PR1 through PR3 complete as one Docker-only conformance slice. The
Docker limit applies to the implemented delivery path, not to the Runtime Target, Agent enrollment, identity, or
certificate contracts. PR3 uses a Vault-owned integration fixture for test certificates. An Agent creates and retains
its private key locally; it never self-signs or owns the CA.
