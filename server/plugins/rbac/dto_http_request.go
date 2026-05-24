package rbac

type replaceUserRolesRequest struct {
	RoleIDs *[]uint64 `json:"role_ids"`
}
