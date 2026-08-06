# Build Center Roadmap

## Phase 0: Authority And Contracts

- Establish Build module descriptor, permissions, menu/route contract, and narrow Project/Container/Task capabilities.
- Keep generated registries and OpenAPI projections derived from canonical inputs.

## Phase 1: Dockerfile Build

- Add `build_jobs`, `build_job_args`, and `build_artifacts` migrations with Chinese table/column comments.
- Add transactional submission, task stage executor, Docker CLI capability, API, and web list/create/detail workflow.
- Validate path boundaries, authorization, idempotency, cancellation, unknown outcomes, artifact settlement, and log redaction.

## Phase 2: History

- Add filters, saved views, artifact runtime-drift indicators, retention, and bounded durable projections only if query needs prove it.

## Phase 3+: Sources And Pipelines

- Add Git snapshots, buildx/multi-platform/cache/secret references, OCI relations, SBOM/signatures, and pipeline triggers.

## Non-goals

- Do not alter Application deployment lifecycle or add a second task system.
- Do not expose Docker endpoint/credentials or arbitrary host paths in Build API.
