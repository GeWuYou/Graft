// Package contract defines stable user-plugin contract values.
package contract

// PermissionCode identifies a stable user-plugin permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// UserReadPermission identifies read access to user-management data.
	UserReadPermission PermissionCode = "user.read"

	// UserSessionReadPermission identifies read access to refresh-session state.
	UserSessionReadPermission PermissionCode = "user.session.read"

	// UserSessionRevokePermission identifies revoke access to refresh-session state.
	UserSessionRevokePermission PermissionCode = "user.session.revoke"

	// UserRead is the canonical permission used by user-plugin consumers.
	UserRead PermissionCode = UserReadPermission

	// UserSessionRead is the canonical permission used by user-plugin consumers.
	UserSessionRead PermissionCode = UserSessionReadPermission

	// UserSessionRevoke is the canonical permission used by user-plugin consumers.
	UserSessionRevoke PermissionCode = UserSessionRevokePermission
)
