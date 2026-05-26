RBAC generated server bindings are produced through `go generate`.

This package is intentionally limited to a guarded RBAC management migration batch:

- `getPermissions`
- `getRoles`
- `getRolePermissions`
- `postRolePermissionAssign`
- `getUserRoles`
- `postUserRolesAssign`

The generated layer constrains header semantics, request-body shape, and the handler-facing method signatures without
broadening route ownership, middleware ownership, or `httpx` envelope ownership.
