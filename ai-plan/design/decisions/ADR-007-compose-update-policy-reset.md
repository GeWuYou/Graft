# ADR-007: Compose Image Tag As Update Strategy

- Status: accepted
- Date: 2026-07-29
- Scope: official Compose deployment configuration and platform-update strategy selection
- Supersedes: the image-reference and deployment-policy portions of ADR-006

## Context

The Beta self-update flow needs one deployment-owned declaration that an administrator can understand without a second policy value that can contradict it. The prior repository-plus-digest `.env` shape made the deployed image identity hard to replace as one value and did not express operator intent. The subsequent two-key `GRAFT_IMAGE_TAG` plus `GRAFT_UPDATE_POLICY` shape could express contradictory intent. Its receipt-writing runner boundary remains valid, but neither image-reference policy is retained.

## Decision

Official Compose deployments use `GRAFT_IMAGE_TAG` as their only image and update-strategy contract for the fixed official server and web repositories:

- `latest` tracks the stable channel; `beta` tracks the Beta channel.
- A SemVer tag `vX.Y.Z` is a fixed stable release; `vX.Y.Z-beta.N` is a fixed Beta release.
- The server reads only the injected container `GRAFT_IMAGE_TAG`; it never reads a host `.env`. Web does not maintain a second writable strategy value.
- The resolved target is always a verified GitHub Release manifest and its immutable server/web digests. It is runtime state, not deployment configuration.
- For a tracking tag, the runner applies the manifest-derived explicit target only as a runner-scoped Compose override. It must retain `latest` or `beta` in deployment `.env` after success. For a fixed-tag upgrade, it atomically replaces the declared tag with the administrator-selected, strictly newer fixed release tag.
- Before pull, migration, or recreation, the runner verifies that target references are the official server and web repositories, share the manifest release tag, and resolve to the selected manifest digests.
- Fixed releases may select only a strictly newer verified release in the same channel. The server enforces version ordering, channel matching, official-release membership, and digest matching; the UI is not an authorization boundary.

Changing between `latest` and `beta`, or between tracking and fixed modes, is a separate product operation and is not part of this slice. `nightly` is not implemented. No `GRAFT_UPDATE_POLICY` compatibility, alternate image key, fallback, alias, dual-read, or migration path exists.

## Consequences

The visible `.env` state is simple and auditable, while immutable runtime identity remains enforced by manifest digest validation. A following deployment retains the declared tracking intent instead of silently becoming pinned to the last resolved release. Deployment strategy stays in Compose configuration rather than System Config or Web state. ADR-006 still governs the one-shot runner, Compose-root trust boundary, backup/migration order, and marker-bounded container-log receipt protocol.
