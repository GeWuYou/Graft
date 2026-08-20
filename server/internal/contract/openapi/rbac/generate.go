package rbacopenapi

//go:generate go tool oapi-codegen --include-operation-ids deleteRolePermissions,deleteUserRoles,deleteUsersRoles,getPermission,getPermissions,getRole,getRoles,getRolePermissions,postRoleClone,postRoleDelete,postRolePermissionsAdd,postRolePermissionsReplace,postRoles,postRoleStatus,postRoleUpdate,getUserRoles,postUserRolesAdd,postUserRolesReplace,postUsersRolesAdd,postUsersRolesReplace --generate types --package rbacopenapi -o zz_generated.management.go ../../../../../openapi/openapi.yaml
