# Build Identity Contract

本文件固定 `v0.1.0` release governance 的 build identity 开发约束。它定义 authority contract，不声明这些入口已经全部实现。

## Scope

- canonical release identity
- future `BuildInfo` minimum fields
- future `graft version` minimum boundary
- build identity visibility contract across `CLI` / `API` / `logs`

## Canonical Release Identity

- official product release identity is the repository Git tag `vMAJOR.MINOR.PATCH`
- official `server` artifact, `web` artifact, and release notes must derive from the same release tag
- `BuildInfo.version` uses bare semver such as `0.1.0`
- the canonical release tag keeps the `v` prefix, for example `v0.1.0`

## Minimum BuildInfo Fields

Future release-grade `BuildInfo` must include:

- `version`
- `git_commit`
- `build_time_utc`
- `git_tree_state`

Optional future metadata may be added later, but must not replace or weaken the baseline fields above.

## `graft version` Minimum Boundary

- current repository state does not yet expose a `graft version` subcommand
- when implemented, `graft version` must be a pure metadata readout
- it must not require PostgreSQL, Redis, HTTP startup, or migration execution
- release builds must expose at least the four minimum BuildInfo fields
- non-release or local builds may identify themselves as `dev`, but must not be presented as official tagged releases

## Visibility Contract

### CLI

- future `graft version` is the canonical operator-facing metadata surface once implemented
- release-grade CLI output must present the minimum BuildInfo baseline without requiring external services

### API

- `v0.1.0` does not yet promise a dedicated operator-facing version API
- no document may imply that an API identity surface already exists unless implementation authority lands first

### Logs

- `v0.1.0` does not yet promise startup-log BuildInfo emission as an authoritative support surface
- logs may later mirror build identity, but that future behavior must not replace the CLI boundary

## Current Pre-Implementation Fallback

Until BuildInfo injection and `graft version` are implemented, the authoritative operator-facing release identity remains:

- release tag
- published artifact names
- release notes
