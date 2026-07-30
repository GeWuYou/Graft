# ADR-008: Deployment Runtime Context

- Status: accepted
- Date: 2026-07-30
- Scope: deployment identity, Docker fact interpretation, and shared controlled-operation prerequisites
- Supersedes: the Compose-root discovery/configuration ownership portion of ADR-006

## Context

Compose root, project metadata, and runtime identity are deployment facts, not Update module configuration. Keeping an environment-variable branch and Docker-label interpretation inside Update duplicates a capability that Backup, Restore, and future Compose management will also need. It also lets separate consumers derive different host paths for one operation.

Docker integration is still required, but Docker is a runtime provider rather than a deployment-policy owner. Container code can inspect containers, labels, mounts, networks, images, and Compose metadata; it must not decide what those facts mean for a Graft deployment. The deployment layer must remain able to add other providers without teaching Container about binary, systemd, Podman, or Kubernetes semantics.

## Decision

Introduce a compile-time `deployment` module that owns a `DeploymentRuntime` capability. It is the only component allowed to interpret declared deployment configuration, raw Docker inspect facts, and host-path evidence into a `DeploymentContext`.

```go
type DeploymentRuntime interface {
    Current(context.Context) DeploymentContext
    Freeze(context.Context, DeploymentFreezeRequest) (DeploymentSnapshot, error)
}
```

`DeploymentContext` is an immutable value object containing the declared/detected runtime state, source, opaque Compose candidates, and structured diagnostics. `DeploymentSnapshot` is its immutable operation-scoped form with the selected candidate and fingerprint. `Current` returns the current read-only context, including structured diagnostics when it cannot be constructed. `Freeze` re-discovers and validates deployment facts, then returns the snapshot that a controlled operation must consume. Consumers do not mutate a context or independently re-resolve one; a changed fact requires another runtime call. The exact Go package and contract placement follow the module/DI architecture, but the capability remains the single public interpretation entry.

The `container` module exposes raw Docker inspect facts only. It may obtain Compose labels, `config_files`, `working_dir`, mount information, image identity, and socket errors, but it does not expose Deployment Contexts or decide whether an inspect result is an official Compose deployment.

`platform-update`, Backup, Restore, diagnostics, and future Compose management consume `DeploymentContext`; none may inspect Docker labels, read deployment environment variables, or derive a Compose root independently. An operation must reject stale, missing, ambiguous, or fingerprint-mismatched snapshots instead of silently resolving a new one during execution.

Deployment configuration uses these canonical keys with no legacy aliases, dual reads, or fallback keys:

- `GRAFT_DEPLOYMENT_RUNTIME` declares the requested runtime, currently `compose` or `binary`.
- `GRAFT_DEPLOYMENT_COMPOSE_ROOT` is optional for Compose and, when set, is a non-empty Docker-daemon-host absolute path.
- `GRAFT_DEPLOYMENT_SERVICE_MANAGER` and `GRAFT_DEPLOYMENT_SERVICE_NAME` describe a supported binary service-control surface when required.
- `GRAFT_IMAGE_TAG` remains the sole official Compose image tag and update-strategy declaration.

No `GRAFT_DEPLOYMENT_BINARY_PATH`, `GRAFT_DEPLOYMENT_WEB_ROOT`, `GRAFT_UPDATE_*` deployment aliases, or path-based `ExecStart` inference are introduced. A binary runtime that lacks supported service-control facts remains diagnostic/manual-only rather than guessing a filesystem layout.

## Consequences

Deployment state has one explicit owner and one immutable operation snapshot, so a status page, preflight, runner, Backup, and Restore can use the same resolved facts. Compose runner trust boundaries from ADR-006 remain: only the frozen daemon-host path is mounted and executed by the runner.

The new module is an infrastructure capability, not a new HTTP surface, persistent store, or startup script. It does not cache deployment facts permanently at server boot, because Docker labels, project configuration, and daemon reachability can change. Operator guidance can expose structured reasons such as socket unavailable, labels absent, ambiguous candidates, invalid explicit root, or fingerprint mismatch without accepting host paths from Web requests.

## Rejected Alternatives

- Update-owned Compose root resolution: duplicates deployment interpretation and couples unrelated controlled operations to Update.
- Dockerfile or container entrypoint injection: a container start script cannot natively know the host Compose root and would duplicate Docker API discovery with extra CLI/tooling dependencies.
- Installation-script-only discovery: useful as an installer convenience but insufficient for existing/manual deployments and runtime drift; it cannot replace execution-time validation.
