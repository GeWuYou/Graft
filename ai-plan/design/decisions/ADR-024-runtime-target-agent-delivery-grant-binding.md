# ADR-024: Runtime Target Agent Delivery Grant Binding

- Status: accepted
- Date: 2026-08-09
- Scope: Runtime Target Agent bootstrap enrollment delivery, Agent generation binding, and Vault PKI issuance authorization; Docker is the only MVP delivery implementation

## Context

ADR-023 establishes one-time Agent enrollment, exact SPIFFE identity, Vault PKI issuance, and mutual TLS, but its
"Vault-managed deployment delivery" wording does not define how Graft can prove that deployment automation delivered
the intended bootstrap material to the intended Agent installation. Certificate issuance alone is not a Runtime
Target authorization decision. Without a durable delivery grant and a bootstrap-time binding check, a valid certificate
could be presented without proving its intended Runtime Target, Agent, and generation association.

This decision completes that delivery boundary while retaining the existing ownership model: Runtime Target owns
business authorization and binding; `credential-vault` owns PKI issuance; deployment automation owns the active
provider's bootstrap-material delivery lifecycle; and the Agent owns its private key. Docker secret lifecycle is the
only delivery mechanism implemented by this MVP. It supersedes ADR-023 section 3 only where ADR-023 assigns delivery
materialization to Vault. All other ADR-023 trust, identity, transport, packaging, and revocation decisions remain in
force.

## Decision

### 1. Provider-neutral platform model; Docker-only delivery implementation

`RuntimeTarget`, `AgentIdentity`, `AgentEnrollment`, generation, certificate identity, and their lifecycle are
platform concepts. They must not be named, persisted, or exposed as Docker-specific entities. A delivery grant binds a
Runtime Target to one Agent Identity and generation; a certificate remains an Agent Identity fact rather than a Docker
container fact.

Docker is the only active Runtime Target provider and Docker secret injection is the only delivery mechanism in this
MVP. Kubernetes, Podman, SSH, generic provider adapters, future provider packages, placeholders, menus, APIs, and
unsupported-runtime logic are out of scope. This preserves a provider-neutral authority without promising behavior for
providers that are not implemented or activated.

### 2. Pending enrollment and delivery grant

An authorized Runtime Target workflow creates one pending Agent enrollment generation and one delivery grant. The
grant binds exactly one canonical `target_id`, `agent_id`, and monotonically increasing `generation`; it has an expiry
and a lifecycle of `pending`, `delivered`, `consumed`, or `revoked`.

Runtime Target generates a cryptographically random 256-bit bootstrap token. It persists neither the token nor a raw
SHA-256 digest. Its only verifier is `SHA-256(token || server-side pepper)`, where the pepper is held outside
module-owned persistence and is not logged, returned, or included in an audit event. The plaintext token is a
one-time delivery payload and is never returned by an Operator HTTP endpoint.

### 3. Authenticated automation handoff and delivery receipt

Existing deployment automation is outside the Graft runtime boundary. The MVP reuses the existing authenticated
deployment trust boundary for grant handoff and receipt submission. Runtime Target receives only the already verified
automation identity and authorization context from that boundary; it does not create a deployment trust bundle, client
certificate identity, listener, shared secret, webhook-signature scheme, or separate automation PKI. Authentication
mechanics remain owned by deployment automation and its established control channel, rather than becoming a second
Runtime Target authority.

The authorized Runtime Target workflow records the expected `automation_id` and an opaque Docker installation
reference with the grant. Those are Docker delivery evidence for this implementation, not fields on `AgentIdentity` or
the provider-neutral enrollment aggregate. The authenticated automation requests a handoff for the still-pending,
unexpired grant through that existing boundary. Runtime Target verifies the supplied trusted automation identity and
grant binding before it releases the one-time bootstrap token. It durably records a unique `handoff_id`, authorized
automation identity, and server handoff time before returning the token while the grant remains `pending`. The token is
released once only: a retransmission, a second `handoff_id`, or a request from another automation identity never
returns it. A lost handoff response is resolved by revoking the pending enrollment and creating a fresh grant, not by
redisclosing the original token.

After Docker secret creation and injection, that same authenticated automation submits this minimum, versioned
receipt envelope:

```text
protocol_version            # exactly `graft.delivery-receipt.v1`
receipt_id                 # immutable, globally unique automation-generated identifier
handoff_id                 # the Graft-issued accepted handoff identifier
delivery_grant_id
target_id
agent_id
generation
automation_id
delivered_at               # automation's asserted delivery time
docker_installation_ref    # exactly the opaque reference frozen on the grant
docker_secret_ref          # non-secret opaque secret/version reference
```

The existing deployment trust boundary authenticates the envelope; this protocol does not define a second signature or
transport-identity scheme. The receipt never contains the bootstrap token, a token verifier or token-derived digest,
private key, certificate PEM, Registry credential, Docker secret value, or a Docker API credential. Graft persists only
a redacted receipt record and audit evidence. `docker_secret_ref` and `docker_installation_ref` are opaque evidence
handles, not values Graft uses to inspect, create, read, update, or remove Docker secrets.

