# Support Boundary

本文件固定 `v0.1.0` release governance 的 support boundary authority。它用于防止 operator 或开发者把未来能力误读成当前承诺。

## Supported In `v0.1.0`

- one active repository release line at a time
- one official `server` artifact and one official `web` artifact from the same release tag
- explicit operator-run migration through `graft migrate up` or `graft dev`
- documentation-first release safety, identity, versioning, config, and upgrade governance
- release-binary authority where the `server` artifact itself carries canonical `BuildInfo`, the default embedded
  migration chain, and the runtime embedded OpenAPI asset
- release-package authority where `LICENSE`, `SBOM`, license compliance report, checksum bundle, and `web` dist may be
  published as external release assets instead of binary-embedded payload

## Not Yet Promised In `v0.1.0`

- `Docker` / `Compose` / `Kubernetes` / hosted deployment support matrix
- automatic rollback tooling
- implicit startup migration or startup-time schema repair
- independent `server` / `web` official release trains
- richer operator-facing introspection UI beyond future minimal `graft version`
- a dedicated operator-facing version API
- authoritative startup-log BuildInfo surface
- a claim that publish workflow YAML is the single source of truth for release support semantics
- a claim that `graft validate release` already proves every external release attachment exists

## Experimental Definition

- a capability is only `experimental` when release notes or dedicated authority docs explicitly label it that way
- internal code paths, draft scripts, unpublished artifacts, or local workflows do not create an implied support
  promise

## Authority Boundary

- `ai-plan/design/release/**` is the authority for release support semantics
- workflow files, GitHub Release forms, or local packaging scripts are derived automation and must not silently
  redefine the support boundary
- when automation and release docs diverge, repair the release docs or align the automation in the same task instead of
  treating workflow behavior as self-authorizing truth

## `v0.1.0` Before And After Boundary

- before `v0.1.0`, the repository does not promise a stable public release-binary contract beyond best-effort internal
  iteration
- starting at `v0.1.0`, official repository releases must keep one coherent same-tag release package and the release
  binary contract defined in `build-identity-contract.md`
- after `v0.1.0`, any expansion of release validation, attachment requirements, support matrix, or operator promises
  must update the release authority docs first; publish automation may implement that policy but does not create it on
  its own

## Current Documentation Status

- operator-facing install, configuration, upgrade, and versioning doc set is deferred in the current phase
- until that doc set is intentionally created, this file and the other `ai-plan/design/release/**` authorities define
  the supported and unsupported release-governance boundary
