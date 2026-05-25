package rbacopenapi

// ReadServerInterface is the minimal generated handler contract for the guarded RBAC read batch.
type ReadServerInterface interface {
	GetPermissions(params GetPermissionsParams)
	GetRoles(params GetRolesParams)
	GetRolePermissions(id uint64, params GetRolePermissionsParams)
}

// UserRoleServerInterface is the minimal generated handler contract for guarded user-role GET/assign migration.
type UserRoleServerInterface interface {
	GetUserRoles(id uint64, params GetUserRolesParams)
	PostUserRolesAssign(id uint64, params PostUserRolesAssignParams, body PostUserRolesAssignJSONRequestBody)
}
