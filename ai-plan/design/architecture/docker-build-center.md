# Docker Build Center (Historical)

## Status

This document records the Docker-first Build Center authority that delivered the original Build Job history and local
Docker execution slice. It is historical evidence only and must not govern new Build write contracts, UI flows, Runtime
Target capability semantics, or future Build implementations.

The canonical successor is [Build Domain v2](build-domain-v2.md). Its delivery order is
[Build Domain v2 Roadmap](../../roadmap/build-domain-v2.md).

## Preserved Evidence

- Build remains a first-level `Build` domain; Docker image inventory remains a Docker runtime resource rather than the
  Build authority.
- Task Runtime remains the only authority for execution status, stages, logs, cancellation, retry, recovery, and
  `task:{id}` realtime updates.
- Application Workspace and Runtime Target connections remain upstream authorities; Build must not accept arbitrary host
  paths, endpoint details, or credential values.
- Legacy completed Build Jobs stay readable as immutable historical evidence.

## Superseded Assumptions

The former direct Docker build input model (`Application ID`, context path, Dockerfile path, image repository, and tag)
is superseded. New Build Domain v2 submissions freeze a Workspace Snapshot and Execution Plan before Task submission,
select a compatible Runtime Target build capability through a Builder Instance or Pool, and produce first-class Artifact
and Publication evidence.
