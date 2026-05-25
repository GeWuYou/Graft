package rbacopenapi

// ReadServerInterface is the minimal generated handler contract for the guarded RBAC read batch.
type ReadServerInterface interface {
	GetPermissions(params GetPermissionsParams)
	GetRoles(params GetRolesParams)
	GetRolePermissions(id uint64, params GetRolePermissionsParams)
}
