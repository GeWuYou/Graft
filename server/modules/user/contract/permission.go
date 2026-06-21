// Package contract defines stable user module contract values.
package contract

// PermissionCode identifies a stable user module permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// UserReadPermission identifies read access to user-management data.
	UserReadPermission PermissionCode = "user.read"

	// UserCreatePermission identifies create access to user-management data.
	UserCreatePermission PermissionCode = "user.create"

	// UserUpdatePermission identifies update access to user-management data.
	UserUpdatePermission PermissionCode = "user.update"

	// UserDisablePermission identifies disable/delete access to user-management data.
	UserDisablePermission PermissionCode = "user.disable"

	// UserSessionReadPermission identifies read access to refresh-session state.
	UserSessionReadPermission PermissionCode = "user.session.read"

	// UserSessionRevokePermission identifies revoke access to refresh-session state.
	UserSessionRevokePermission PermissionCode = "user.session.revoke"

	// UserRead is the canonical permission used by user module consumers.
	UserRead PermissionCode = UserReadPermission

	// UserCreate is the canonical permission used by user module consumers.
	UserCreate PermissionCode = UserCreatePermission

	// UserUpdate is the canonical permission used by user module consumers.
	UserUpdate PermissionCode = UserUpdatePermission

	// UserDisable is the canonical permission used by user module consumers.
	UserDisable PermissionCode = UserDisablePermission

	// UserSessionRead is the canonical permission used by user module consumers.
	UserSessionRead PermissionCode = UserSessionReadPermission

	// UserSessionRevoke is the canonical permission used by user module consumers.
	UserSessionRevoke PermissionCode = UserSessionRevokePermission
)
