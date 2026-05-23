package rbac

type roleListResponse struct {
	Items []roleListItem `json:"items"`
}

type roleListItem struct {
	ID              uint64  `json:"id"`
	Name            string  `json:"name"`
	Display         string  `json:"display"`
	Description     *string `json:"description,omitempty"`
	Builtin         bool    `json:"builtin"`
	UpdatedAt       string  `json:"updated_at"`
	PermissionCount int     `json:"permission_count"`
	UserCount       int     `json:"user_count"`
}

type rolePermissionBindingResponse struct {
	PermissionIDs []uint64 `json:"permission_ids"`
}

type userRoleBindingResponse struct {
	RoleIDs []uint64 `json:"role_ids"`
}

type permissionListResponse struct {
	Items []permissionListItem `json:"items"`
}

type permissionListItem struct {
	ID               uint64  `json:"id"`
	Code             string  `json:"code"`
	Display          string  `json:"display"`
	Description      *string `json:"description,omitempty"`
	Category         string  `json:"category"`
	CreatedAt        string  `json:"created_at"`
	UpdatedAt        string  `json:"updated_at"`
	RoleBindingCount int     `json:"role_binding_count"`
}
