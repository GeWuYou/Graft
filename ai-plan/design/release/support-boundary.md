# Support Boundary

本文件固定 `v0.1.0` release governance 的 support boundary authority。它用于防止 operator 或开发者把未来能力误读成当前承诺。

## Supported In `v0.1.0`

- one active repository release line at a time
- one official `server` artifact and one official `web` artifact from the same release tag
- explicit operator-run migration through `graft migrate up` or `graft dev`
- documentation-first release safety, identity, versioning, config, and upgrade governance

## Not Yet Promised In `v0.1.0`

- `Docker` / `Compose` / `Kubernetes` / hosted deployment support matrix
- automatic rollback tooling
- implicit startup migration or startup-time schema repair
- independent `server` / `web` official release trains
- richer operator-facing introspection UI beyond future minimal `graft version`
- a dedicated operator-facing version API
- authoritative startup-log BuildInfo surface

## Experimental Definition

- a capability is only `experimental` when release notes or dedicated authority docs explicitly label it that way
- internal code paths, draft scripts, unpublished artifacts, or local workflows do not create an implied support
  promise

## Current Documentation Status

- operator-facing install, configuration, upgrade, and versioning doc set is deferred in the current phase
- until that doc set is intentionally created, this file and the other `ai-plan/design/release/**` authorities define
  the supported and unsupported release-governance boundary
