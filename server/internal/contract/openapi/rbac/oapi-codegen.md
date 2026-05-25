RBAC generated server bindings are produced through `go generate`.

This package is intentionally limited to the guarded RBAC read batch:

- `getPermissions`
- `getRoles`
- `getRolePermissions`

The generated layer constrains header semantics and the handler-facing read shape without broadening route ownership,
middleware ownership, or `httpx` envelope ownership.
