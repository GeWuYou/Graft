# Runtime Targets Infrastructure IA Trace

## 2026-07-12 Work Intake And Authority Foundation

- Work Intake classified this as a long-running `feature`: design, roadmap, and active topic are required; ADR is not required because the approved IA is documented directly in the repository design authority.
- Added Runtime Target and Infrastructure design authority. It keeps the existing sidebar mode and limits Section Labels to non-interactive visual grouping.
- Repaired the navigation design conflict: Docker, Kubernetes, and Podman may be ordinary Infrastructure Provider menus only when matching registered Targets report real capabilities.
- Reframed Container list semantics: user-facing “source” is replaced by independent `deployment_type` and `runtime_target`; standalone containers remain outside Application.
- Reaffirmed Application/Project ownership of Compose lifecycle and the shared nature of Registry/Certificates.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation"],
  "pending_batches": ["menu-section-contract-and-sidebar-rendering", "runtime-target-foundation", "container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "work-intake-and-authority-foundation",
  "next_batch": "menu-section-contract-and-sidebar-rendering",
  "closeout_status": "batch-1-complete"
}
```

## 2026-07-12 Runtime Target Foundation

- Added the OpenAPI-owned runtime-target list, detail, and explicit refresh contract. The server maps persisted target facts into generated types; the web list/detail drawer consumes only those generated contract types.
- Registered the `runtime-target` module with Local Docker Unix-socket discovery/upsert that is bounded and non-fatal during boot. The first release exposes no remote connection creation, plaintext TCP, or credential material.
- Added refresh authorization and domain audit events with resource type `runtime_target`; the audit payload has target identity, provider, and result only, never endpoint or credentials.
- Added the target table's Chinese comments, `deleted_at = 0` live-row semantics, partial provider/endpoint uniqueness, and regenerated migration and module registry embeds. No-cache remains intentional because rendering reads persisted probe facts.
- Remote Docker mTLS is deferred until a credential-vault authority owns certificate and private-key lifecycle; this batch must not introduce a temporary credentials table or plain TCP connection path.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering", "runtime-target-foundation"],
  "pending_batches": ["container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "runtime-target-foundation",
  "next_batch": "container-deployment-type-and-target-filter",
  "closeout_status": "batch-3-complete"
}
```

## 2026-07-12 Runtime Target Table Design Summary

- Owner: new `server/modules/runtime-target` module; it owns target connection identity, provider capability discovery facts, availability and probe timestamp, but never container state or Compose lifecycle.
- Table: `runtime_targets` uses `id`, normalized `provider` and `endpoint`, display and masked endpoint labels, `connection_kind`, capability JSON, availability/error/check facts, `system_managed`, and standard `created_at/created_by/updated_at/updated_by/deleted_at/deleted_by` audit fields.
- Live semantics: all reads require `deleted_at = 0`; a partial unique index on `(provider, endpoint) WHERE deleted_at = 0` prevents duplicate live targets. List and direct lookup indexes cover `(deleted_at, provider, availability, checked_at DESC)` and live provider/endpoint lookup.
- Cache decision: no cache. Local Docker discovery is bounded and runs only on module boot or an explicit refresh request; list rendering consumes persisted probe facts and never polls Docker.
- Security: first release has no target creation/update API and stores no remote endpoint credentials, TLS material, or secrets. Remote mTLS requires a future credential-vault authority.

## 2026-07-12 Menu Section Contract And Sidebar Rendering

- Added `SectionKey` to backend menu entry metadata and rejected it on non-navigable domain groups; it does not create a menu node, route, permission, breadcrumb, tab, quick action, or search result.
- Kept the existing Bootstrap OpenAPI wire shape unchanged in this bounded batch. The web module registration attaches matching visual-only section metadata to the already authorized `/containers` route rather than emitting an undeclared wire field.
- Reclassified the visible Infrastructure entry as `Docker` while preserving the canonical `/containers` route and container API. The page semantic title remains `Containers`.
- The existing sidebar now renders a `Runtime` / `运行时` text label once before visible runtime entries. Header navigation does not render section labels, and collapsed sidebars hide them.
- Validation passed: focused Go and Vitest suites, `cd server && go run ./cmd/graft validate backend`, `cd web && bun run check`, and `git diff --check`.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering"],
  "pending_batches": ["runtime-target-foundation", "container-deployment-type-and-target-filter", "docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "menu-section-contract-and-sidebar-rendering",
  "next_batch": "runtime-target-foundation",
  "closeout_status": "batch-2-complete"
}
```

## 2026-07-12 Container Deployment Type And Runtime Target Filter

- Replaced the Container list's user-facing "source" filter set with independent OpenAPI-owned `deployment_type` and `runtime_target_id` query fields. The public deployment taxonomy is deliberately limited to `standalone`, `compose`, and `unknown`; Swarm and Kubernetes remain future provider-specific semantics rather than ambiguous filters.
- Container service resolves Docker target identity through the narrow `moduleapi.RuntimeTargetReader`, so Container does not own target connections, endpoints, credentials, or a duplicate target list. The reader validates provider and integer boundaries before exposing a target summary.
- The Docker container page loads Docker Runtime Targets for the target selector, does not render a redundant Provider filter, and presents Compose project/service context separately from Standalone and Unknown deployment badges.
- Cache decision: deferred with reason. Container reads remain runtime-backed while target metadata is persisted; no new cache is justified until measured list-read latency or target fan-out identifies a hotspot. The future trigger is an observed performance regression under authorized multi-target reads.
- Validation passed: `go run ./cmd/graft validate openapi`, focused Go container/runtime-target tests, focused container Vitest suites, `go run ./cmd/graft validate backend`, frontend format/type/i18n/lint/style/hygiene/build gates, and `git diff --check`. The full `bun run check` test wave exposed two unrelated timing failures in permission and Monaco tests; both passed on direct retry.

## Loop Batch State

```json
{
  "loop_mode": "topic-completion-loop",
  "completed_batches": ["work-intake-and-authority-foundation", "menu-section-contract-and-sidebar-rendering", "runtime-target-foundation", "container-deployment-type-and-target-filter"],
  "pending_batches": ["docker-resources-and-application-integration", "cross-boundary-acceptance-and-archive-readiness"],
  "current_batch": "container-deployment-type-and-target-filter",
  "next_batch": "docker-resources-and-application-integration",
  "closeout_status": "batch-4-complete"
}
```
