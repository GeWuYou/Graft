package contract

// PermissionCode identifies a stable audit-plugin permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AuditReadPermission identifies read access to audit-log data.
	AuditReadPermission PermissionCode = "audit.read"

	// AuditRead is the canonical permission used by audit-plugin consumers.
	AuditRead PermissionCode = AuditReadPermission
)
