package rbac

type createRoleRequest struct {
	Name        string  `json:"name"`
	Display     string  `json:"display"`
	Description *string `json:"description"`
}

type updateRoleRequest struct {
	Name        string  `json:"name"`
	Display     string  `json:"display"`
	Description *string `json:"description"`
}

type replaceRolePermissionsRequest struct {
	PermissionIDs *[]uint64 `json:"permission_ids"`
}

type replaceUserRolesRequest struct {
	RoleIDs *[]uint64 `json:"role_ids"`
}
