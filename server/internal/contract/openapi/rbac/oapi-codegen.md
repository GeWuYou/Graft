RBAC generated server bindings are produced through `go generate`.

This package is intentionally limited to `getPermissions` so the guarded progressive migration can validate generated
server constraints without broadening route ownership, middleware ownership, or `httpx` envelope ownership.
