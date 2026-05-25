package rbacopenapi

//go:generate go tool oapi-codegen --include-operation-ids getPermissions,getRoles,getRolePermissions,postRolePermissionAssign,getUserRoles,postUserRolesAssign --generate types --package rbacopenapi -o zz_generated.management.go ../../../../../openapi/openapi.yaml
