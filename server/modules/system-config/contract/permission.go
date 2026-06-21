package contract

// PermissionCode identifies a stable system configuration permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// SystemConfigReadPermission identifies read access to system configuration definitions and values.
	SystemConfigReadPermission PermissionCode = "system-config.read"
	// SystemConfigWritePermission identifies write access to user overrides.
	SystemConfigWritePermission PermissionCode = "system-config.write"
)
