package rbacopenapi

// ReadServerInterface is the minimal generated handler contract for the guarded RBAC read batch.
type ReadServerInterface interface {
	GetPermissions(params GetPermissionsParams)
	GetRoles(params GetRolesParams)
	GetRolePermissions(id uint64, params GetRolePermissionsParams)
}

// UserRoleServerInterface is the minimal generated handler contract for guarded user-role read migration.
type UserRoleServerInterface interface {
	GetUserRoles(id uint64, params GetUserRolesParams)
}

// WriteServerInterface is the minimal generated handler contract for guarded RBAC write migration.
type WriteServerInterface interface {
	PostRoles(params PostRolesParams, body PostRolesJSONRequestBody)
	PostRoleUpdate(id uint64, params PostRoleUpdateParams, body PostRoleUpdateJSONRequestBody)
	PostRolePermissionAssign(id uint64, params PostRolePermissionAssignParams, body PostRolePermissionAssignJSONRequestBody)
	PostUserRolesAssign(id uint64, params PostUserRolesAssignParams, body PostUserRolesAssignJSONRequestBody)
}
