# Docker Build Center Architecture

## Authority

The Build domain owns explicit build jobs and immutable artifact evidence. The Task Runtime remains the sole authority
for execution state, stages, logs, cancellation, retry, crash recovery, and `task:{id}` realtime updates. The Container
module owns Docker SDK/CLI execution and daemon error mapping. Project owns authorized Application workspace context;
Build consumes a narrow resolver capability and never receives Project entities or arbitrary host paths.

## Navigation

Build is a first-level domain under the existing `domain.build` group. The canonical page is `Build Jobs` at
`/build/jobs`; Docker image inventory remains a Container runtime resource and is not the build authority.

## First Slice

The first executor is a local Docker CLI adapter using controlled argument arrays, `--progress=plain`, `--iidfile`, and
context cancellation. Inputs are an authorized Application ID, workspace-relative context/Dockerfile paths, an image
repository/tag, and non-sensitive build arguments. Runtime target identity is derived server-side. Registry push,
buildx, multi-platform output, secrets, SBOM, signatures, Git sources, and automatic deployment are later phases.

## Stable Relationships

```text
Application workspace -> BuildJob -> Task Runtime -> Container Docker build -> BuildArtifact
```

BuildJob stores frozen configuration and source/display snapshots but not an independent execution status or log store.
BuildArtifact stores image ID, digest, size, platform, and generated reference evidence; repository/tag remains a mutable
runtime reference. Repeated settlement is idempotent and does not guess success after an unknown executor outcome.

## Security And Compatibility

The API rejects absolute paths, traversal, control characters, symlink escapes, arbitrary Docker endpoint/CLI input,
secret values, and caller-selected runtime targets. Build permissions are `build.read`, `build.create`, `build.cancel`,
and `build.retry`. No compatibility layer is introduced while the canonical module/menu/contract authority can be repaired
directly.

## Delivery

Phase 0 establishes module contracts and task integration seams. Phase 1 adds the three immutable-history tables, the
transactional submission path, Docker executor, API, and web list/create/detail flow. Later phases add history projections,
Git and advanced Docker executors, then pipeline triggers and supply-chain artifacts.
