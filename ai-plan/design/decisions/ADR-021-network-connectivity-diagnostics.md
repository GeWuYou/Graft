# ADR-021: Network Connectivity Diagnostics

- Status: accepted
- Date: 2026-08-04

## Context

The existing Network page only runs a fixed, HTTP-only diagnostic for one target. The platform needs both a compact
outbound health view and detailed target troubleshooting without adopting a proxy-client product model. Future SMTP,
LDAP, OCI, OIDC, webhook, Kubernetes, and runtime checks cannot be represented safely by fixed HTTP result columns.

## Decision

1. Keep `Platform > Network` as the single entry. `/:targetId` identifies Target Diagnostics; a report/check ID is
   data selected within that page and never its primary route identity.
2. Make `server/internal/moduleapi` plus `server/modules/network` the canonical Connectivity Target Registry and
   Probe/Report boundary. The target descriptor includes stable target identity, owning module identity, category,
   ordered probe kinds, and typed features.
3. Store diagnostics as a versioned, sanitized `ConnectivityReport` with `ProbeResult[]`, rather than fixed
   DNS/TCP/TLS/HTTP fields. Each new protocol contributes adapters and declared capabilities, not report schema
   columns or UI forks.
4. Batch Connectivity and Target Diagnostics share one execution/report model. Batch is only the aggregate/list
   projection; it must not grow an independent probe pipeline.
5. Preserve the existing narrow HTTP diagnostic contract only as a temporary migration adapter. Phase 2 replaces its
   persistence/API boundary; Phase 3 replaces its web consumer. It is not a second long-term authority.
6. Route explanations show matched strategy, final decision, and reason without exposing a rules graph. Exit IP is
   masked by default, optional by capability, permission-gated when revealed, audited, and never persisted unmasked.

## Consequences

This creates a stable cross-module extension point and allows each future network consumer to self-describe its
diagnostic capability. It adds a controlled migration period while current HTTP API/storage remain in place, so Phase
2 must remove dual-write or dual-authority behavior rather than preserving it as compatibility indefinitely.
