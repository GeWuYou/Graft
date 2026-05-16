package contract

// PermissionCode identifies a stable rbac-plugin permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// RoleReadPermission identifies read access to role-management data.
	RoleReadPermission PermissionCode = "role.read"
	// PermissionReadPermission identifies read access to permission-management data.
	PermissionReadPermission PermissionCode = "permission.read"

	// RoleRead is the canonical permission used by rbac-plugin consumers.
	RoleRead PermissionCode = RoleReadPermission
	// PermissionRead is the canonical permission used by rbac-plugin consumers.
	PermissionRead PermissionCode = PermissionReadPermission
)
