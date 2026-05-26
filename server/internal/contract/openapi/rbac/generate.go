package rbacopenapi

//go:generate go tool oapi-codegen --include-operation-ids getPermission,getPermissions,getRole,getRoles,getRolePermissions,postRoleDelete,postRolePermissionsAdd,postRolePermissionsRemove,postRolePermissionsReplace,postRoles,postRoleStatus,postRoleUpdate,getUserRoles,postUserRolesAdd,postUserRolesRemove,postUserRolesReplace,postUsersRolesAdd,postUsersRolesRemove,postUsersRolesReplace --generate types --package rbacopenapi -o zz_generated.management.go ../../../../../openapi/openapi.yaml