Receipt acceptance requires all of the following: the authenticated `automation_id` equals the identity recorded at
handoff and in the envelope; all target, Agent, generation, grant, and installation bindings equal the frozen grant;
the handoff precedes receipt acceptance; and the grant remains pending and unexpired. The server records its own
receipt time as the authorization time; `delivered_at` is retained only as asserted audit evidence and must be
parseable and no later than receipt acceptance. The first valid receipt changes the grant to `delivered`. A retry with
the same `receipt_id` and semantically equivalent canonical envelope returns the existing non-secret receipt result;
a reused `receipt_id`, `handoff_id`, or grant with changed content, a receipt for a consumed, revoked, expired, or
unknown grant, or any identity/binding mismatch fails closed and must not replace the recorded evidence. The original
grant must then be revoked before another delivery attempt.

A receipt is durable delivery evidence and audit input, not the binding authority. It proves only that the authorized
automation asserted a particular Docker secret injection against the frozen installation reference. It does not prove
Docker accepted or retained the secret, that the intended process read it, that the process is the intended Agent, or
that the Agent possesses the matching private key. Runtime Target binds and activates the Agent only after successful
bootstrap validation. A receipt cannot activate a generation, substitute for a bootstrap token, or authorize a
different Target, Agent Identity, generation, Docker service, or certificate request.

The external acceptance precondition for the delivery implementation is a target-deployment conformance demonstration
through the existing deployment trust boundary: (1) an unauthorized automation action cannot receive a token or submit
a receipt; (2) an authorized action can receive exactly one bound handoff, create and inject a Docker secret, and
submit the minimal receipt; (3) a duplicate receipt is idempotent while altered or cross-grant replays fail closed;
(4) the persisted receipt and audit records contain no secret material; and (5) receipt success alone cannot activate
an Agent. This precondition does not require a new automation identity PKI.

### 4. Bootstrap authorization and Vault issuance

The Agent creates its key locally and sends the bootstrap token and CSR only to Graft's dedicated
server-authenticated TLS bootstrap listener. Graft never forwards the token to Vault and never receives or persists the
Agent private key.

Within one database transaction, the listener and Runtime Target service validate the token verifier, pending and
unexpired delivery grant, canonical target/Agent Identity/generation binding, current Docker delivery evidence, and CSR
public-key fingerprint,
then record a stable issuance/idempotency key and the authorized bootstrap attempt. Only after that authorization
succeeds may `credential-vault`, using its AppRole-backed connection, request Vault PKI issuance for the exact ADR-023
SPIFFE URI SAN and the validated CSR public key. Vault remains the PKI issuer; it is not a Runtime Target binding
authority.

Vault issuance is an external side effect and is not part of that database transaction. A timeout or restart after the
request is recoverable only by querying or reconciling the same stable issuance key with the Vault adapter before a
new issuance is attempted. The adapter must not create a second certificate merely because the listener did not observe
the first response.

On success, the bootstrap response returns only the certificate chain and non-secret trust-bundle reference over TLS
with `Cache-Control: no-store`. Graft stores certificate serial, public-key fingerprint, expiry, trust-bundle version,
and redacted issuance evidence, but never private-key material, PEM, or plaintext bootstrap material.

### 5. Completion, retries, and failure behavior

After verified issuance, one database transaction consumes the token and delivery grant, activates the enrolled
generation, persists certificate evidence, and writes a durable redacted audit fact. A repeated completion request
using the same token and the same CSR fingerprint returns the known prior non-secret result or resumes the documented
issuance reconciliation path. A different CSR fingerprint, token replay after consumption, expired grant or token,
revoked enrollment, certificate mismatch, or any incomplete authorization fails closed and does not activate a
generation.

All later Agent traffic uses ADR-023 mutual TLS and exact active-generation validation. The bootstrap listener is not a
bearer-token substitute and does not admit normal Agent ledger or telemetry operations.

### 6. Publication gate

No Operator enrollment endpoint and no Agent endpoint is published in canonical `openapi/**` or generated server/Web
projections until the complete Docker-only conformance slice has passed: delivery-grant handoff and receipt verification,
bootstrap authorization, Vault issuance, durable activation, mTLS reconnect, revocation, and ledger replay tests.
Internal contracts may be stabilized first, but their existence is not permission to publish a partial HTTP surface.

## Consequences

- `credential-vault` has a narrow provider-neutral CSR-based Vault PKI issuance role after Runtime Target
  authorization; it does not deliver Docker secrets or own business enrollment activation.
- Runtime Target needs durable provider-neutral Agent Identity, pending enrollment, delivery-grant, receipt-evidence,
  token-verifier, CSR-fingerprint, and certificate-evidence facts with transactional lifecycle transitions and redacted
  audit outbox coverage. Docker delivery facts remain receipt evidence, not the enrollment model.
- The deployment environment must supply a verifiable automation-to-Graft grant handoff; Graft does not provision,
  inspect, or remove Docker secrets.
- Bootstrap, certificate binding, and replay behavior have explicit service and HTTP test seams before any public Agent
  contract becomes available.

## Rejected Alternatives

- Letting Vault delivery alone establish Runtime Target membership: PKI issuance proves material issuance, not the
  intended business binding.
- Returning a bootstrap token through bearer Operator APIs: it expands secret custody and creates an avoidable
  transport, logging, and persistence exposure.
- Storing `SHA-256(token)` without a server-side pepper: a database disclosure would permit offline candidate-token
  verification.
- Treating an automation delivery receipt as activation: delivery evidence cannot prove possession of the matching
  private key or a valid CSR.
- Publishing partial enrollment or Agent operations before the end-to-end protocol works: callers would receive a
  contract that cannot safely establish identity, revocation, or durable replay protection.
