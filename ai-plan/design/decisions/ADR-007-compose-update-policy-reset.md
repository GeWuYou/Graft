# ADR-007: Compose Update Policy Reset

- Status: accepted
- Date: 2026-07-29
- Scope: official Compose deployment configuration and platform-update policy selection
- Supersedes: the image-reference and deployment-policy portions of ADR-006

## Context

The Beta self-update flow needs a deployment-owned policy that an administrator can understand and the Compose runner can atomically update. The prior repository-plus-digest `.env` shape made the deployed image identity hard to replace as one value and did not express operator intent. Its receipt-writing runner boundary remains valid, but its image-reference policy does not.

## Decision

Official Compose deployments use these `.env` keys as their only image and update-policy contract:

- `GRAFT_IMAGE_TAG` is the single shared tag for the fixed official server and web repositories. Initial deployment may select `latest`, `beta`, or a fixed release tag.
- `GRAFT_UPDATE_POLICY` is one of `stable`, `beta`, `fixed`, or `manual`.
- `stable` and `beta` resolve only a verified GitHub Release manifest in the respective channel. `fixed` resolves an administrator-selected verified release. `manual` never changes image references or executes automated upgrade work.
- The runner accepts manifest-derived complete target references, verifies that they are the official server and web repositories with one explicit shared tag, then atomically writes `GRAFT_IMAGE_TAG`, pulls, and verifies the resulting server and web digests against the selected manifest before migration or recreation.

`nightly` is not implemented. `latest` and `beta` are mutable manual-deployment choices, never manifest-verified runner targets. No alternate image key has a fallback, alias, dual-read, or migration path.

## Consequences

The visible `.env` state is simple and auditable, while immutable runtime identity remains enforced by manifest digest validation. Deployment policy stays in Compose configuration rather than System Config or Web state. ADR-006 still governs the one-shot runner, Compose-root trust boundary, backup/migration order, and marker-bounded container-log receipt protocol.
