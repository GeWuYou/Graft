Monitor-only generated server bindings for server status and request performance are produced through `go generate`.

This package remains limited to monitor-owned read operations so generated server constraints stay bounded without
broadening the repository-wide runtime pattern.
