package contract

// PermissionCode identifies a stable audit module permission contract.
type PermissionCode string

// String returns the wire-format permission code.
func (c PermissionCode) String() string {
	return string(c)
}

const (
	// AuditReadPermission identifies read access to audit-log data.
	AuditReadPermission PermissionCode = "audit.read"
	// AuditManagePermission identifies access to audit visibility policy management.
	AuditManagePermission PermissionCode = "audit.manage"

	// AuditRead is the canonical permission used by audit module consumers.
	AuditRead PermissionCode = AuditReadPermission
	// AuditManage is the canonical permission used by audit visibility policy management.
	AuditManage PermissionCode = AuditManagePermission
)
