Monitor-only generated server bindings are produced through `go generate`.

This package is intentionally limited to `getMonitorServerStatus` so the spike can validate generated server constraints
without broadening the repository-wide runtime pattern.
