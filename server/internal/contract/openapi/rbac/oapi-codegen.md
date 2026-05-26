RBAC generated server bindings are produced through `go generate`.

This package is intentionally limited to a guarded RBAC management migration batch:

- `getPermission`
- `getPermissions`
- `getRole`
- `getRoles`
- `getRolePermissions`
- `postRoleDelete`
- `postRolePermissionsAdd`
- `postRolePermissionsRemove`
- `postRolePermissionsReplace`
- `postRoleStatus`
- `postRoleUpdate`
- `postRoles`
- `getUserRoles`
- `postUserRolesAdd`
- `postUserRolesRemove`
- `postUserRolesReplace`
- `postUsersRolesAdd`
- `postUsersRolesRemove`
- `postUsersRolesReplace`

The generated layer constrains header semantics, request-body shape, and the handler-facing method signatures without
broadening route ownership, middleware ownership, or `httpx` envelope ownership.
