package contract

// PermissionCode identifies a stable monitor-plugin permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// ServerStatusReadPermission identifies read access to server status data.
	ServerStatusReadPermission PermissionCode = "monitor.server-status.read"

	// ServerStatusRead is the canonical permission used by monitor-plugin consumers.
	ServerStatusRead PermissionCode = ServerStatusReadPermission
)
