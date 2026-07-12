package contract

// PermissionCode identifies a stable Security module permission.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string { return string(c) }

const (
	// OverviewReadPermission permits access to the aggregate security overview.
	OverviewReadPermission PermissionCode = "security.overview.read"
)
