# Build Identity Contract

本文件固定 `v0.1.0` release governance 的 build identity 开发约束。它定义 authority contract，不声明这些入口已经全部实现。

## Scope

- canonical release identity
- `BuildInfo` minimum fields
- `graft version` minimum boundary
- build identity visibility contract across `CLI` / `API` / `logs`

## Canonical Release Identity

- official product release identity is the repository Git tag `vMAJOR.MINOR.PATCH`
- official `server` artifact, `web` artifact, and release notes must derive from the same release tag
- `BuildInfo.version` uses bare semver such as `0.1.0`
- the canonical release tag keeps the `v` prefix, for example `v0.1.0`

## Minimum BuildInfo Fields

Release-grade `BuildInfo` must include:

- `version`
- `git_commit`
- `build_time_utc`
- `git_tree_state`

Optional future metadata may be added later, but must not replace or weaken the baseline fields above.

## Release Binary Required Payload

The official `server` release binary contract must carry these runtime-readable assets inside the binary itself:

- canonical `BuildInfo`
- the default embedded migration chain used by the repository-default migration entry
- the runtime embedded bundled OpenAPI asset used as the canonical runtime HTTP contract snapshot

These assets are part of the release-binary authority because they are required for runtime-safe operator inspection or
runtime-safe explicit validation without depending on GitHub Release attachments, CI logs, or external network fetches.

The following release assets may be published alongside a release tag without being embedded in the `server` binary:

- `LICENSE`
- `SBOM`
- license compliance report
- `web` dist artifact
- checksum manifests and other release-note attachments

External release assets may be required by the wider release package, but they do not redefine the binary contract and
must not be described as if the `server` binary reads them at runtime.

## `graft version` Minimum Boundary

- current repository state exposes a `graft version` subcommand
- `graft version` is a pure metadata readout
- it must not require PostgreSQL, Redis, HTTP startup, or migration execution
- release builds must expose at least the four minimum BuildInfo fields
- non-release or local builds may identify themselves as `dev`, but must not be presented as official tagged releases

## Visibility Contract

### CLI

- `graft version` is the canonical operator-facing metadata surface
- release-grade CLI output must present the minimum BuildInfo baseline without requiring external services

### API

- `v0.1.0` does not yet promise a dedicated operator-facing version API
- no document may imply that an API identity surface already exists unless implementation authority lands first

### Logs

- `v0.1.0` does not yet promise startup-log BuildInfo emission as an authoritative support surface
- logs may later mirror build identity, but that future behavior must not replace the CLI boundary

## Current Local-Build Fallback

When BuildInfo ldflags are not injected, the authoritative operator-facing fallback remains:

- release tag
- published artifact names
- release notes

For local binaries and `go run` development flows, `graft version` must report the explicit fallback baseline:

- `version=dev`
- `git_commit=unknown`
- `build_time_utc=unknown`
- `git_tree_state=unknown`

## `graft validate release` Minimum Guarantee

`graft validate release` is the canonical repository entrypoint for release-binary contract checks. Its minimum
guarantee is intentionally narrow:

- verify release-grade `BuildInfo` is present
- verify the binary identifies itself as a clean release build
- verify the runtime embedded bundled OpenAPI asset matches the canonical repository bundle
- verify the default embedded migration chain can be synthesized as the release-default migration authority

`graft validate release` must not be described as proving the full publish bundle, GitHub Release attachment set,
operator deployment environment, or release-note completeness unless those checks are explicitly added to this
authority later.
