package rbacopenapi

// ReadServerInterface is the minimal generated handler contract for the guarded RBAC read batch.
type ReadServerInterface interface {
	GetPermission(id uint64, params GetPermissionParams)
	GetPermissions(params GetPermissionsParams)
	GetRole(id uint64, params GetRoleParams)
	GetRoles(params GetRolesParams)
	GetRolePermissions(id uint64, params GetRolePermissionsParams)
}

// UserRoleServerInterface is the minimal generated handler contract for guarded user-role read migration.
type UserRoleServerInterface interface {
	GetUserRoles(id uint64, params GetUserRolesParams)
}

// WriteServerInterface is the minimal generated handler contract for guarded RBAC write migration.
type WriteServerInterface interface {
	PostRoleDelete(id uint64, params PostRoleDeleteParams)
	PostRolePermissionsAdd(id uint64, params PostRolePermissionsAddParams, body PostRolePermissionsAddJSONRequestBody)
	PostRolePermissionsRemove(id uint64, params PostRolePermissionsRemoveParams, body PostRolePermissionsRemoveJSONRequestBody)
	PostRolePermissionsReplace(id uint64, params PostRolePermissionsReplaceParams, body PostRolePermissionsReplaceJSONRequestBody)
	PostRoles(params PostRolesParams, body PostRolesJSONRequestBody)
	PostRoleStatus(id uint64, params PostRoleStatusParams, body PostRoleStatusJSONRequestBody)
	PostRoleUpdate(id uint64, params PostRoleUpdateParams, body PostRoleUpdateJSONRequestBody)
	PostUserRolesAdd(id uint64, params PostUserRolesAddParams, body PostUserRolesAddJSONRequestBody)
	PostUserRolesRemove(id uint64, params PostUserRolesRemoveParams, body PostUserRolesRemoveJSONRequestBody)
	PostUserRolesReplace(id uint64, params PostUserRolesReplaceParams, body PostUserRolesReplaceJSONRequestBody)
	PostUsersRolesAdd(params PostUsersRolesAddParams, body PostUsersRolesAddJSONRequestBody)
	PostUsersRolesRemove(params PostUsersRolesRemoveParams, body PostUsersRolesRemoveJSONRequestBody)
	PostUsersRolesReplace(params PostUsersRolesReplaceParams, body PostUsersRolesReplaceJSONRequestBody)
}
