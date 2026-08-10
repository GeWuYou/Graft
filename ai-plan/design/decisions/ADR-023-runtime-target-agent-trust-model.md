# ADR-023: Runtime Target Agent Trust Model

- Status: accepted
- Date: 2026-08-08
- Scope: `credential-vault`, Build Domain v2 Docker Builder Agent enrollment, identity, telemetry report admission, packaging, and revocation

## Context

The Build Domain v2 credential and telemetry authority RFC makes Runtime Target the Docker Builder Agent control-plane
owner, but deliberately leaves the deployable trust protocol open. Phase 4 cannot admit dynamic placement until an Agent
is bound to exactly one Runtime Target, Docker Provider, Builder scope, and capability profile, and its reports are
authenticated, integrity-protected, replay-resistant, and revocable. This decision must not create a second Agent
registry, scheduler, telemetry store, or Build execution runtime.

## Decision

### 1. Managed trust backend

The reference managed backend is HashiCorp Vault PKI. Runtime Target's control plane authenticates to Vault using the
deployment's existing machine identity (Kubernetes auth, AppRole, or an operator-provided equivalent), never a static
CA private key in Graft configuration. Vault owns the Agent CA key and issues short-lived client certificates from a
dedicated intermediate CA. The control plane stores only the CA chain, certificate serial, enrollment generation,
public-key fingerprint, expiry and revocation state. Private keys remain in Agent service storage.

The backend boundary is `AgentTrustProvider`; Vault PKI is the first conformance implementation. A future managed
backend may replace it only by implementing the same issuance, rotation, revocation and audit contract. The existing
file-backed `CredentialProvider` for Registry credentials is unrelated and is not a trust backend or fallback for Agent
identity.

### 2. Identity and SAN/URI rules

Every enrollment generation has one stable `agent_id` and a monotonically increasing `generation`. The certificate URI
SAN is the stable workload identity:

`spiffe://graft/runtime-target/<target_id>/builder-agent/<agent_id>`

`target_id` and `agent_id` use their canonical textual encodings. DNS, IP and email SANs are omitted; endpoint names are
connection data, not identity. `generation` is server-side enrollment metadata associated through certificate evidence
and ledger/receipt evidence, not part of the SPIFFE workload identity. The control plane accepts a certificate only when its URI SAN, certificate issuer, serial,
public-key fingerprint, target binding, provider ID, scope ID and active generation metadata match the enrollment record.
URI parsing is exact and rejects aliases, prefixes, legacy `/generation/<generation>` URI forms, and a second identity field.

### 3. Enrollment and delivery

Enrollment is an operator-authorized Runtime Target operation. Runtime Target creates the pending enrollment with a
one-time bootstrap secret (minimum 32 random bytes), stores only a salted digest and a five-minute expiry, and requests
the opaque enrollment reference plus non-secret trust metadata from `credential-vault`. Vault-managed deployment delivery
materializes the bootstrap secret, client private key and certificate chain directly into the Agent installation. Build,
Task, browser, Runtime Target HTTP responses, telemetry reads, logs and execution evidence never carry those values.

The Agent reads its vault-delivered identity from an operator-owned `0700` state directory and never exports its private
key. It proves enrollment over TLS 1.3 server-authenticated bootstrap transport, then uses mutual TLS for all later
report and control traffic. The bootstrap secret is invalidated atomically after successful enrollment or expiry.
Re-enrollment and rotation create a new generation and require the old generation to be disabled first.

### 4. Transport and payload signatures

Agent-to-control-plane transport is TLS 1.3 mutual TLS using the enrolled certificate. The initial protocol is online
Agent pull and acknowledgement, so mTLS identity, active-generation validation, issued snapshot digest and single-use
issuance protect each report. Payload signatures for offline queued delivery remain deferred and require a separate
contract extension. The receiver rejects unknown fields, expired observations, clock skew beyond the existing bounded
limit, duplicate or non-monotonic sequences, digest mismatch and any binding mismatch before persistence or Build
projection.

### 5. Packaging and runtime hardening

The canonical distribution is a Graft-signed OCI image containing the pinned Agent binary and Docker CLI. It runs as a
dedicated non-root service with a read-only root filesystem, a `0700` state volume, no ambient credential mounts, and only
the explicitly bound Docker socket/API endpoint. The image digest and Agent implementation version are recorded in the
Runtime Target enrollment record. Package managers, ad-hoc binaries and host-wide service discovery are not enrollment
or trust authorities. Any systemd or Compose installation is a derived deployment wrapper that must preserve these
constraints and the same image digest.

### 6. Revocation and propagation

Disable, retirement, target rebinding, key rotation and suspected compromise immediately mark the enrollment generation
revoked in the Runtime Target control-plane store and append a redacted audit fact. The report ingress checks revocation
on every report. Revocation is propagated to Agent instances through the authenticated control channel and on reconnect;
the Agent stops reporting and execution on receipt. Ingress and provider readers may cache revocation state for at most 60
seconds and must fail closed when the cache is stale or the control store is unavailable. Certificates have a maximum
24-hour lifetime, so missed propagation cannot extend trust beyond the certificate expiry. Revocation resets no accepted
sequence for a still-active generation; a new generation starts at sequence zero only after explicit disable.

## Consequences

Phase 4 now has a testable trust boundary: Vault-backed issuance, exact URI identity, one-time enrollment, mTLS snapshot
acknowledgement, reproducible packaging, and bounded fail-closed revocation. Agent private keys and bootstrap
secrets stay outside Build and Task data. Deployments must operate Vault (or a conformant provider), retain a protected
Agent state volume, and test restart, rotation, revocation, snapshot replay and real Docker ledger reporting before dynamic
placement is enabled.

## Rejected Alternatives

- A shared static CA key or long-lived bootstrap token in Compose/configuration: it makes compromise global and cannot
  express generation revocation.
- DNS/IP SAN identity or endpoint-based identity: endpoints are mutable connection facts and permit rebinding ambiguity.
- Agent-local identity generation or returning bootstrap/private-key material through Runtime Target: it gives the
  control plane secret custody it does not own and defeats vault delivery boundaries.
- A generic signed HTTP ingress, Docker Engine metrics, or host metrics: none proves Agent scope, ledger provenance or
  controlled Builder capacity.
- Shipping host binaries or an ambient Docker credential store: it creates undeclared installation and secret authorities.
- Best-effort revocation or an unbounded cache: stale Agents could continue to influence dynamic placement after disable.
